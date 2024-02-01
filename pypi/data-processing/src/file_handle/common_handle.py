# Copyright 2023 KubeAGI.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


import base64
import logging
import traceback

import ulid

from common import log_tag_const
from common.config import config
from database_operate import (data_process_detail_db_operate,
                              data_process_document_chunk_db_operate,
                              data_process_document_db_operate)
from embeddings.openai_embeddings import OpenAIEmbeddings
from kube import model_cr
from llm_api_service.qa_provider_open_ai import QAProviderOpenAI
from llm_api_service.qa_provider_zhi_pu_ai_online import \
    QAProviderZhiPuAIOnline
from service.data_process_qa_remove_duplicate import QARemoveDuplicate
from transform.text import clean_transform, privacy_transform
from utils import csv_utils, date_time_utils, file_utils, json_utils

logger = logging.getLogger(__name__)


def text_manipulate(
    all_document_for_process, file_name, support_type, conn_pool, create_user
):
    """Manipulate the text content.

    all_document_for_process: document info
    file_name: file name;
    support_type: support type;
    conn_pool: database connection pool;
    create_user: creator;
    """

    logger.debug(f"{log_tag_const.COMMON_HANDLE} Start to manipulate the text")

    try:
        support_type_map = _convert_support_type_to_map(support_type)
        document_chunk_size = len(all_document_for_process)

        # 更新文件状态为开始
        task_id = all_document_for_process[0].get("task_id")
        document_id = all_document_for_process[0].get("document_id")
        _update_document_status_and_start_time(
            id=all_document_for_process[0].get("document_id"),
            chunk_size=document_chunk_size,
            conn_pool=conn_pool,
        )

        text_process_success_num = 0
        for document in all_document_for_process:
            document_chunk_id = document.get("id")
            # Clean the data such as removing invisible characters.
            clean_result = _data_clean(
                support_type_map=support_type_map,
                file_name=file_name,
                data=document.get("content"),
                conn_pool=conn_pool,
                task_id=task_id,
                document_id=document_id,
                document_chunk_id=document_chunk_id,
                create_user=create_user,
            )

            if clean_result["status"] == 200:
                content = clean_result["data"]

            # Remove the privacy info such as removing email.
            clean_result = _remove_privacy_info(
                support_type_map=support_type_map,
                file_name=file_name,
                data=document.get("content"),
                conn_pool=conn_pool,
                task_id=task_id,
                document_id=document_id,
                document_chunk_id=document_chunk_id,
                create_user=create_user,
            )

            if clean_result["status"] == 200:
                content = clean_result["data"]

            if support_type_map.get("qa_split"):
                logger.debug(f"{log_tag_const.QA_SPLIT} Start to split QA.")
                text_process_success_num += 1

                qa_response = _qa_split(
                    support_type_map=support_type_map,
                    task_id=task_id,
                    document_chunk_size=document_chunk_size,
                    document_chunk_id=document_chunk_id,
                    file_name=file_name,
                    content=content,
                    document_id=document_id,
                    text_process_success_num=text_process_success_num,
                    conn_pool=conn_pool,
                    create_user=create_user,
                )

                if qa_response.get("status") != 200:
                    return qa_response

        # 文件处理成功，更新data_process_task_document中的文件状态
        _updata_document_status_and_end_time(
            id=document_id, status="success", conn_pool=conn_pool
        )

        if support_type_map.get("qa_split"):
            # 是否选择了QA拆分
            qa_list_dict = support_type_map.get("qa_split")
            remove_duplicate_config = qa_list_dict.get("remove_duplicate_config")
            if remove_duplicate_config:
                # 进行了QA去重配置
                logger.debug(f"{log_tag_const.QA_SPLIT} Start to QA remove duplicate.")
                remove_duplicate_response = _remove_duplicate(
                    document_id=document_id,
                    remove_duplicate_config=remove_duplicate_config,
                    conn_pool=conn_pool,
                    create_user=create_user
                )

                if remove_duplicate_response.get("status") != 200:
                    return remove_duplicate_response

            # 通过documentId查询生成的所有QA数据
            qa_list = data_process_detail_db_operate.query_question_answer_clean_list(
                document_id=document_id, pool=conn_pool
            )

            qa_data_dict = [["q", "a", "file_name", "page_number", "chunk_content"]]
            for item in qa_list.get("data"):
                meta_info = item.get("meta_info")
                if meta_info:
                    meta_json = json_utils.loads(meta_info)
                    meta_source = meta_json.get("source")
                else:
                    meta_source = item.get("file_name")

                qa_data_dict.append(
                    [
                        item.get("question"),
                        item.get("answer"),
                        meta_source,
                        item.get("page_number"),
                        item.get("content"),
                    ]
                )

            # Save the csv file.
            file_name_without_extension = file_utils.get_file_name_without_extension(
                file_name
            )
            file_name_csv = file_name_without_extension + ".csv"
            csv_utils.save_csv(
                file_name=file_name_csv, phase_value="final", data=qa_data_dict
            )

            logger.debug(f"{log_tag_const.COMMON_HANDLE} Finish manipulating the text")
            return {
                "status": 200,
                "message": "",
                "data": {
                    "object_name": file_name_csv,
                    "object_count": len(qa_list.get("data")),
                },
            }

        return {"status": 200, "message": "", "data": ""}
    except Exception as ex:
        logger.error(
            "".join(
                [
                    f"{log_tag_const.COMMON_HANDLE} There is an error when manipulate ",
                    f"the text in common handler. \n{traceback.format_exc()}",
                ]
            )
        )
        logger.debug(f"{log_tag_const.COMMON_HANDLE} Finish manipulating the text")
        return {"status": 400, "message": str(ex), "data": traceback.format_exc()}


def _data_clean(
    support_type_map,
    data,
    task_id,
    document_id,
    document_chunk_id,
    file_name,
    create_user,
    conn_pool,
):
    """Clean the data.

    support_type_map: example
        {
            "qa_split": {
                "type": "qa_split",
                "name": "xx",
                "namespace": "xx"
            },
            "remove_invisible_characters": {
                "type": "remove_invisible_characters"
            },
            "space_standardization": {
                "type": "space_standardization"
            },
            "remove_email": {
                "type": "remove_email"
            }
        }
    data: data;
    file_name: file name;
    conn_pool: database connection pool;
    task_id: data process task id;
    """
    # remove invisible characters
    if support_type_map.get("remove_invisible_characters"):
        result = clean_transform.remove_invisible_characters(text=data)
        if result["status"] == 200:
            clean_data = result["data"]["clean_data"]
            if len(clean_data) > 0:
                for item in clean_data:
                    # 避免重试的时候，新增重复性数据
                    delete_transform_item = {
                        "task_id": task_id,
                        "document_id": document_id,
                        "document_chunk_id": document_chunk_id,
                    }
                    data_process_detail_db_operate.delete_transform_by_document_chunk(
                        delete_transform_item, pool=conn_pool
                    )

                    task_detail_item = {
                        "id": ulid.ulid(),
                        "task_id": task_id,
                        "document_id": document_id,
                        "document_chunk_id": document_chunk_id,
                        "file_name": file_name,
                        "transform_type": "remove_invisible_characters",
                        "pre_content": item["pre_content"],
                        "post_content": item["post_content"],
                        "status": "success",
                        "create_user": create_user,
                    }
                    data_process_detail_db_operate.insert_transform_info(
                        task_detail_item, pool=conn_pool
                    )
            data = result["data"]["text"]
        else:
            task_detail_item = {
                "id": ulid.ulid(),
                "task_id": task_id,
                "document_id": document_id,
                "document_chunk_id": document_chunk_id,
                "file_name": file_name,
                "transform_type": "remove_invisible_characters",
                "status": "fail",
                "error_message": result.get("message"),
                "create_user": create_user,
            }
            data_process_detail_db_operate.insert_transform_info(
                task_detail_item, pool=conn_pool
            )

    # process for space standardization
    if support_type_map.get("space_standardization"):
        result = clean_transform.space_standardization(text=data)
        if result["status"] == 200:
            clean_data = result["data"]["clean_data"]
            if len(clean_data) > 0:
                for item in clean_data:
                    # 避免重试的时候，新增重复性数据
                    delete_transform_item = {
                        "task_id": task_id,
                        "document_id": document_id,
                        "document_chunk_id": document_chunk_id,
                    }
                    data_process_detail_db_operate.delete_transform_by_document_chunk(
                        delete_transform_item, pool=conn_pool
                    )

                    task_detail_item = {
                        "id": ulid.ulid(),
                        "task_id": task_id,
                        "document_id": document_id,
                        "document_chunk_id": document_chunk_id,
                        "file_name": file_name,
                        "transform_type": "space_standardization",
                        "pre_content": item["pre_content"],
                        "post_content": item["post_content"],
                        "status": "success",
                        "create_user": create_user,
                    }
                    data_process_detail_db_operate.insert_transform_info(
                        task_detail_item, pool=conn_pool
                    )
            data = result["data"]["text"]
        else:
            task_detail_item = {
                "id": ulid.ulid(),
                "task_id": task_id,
                "document_id": document_id,
                "document_chunk_id": document_chunk_id,
                "file_name": file_name,
                "transform_type": "space_standardization",
                "status": "fail",
                "error_message": result.get("message"),
                "create_user": create_user,
            }
            data_process_detail_db_operate.insert_transform_info(
                task_detail_item, pool=conn_pool
            )

    # process for remove garbled text
    if support_type_map.get("remove_garbled_text"):
        result = clean_transform.remove_garbled_text(text=data)
        if result["status"] == 200:
            if result["data"]["found"] > 0:
                # 避免重试的时候，新增重复性数据
                delete_transform_item = {
                    "task_id": task_id,
                    "document_id": document_id,
                    "document_chunk_id": document_chunk_id,
                }
                data_process_detail_db_operate.delete_transform_by_document_chunk(
                    delete_transform_item, pool=conn_pool
                )

                task_detail_item = {
                    "id": ulid.ulid(),
                    "task_id": task_id,
                    "document_id": document_id,
                    "document_chunk_id": document_chunk_id,
                    "file_name": file_name,
                    "transform_type": "remove_garbled_text",
                    "pre_content": data,
                    "post_content": result["data"]["text"],
                    "status": "success",
                    "create_user": create_user,
                }
                data_process_detail_db_operate.insert_transform_info(
                    task_detail_item, pool=conn_pool
                )
            data = result["data"]["text"]
        else:
            task_detail_item = {
                "id": ulid.ulid(),
                "task_id": task_id,
                "document_id": document_id,
                "document_chunk_id": document_chunk_id,
                "file_name": file_name,
                "transform_type": "remove_garbled_text",
                "status": "fail",
                "error_message": result.get("message"),
                "create_user": create_user,
            }
            data_process_detail_db_operate.insert_transform_info(
                task_detail_item, pool=conn_pool
            )

    # process for Traditional Chinese to Simplified Chinese
    if support_type_map.get("traditional_to_simplified"):
        result = clean_transform.traditional_to_simplified(text=data)
        if result["status"] == 200:
            if result["data"]["found"] > 0:
                # 避免重试的时候，新增重复性数据
                delete_transform_item = {
                    "task_id": task_id,
                    "document_id": document_id,
                    "document_chunk_id": document_chunk_id,
                }
                data_process_detail_db_operate.delete_transform_by_document_chunk(
                    delete_transform_item, pool=conn_pool
                )

                task_detail_item = {
                    "id": ulid.ulid(),
                    "task_id": task_id,
                    "document_id": document_id,
                    "document_chunk_id": document_chunk_id,
                    "file_name": file_name,
                    "transform_type": "traditional_to_simplified",
                    "pre_content": data,
                    "post_content": result["data"]["text"],
                    "status": "success",
                    "create_user": create_user,
                }
                data_process_detail_db_operate.insert_transform_info(
                    task_detail_item, pool=conn_pool
                )
            data = result["data"]["text"]
        else:
            task_detail_item = {
                "id": ulid.ulid(),
                "task_id": task_id,
                "document_id": document_id,
                "document_chunk_id": document_chunk_id,
                "file_name": file_name,
                "transform_type": "traditional_to_simplified",
                "status": "fail",
                "error_message": result.get("message"),
                "create_user": create_user,
            }
            data_process_detail_db_operate.insert_transform_info(
                task_detail_item, pool=conn_pool
            )

    # process for clean html code in text samples
    if support_type_map.get("remove_html_tag"):
        result = clean_transform.remove_html_tag(text=data)
        if result["status"] == 200:
            if result["data"]["found"] > 0:
                # 避免重试的时候，新增重复性数据
                delete_transform_item = {
                    "task_id": task_id,
                    "document_id": document_id,
                    "document_chunk_id": document_chunk_id,
                }
                data_process_detail_db_operate.delete_transform_by_document_chunk(
                    delete_transform_item, pool=conn_pool
                )

                task_detail_item = {
                    "id": ulid.ulid(),
                    "task_id": task_id,
                    "document_id": document_id,
                    "document_chunk_id": document_chunk_id,
                    "file_name": file_name,
                    "transform_type": "remove_html_tag",
                    "pre_content": data,
                    "post_content": result["data"]["text"],
                    "status": "success",
                    "create_user": create_user,
                }
                data_process_detail_db_operate.insert_transform_info(
                    task_detail_item, pool=conn_pool
                )
            data = result["data"]["text"]
        else:
            task_detail_item = {
                "id": ulid.ulid(),
                "task_id": task_id,
                "document_id": document_id,
                "document_chunk_id": document_chunk_id,
                "file_name": file_name,
                "transform_type": "remove_html_tag",
                "status": "fail",
                "error_message": result.get("message"),
                "create_user": create_user,
            }
            data_process_detail_db_operate.insert_transform_info(
                task_detail_item, pool=conn_pool
            )

    # process for remove emojis
    if support_type_map.get("remove_emojis"):
        result = clean_transform.remove_emojis(text=data)
        if result["status"] == 200:
            clean_data = result["data"]["clean_data"]
            if len(clean_data) > 0:
                for item in clean_data:
                    # 避免重试的时候，新增重复性数据
                    delete_transform_item = {
                        "task_id": task_id,
                        "document_id": document_id,
                        "document_chunk_id": document_chunk_id,
                    }
                    data_process_detail_db_operate.delete_transform_by_document_chunk(
                        delete_transform_item, pool=conn_pool
                    )

                    task_detail_item = {
                        "id": ulid.ulid(),
                        "task_id": task_id,
                        "document_id": document_id,
                        "document_chunk_id": document_chunk_id,
                        "file_name": file_name,
                        "transform_type": "remove_emojis",
                        "pre_content": item["pre_content"],
                        "post_content": item["post_content"],
                        "status": "success",
                        "create_user": create_user,
                    }
                    data_process_detail_db_operate.insert_transform_info(
                        task_detail_item, pool=conn_pool
                    )
            data = result["data"]["text"]
        else:
            task_detail_item = {
                "id": ulid.ulid(),
                "task_id": task_id,
                "document_id": document_id,
                "document_chunk_id": document_chunk_id,
                "file_name": file_name,
                "transform_type": "remove_emojis",
                "status": "fail",
                "error_message": result.get("message"),
                "create_user": create_user,
            }
            data_process_detail_db_operate.insert_transform_info(
                task_detail_item, pool=conn_pool
            )

    return {"status": 200, "message": "", "data": data}


def _remove_privacy_info(
    support_type_map,
    data,
    task_id,
    document_id,
    document_chunk_id,
    file_name,
    create_user,
    conn_pool,
):
    """ "Remove the privacy info such as removing email.

    support_type_map: example
        {
            "qa_split": {
                "type": "qa_split",
                "name": "xx",
                "namespace": "xx"
            },
            "remove_invisible_characters": {
                "type": "remove_invisible_characters"
            },
            "space_standardization": {
                "type": "space_standardization"
            },
            "remove_email": {
                "type": "remove_email"
            }
        }
    data: data;
    file_name: file name;
    conn_pool: database connection pool;
    task_id: data process task id;
    """
    # remove email
    if support_type_map.get("remove_email"):
        result = privacy_transform.remove_email(text=data)
        if result["status"] == 200:
            clean_data = result["data"]["clean_data"]
            if len(clean_data) > 0:
                for item in clean_data:
                    # 避免重试的时候，新增重复性数据
                    delete_transform_item = {
                        "task_id": task_id,
                        "document_id": document_id,
                        "document_chunk_id": document_chunk_id,
                    }
                    data_process_detail_db_operate.delete_transform_by_document_chunk(
                        delete_transform_item, pool=conn_pool
                    )

                    task_detail_item = {
                        "id": ulid.ulid(),
                        "task_id": task_id,
                        "document_id": document_id,
                        "document_chunk_id": document_chunk_id,
                        "file_name": file_name,
                        "transform_type": "remove_email",
                        "pre_content": item["pre_content"],
                        "post_content": item["post_content"],
                        "status": "success",
                        "create_user": create_user,
                    }
                    data_process_detail_db_operate.insert_transform_info(
                        task_detail_item, pool=conn_pool
                    )
            data = result["data"]["text"]
        else:
            task_detail_item = {
                "id": ulid.ulid(),
                "task_id": task_id,
                "document_id": document_id,
                "document_chunk_id": document_chunk_id,
                "file_name": file_name,
                "transform_type": "remove_email",
                "status": "fail",
                "error_message": result.get("message"),
                "create_user": create_user,
            }
            data_process_detail_db_operate.insert_transform_info(
                task_detail_item, pool=conn_pool
            )

    # remove ip addresses
    if support_type_map.get("remove_ip_address"):
        result = privacy_transform.remove_ip_address(text=data)
        if result["status"] == 200:
            clean_data = result["data"]["clean_data"]
            if len(clean_data) > 0:
                for item in clean_data:
                    # 避免重试的时候，新增重复性数据
                    delete_transform_item = {
                        "task_id": task_id,
                        "document_id": document_id,
                        "document_chunk_id": document_chunk_id,
                    }
                    data_process_detail_db_operate.delete_transform_by_document_chunk(
                        delete_transform_item, pool=conn_pool
                    )

                    task_detail_item = {
                        "id": ulid.ulid(),
                        "task_id": task_id,
                        "document_id": document_id,
                        "document_chunk_id": document_chunk_id,
                        "file_name": file_name,
                        "transform_type": "remove_ip_address",
                        "pre_content": item["pre_content"],
                        "post_content": item["post_content"],
                        "status": "success",
                        "create_user": create_user,
                    }
                    data_process_detail_db_operate.insert_transform_info(
                        task_detail_item, pool=conn_pool
                    )
            data = result["data"]["text"]
        else:
            task_detail_item = {
                "id": ulid.ulid(),
                "task_id": task_id,
                "document_id": document_id,
                "document_chunk_id": document_chunk_id,
                "file_name": file_name,
                "transform_type": "remove_ip_address",
                "status": "fail",
                "error_message": result.get("message"),
                "create_user": create_user,
            }
            data_process_detail_db_operate.insert_transform_info(
                task_detail_item, pool=conn_pool
            )

    # remove number
    if support_type_map.get("remove_number"):
        # remove phone
        result = privacy_transform.remove_phone(text=data)
        if result["status"] == 200:
            clean_data = result["data"]["clean_data"]
            if len(clean_data) > 0:
                for item in clean_data:
                    # 避免重试的时候，新增重复性数据
                    delete_transform_item = {
                        "task_id": task_id,
                        "document_id": document_id,
                        "document_chunk_id": document_chunk_id,
                    }
                    data_process_detail_db_operate.delete_transform_by_document_chunk(
                        delete_transform_item, pool=conn_pool
                    )

                    task_detail_item = {
                        "id": ulid.ulid(),
                        "task_id": task_id,
                        "document_id": document_id,
                        "document_chunk_id": document_chunk_id,
                        "file_name": file_name,
                        "transform_type": "remove_number",
                        "pre_content": item["pre_content"],
                        "post_content": item["post_content"],
                        "status": "success",
                        "create_user": create_user,
                    }
                    data_process_detail_db_operate.insert_transform_info(
                        task_detail_item, pool=conn_pool
                    )
            data = result["data"]["text"]
        else:
            task_detail_item = {
                "id": ulid.ulid(),
                "task_id": task_id,
                "document_id": document_id,
                "document_chunk_id": document_chunk_id,
                "file_name": file_name,
                "transform_type": "remove_number",
                "status": "fail",
                "error_message": result.get("message"),
                "create_user": create_user,
            }
            data_process_detail_db_operate.insert_transform_info(
                task_detail_item, pool=conn_pool
            )

        # remove id card
        result = privacy_transform.remove_id_card(text=data)
        if result["status"] == 200:
            clean_data = result["data"]["clean_data"]
            if len(clean_data) > 0:
                for item in clean_data:
                    # 避免重试的时候，新增重复性数据
                    delete_transform_item = {
                        "task_id": task_id,
                        "document_id": document_id,
                        "document_chunk_id": document_chunk_id,
                    }
                    data_process_detail_db_operate.delete_transform_by_document_chunk(
                        delete_transform_item, pool=conn_pool
                    )

                    task_detail_item = {
                        "id": ulid.ulid(),
                        "task_id": task_id,
                        "document_id": document_id,
                        "document_chunk_id": document_chunk_id,
                        "file_name": file_name,
                        "transform_type": "remove_number",
                        "pre_content": item["pre_content"],
                        "post_content": item["post_content"],
                        "status": "success",
                        "create_user": create_user,
                    }
                    data_process_detail_db_operate.insert_transform_info(
                        task_detail_item, pool=conn_pool
                    )
            data = result["data"]["text"]
        else:
            task_detail_item = {
                "id": ulid.ulid(),
                "task_id": task_id,
                "document_id": document_id,
                "document_chunk_id": document_chunk_id,
                "file_name": file_name,
                "transform_type": "remove_number",
                "status": "fail",
                "error_message": result.get("message"),
                "create_user": create_user,
            }
            data_process_detail_db_operate.insert_transform_info(
                task_detail_item, pool=conn_pool
            )

        # remove weixin
        result = privacy_transform.remove_weixin(text=data)
        if result["status"] == 200:
            clean_data = result["data"]["clean_data"]
            if len(clean_data) > 0:
                for item in clean_data:
                    # 避免重试的时候，新增重复性数据
                    delete_transform_item = {
                        "task_id": task_id,
                        "document_id": document_id,
                        "document_chunk_id": document_chunk_id,
                    }
                    data_process_detail_db_operate.delete_transform_by_document_chunk(
                        delete_transform_item, pool=conn_pool
                    )

                    task_detail_item = {
                        "id": ulid.ulid(),
                        "task_id": task_id,
                        "document_id": document_id,
                        "document_chunk_id": document_chunk_id,
                        "file_name": file_name,
                        "transform_type": "remove_number",
                        "pre_content": item["pre_content"],
                        "post_content": item["post_content"],
                        "status": "success",
                        "create_user": create_user,
                    }
                    data_process_detail_db_operate.insert_transform_info(
                        task_detail_item, pool=conn_pool
                    )
            data = result["data"]["text"]
        else:
            task_detail_item = {
                "id": ulid.ulid(),
                "task_id": task_id,
                "document_id": document_id,
                "document_chunk_id": document_chunk_id,
                "file_name": file_name,
                "transform_type": "remove_number",
                "status": "fail",
                "error_message": result.get("message"),
                "create_user": create_user,
            }
            data_process_detail_db_operate.insert_transform_info(
                task_detail_item, pool=conn_pool
            )

        # remove bank card
        result = privacy_transform.remove_bank_card(text=data)
        if result["status"] == 200:
            clean_data = result["data"]["clean_data"]
            if len(clean_data) > 0:
                for item in clean_data:
                    # 避免重试的时候，新增重复性数据
                    delete_transform_item = {
                        "task_id": task_id,
                        "document_id": document_id,
                        "document_chunk_id": document_chunk_id,
                    }
                    data_process_detail_db_operate.delete_transform_by_document_chunk(
                        delete_transform_item, pool=conn_pool
                    )

                    task_detail_item = {
                        "id": ulid.ulid(),
                        "task_id": task_id,
                        "document_id": document_id,
                        "document_chunk_id": document_chunk_id,
                        "file_name": file_name,
                        "transform_type": "remove_number",
                        "pre_content": item["pre_content"],
                        "post_content": item["post_content"],
                        "status": "success",
                        "create_user": create_user,
                    }
                    data_process_detail_db_operate.insert_transform_info(
                        task_detail_item, pool=conn_pool
                    )
            data = result["data"]["text"]
        else:
            task_detail_item = {
                "id": ulid.ulid(),
                "task_id": task_id,
                "document_id": document_id,
                "document_chunk_id": document_chunk_id,
                "file_name": file_name,
                "transform_type": "remove_number",
                "status": "fail",
                "error_message": result.get("message"),
                "create_user": create_user,
            }
            data_process_detail_db_operate.insert_transform_info(
                task_detail_item, pool=conn_pool
            )

    return {"status": 200, "message": "", "data": data}


def _qa_split(
    support_type_map,
    task_id,
    document_chunk_size,
    document_chunk_id,
    file_name,
    content,
    document_id,
    text_process_success_num,
    conn_pool,
    create_user,
):
    qa_list_dict = support_type_map.get("qa_split")
    llm_config = qa_list_dict.get("llm_config")

    # 更新chunk状态为开始
    _update_document_chunk_status_and_start_time(
        id=document_chunk_id, update_user=create_user, conn_pool=conn_pool
    )

    qa_response = _generate_qa_list(content=content, llm_config=llm_config)

    if qa_response.get("status") != 200:
        # 处理失败
        # 更新data_process_task_document_chunk中的状态
        _updata_document_chunk_status_and_end_time(
            id=document_chunk_id,
            update_user=create_user,
            status="fail",
            conn_pool=conn_pool,
        )

        # 更新data_process_task_document中的文件状态
        _updata_document_status_and_end_time(
            id=document_id, status="fail", conn_pool=conn_pool
        )
    else:
        # 将QA数据存入表中
        qa_data = qa_response.get("data")
        for _, item in enumerate(qa_data):
            qa_insert_item = {
                "id": ulid.ulid(),
                "task_id": task_id,
                "document_id": document_id,
                "document_chunk_id": document_chunk_id,
                "file_name": file_name,
                "question": item[0],
                "answer": item[1],
                "create_user": create_user,
            }

            data_process_detail_db_operate.insert_question_answer_info(
                qa_insert_item, pool=conn_pool
            )

        # 更新data_process_task_document_chunk中的状态
        _updata_document_chunk_status_and_end_time(
            id=document_chunk_id,
            update_user=create_user,
            status="success",
            conn_pool=conn_pool,
        )

        # 更新文件处理进度
        progress = int(text_process_success_num / document_chunk_size * 100)
        _updata_document_progress(
            id=document_id,
            progress=progress,
            update_user=create_user,
            conn_pool=conn_pool,
        )

    return qa_response


def _generate_qa_list(content, llm_config):
    """Generate the Question and Answer list.

    content: the text used to generate QA;
    llm_config: llms config info;
    """
    name = llm_config.get("name")
    namespace = llm_config.get("namespace")
    model = llm_config.get("model")
    temperature = llm_config.get("temperature")
    prompt_template = llm_config.get("prompt_template")
    top_p = llm_config.get("top_p")
    max_tokens = llm_config.get("max_tokens")

    # llms cr 中模型相关信息
    llm_spec_info = model_cr.get_spec_for_llms_k8s_cr(name=name, namespace=namespace)

    # Generate the QA list.
    qa_list = []
    if llm_spec_info.get("data").get("provider").get("worker"):
        # get base url for configmap
        base_url = model_cr.get_worker_base_url_k8s_configmap(
            name=config.k8s_default_config, namespace=config.k8s_pod_namespace
        )
        logger.debug(
            "".join(
                [
                    f"worker llm \n",
                    f"name: {name}\n",
                    f"namespace: {namespace}\n",
                    f"model: {model}\n",
                    f"base_url: {base_url}\n",
                ]
            )
        )

        # generate QA list
        qa_provider = QAProviderOpenAI(
            api_key="fake",
            base_url=base_url,
            model=model,
            temperature=temperature,
            max_tokens=max_tokens,
        )

        data = qa_provider.generate_qa_list(
            text=content, prompt_template=prompt_template
        )

        if data.get("status") != 200:
            # 文件处理失败
            return data

        qa_list.extend(data.get("data"))
    else:
        endpoint = llm_spec_info.get("data").get("provider").get("endpoint")
        base_url = endpoint.get("url")
        secret_name = endpoint.get("authSecret").get("name")

        # get api key for secret
        secret_info = model_cr.get_secret_info(name=secret_name, namespace=namespace)
        api_key = secret_info.get("apiKey")
        llm_type = llm_spec_info.get("data").get("type")

        logger.debug(
            "".join(
                [
                    f"3rd_party llm \n",
                    f"name: {name}\n",
                    f"namespace: {namespace}\n",
                    f"model: {model}\n",
                    f"llm_type: {llm_type}\n",
                ]
            )
        )

        if llm_type == "zhipuai":
            zhipuai_api_key = base64.b64decode(api_key).decode("utf-8")
            qa_provider = QAProviderZhiPuAIOnline(api_key=zhipuai_api_key)

            # generate QA list
            data = qa_provider.generate_qa_list(
                text=content,
                model=model,
                prompt_template=prompt_template,
                top_p=top_p,
                temperature=temperature,
            )
            if data.get("status") != 200:
                return data

            qa_list.extend(data.get("data"))

        elif llm_type == "openai":
            # generate QA list
            qa_provider = QAProviderOpenAI(
                api_key="fake",
                base_url=base_url,
                model=model,
                temperature=temperature,
                max_tokens=max_tokens,
            )

            data = qa_provider.generate_qa_list(
                text=content, prompt_template=prompt_template
            )

            if data.get("status") != 200:
                return data

            qa_list.extend(data.get("data"))
        else:
            return {"status": 1000, "message": "暂时不支持该类型的模型", "data": ""}

    return {"status": 200, "message": "", "data": qa_list}


def _convert_support_type_to_map(supprt_type):
    """Convert support type to map.

    support_type: support type list
    example
    [
        {
            "type": "qa_split"
        },
        {
            "type": "remove_invisible_characters"
        },
        {
            "type": "space_standardization"
        },
        {
            "type": "remove_email"
        }
    ]
    """
    result = {}
    for item in supprt_type:
        result[item["type"]] = item

    return result


def _update_document_status_and_start_time(id, chunk_size, conn_pool):
    try:
        now = date_time_utils.now_str()
        document_update_item = {
            "id": id,
            "status": "doing",
            "start_time": now,
            "chunk_size": chunk_size,
        }
        data_process_document_db_operate.update_document_status_and_start_time(
            document_update_item, pool=conn_pool
        )

        return {"status": 200, "message": "", "data": ""}
    except Exception as ex:
        logger.error(
            "".join(
                [
                    f"{log_tag_const.COMMON_HANDLE} update document status ",
                    f"\n{traceback.format_exc()}",
                ]
            )
        )
        return {"status": 1000, "message": str(ex), "data": traceback.format_exc()}


def _updata_document_status_and_end_time(id, status, conn_pool):
    try:
        now = date_time_utils.now_str()
        document_update_item = {"id": id, "status": status, "end_time": now}
        data_process_document_db_operate.update_document_status_and_end_time(
            document_update_item, pool=conn_pool
        )

        return {"status": 200, "message": "", "data": ""}
    except Exception as ex:
        logger.error(
            "".join(
                [
                    f"{log_tag_const.COMMON_HANDLE} update document status ",
                    f"\n{traceback.format_exc()}",
                ]
            )
        )
        return {"status": 1000, "message": str(ex), "data": traceback.format_exc()}


def _updata_document_progress(id, progress, update_user, conn_pool):
    try:
        document_update_item = {
            "id": id,
            "update_user": update_user,
            "progress": progress,
        }
        data_process_document_db_operate.update_document_progress(
            document_update_item, pool=conn_pool
        )

        return {"status": 200, "message": "", "data": ""}
    except Exception as ex:
        logger.error(
            "".join(
                [
                    f"{log_tag_const.COMMON_HANDLE} update document progress ",
                    f"\n{traceback.format_exc()}",
                ]
            )
        )
        return {"status": 1000, "message": str(ex), "data": traceback.format_exc()}


def _update_document_chunk_status_and_start_time(id, update_user, conn_pool):
    try:
        now = date_time_utils.now_str()
        document_chunk_update_item = {
            "id": id,
            "status": "doing",
            "update_user": update_user,
            "start_time": now,
        }
        data_process_document_chunk_db_operate.update_document_chunk_status_and_start_time(
            document_chunk_update_item, pool=conn_pool
        )

        return {"status": 200, "message": "", "data": ""}
    except Exception as ex:
        logger.error(
            "".join(
                [
                    f"{log_tag_const.COMMON_HANDLE} update chunk document status ",
                    f"\n{traceback.format_exc()}",
                ]
            )
        )
        return {"status": 1000, "message": str(ex), "data": traceback.format_exc()}


def _updata_document_chunk_status_and_end_time(id, status, update_user, conn_pool):
    try:
        now = date_time_utils.now_str()
        document_chunk_update_item = {
            "id": id,
            "status": status,
            "update_user": update_user,
            "end_time": now,
        }
        data_process_document_chunk_db_operate.update_document_chunk_status_and_end_time(
            document_chunk_update_item, pool=conn_pool
        )

        return {"status": 200, "message": "", "data": ""}
    except Exception as ex:
        logger.error(
            "".join(
                [
                    f"{log_tag_const.COMMON_HANDLE} update document status ",
                    f"\n{traceback.format_exc()}",
                ]
            )
        )
        return {"status": 1000, "message": str(ex), "data": traceback.format_exc()}


def _remove_duplicate(document_id, remove_duplicate_config, conn_pool, create_user):
    # 通过documentId查询生成的所有QA数据
    qa_list = data_process_detail_db_operate.query_question_answer_list(
        document_id=document_id, pool=conn_pool
    )

    remove_duplicate_res = _qa_remove_duplicate(
        qa_list=qa_list.get("data"),
        remove_duplicate_config=remove_duplicate_config,
        conn_pool=conn_pool,
    )
    if remove_duplicate_res.get("status") != 200:
        # 更新data_process_task_document中的文件状态
        _updata_document_status_and_end_time(
            id=document_id, status="fail", conn_pool=conn_pool
        )
        return remove_duplicate_res

    # 将QA去重的数据存入question_answer_clean表中
    qa_data = remove_duplicate_res.get("data")
    for _, item in enumerate(qa_data):
        duplicated_flag = 1
        if item.get("duplicated_flag") is not None:
            duplicated_flag = item.get("duplicated_flag")
        qa_insert_item = {
                "id": item.get("id"),
                "task_id": item.get("task_id"),
                "document_id": item.get("document_id"),
                "document_chunk_id": item.get("document_chunk_id"),
                "file_name": item.get("file_name"),
                "question": item.get("question"),
                "answer": item.get("answer"),
                "question_score": item.get("question_distance"),
                "answer_score": item.get("answer_distance"),
                "duplicated_flag": duplicated_flag,
                "compare_with_id": item.get("compare_with_id"),
                "create_user": create_user,
            }
        data_process_detail_db_operate.insert_question_answer_clean_info(
            qa_insert_item, pool=conn_pool
        )
    return remove_duplicate_res

def _qa_remove_duplicate(qa_list, remove_duplicate_config, conn_pool):
    name = remove_duplicate_config.get("embedding_name")
    namespace = remove_duplicate_config.get("embedding_namespace")
    model = remove_duplicate_config.get("embedding_model")
    provider = remove_duplicate_config.get("embedding_provider")
    similarity = float(remove_duplicate_config.get("similarity"))

    # llms cr 中模型相关信息
    llm_spec_info = model_cr.get_spec_for_embedding_k8s_cr(name=name, namespace=namespace)

    if provider == "worker":
        # get base url for configmap
        base_url = model_cr.get_worker_base_url_k8s_configmap(
            name=config.k8s_default_config, namespace=config.k8s_pod_namespace
        )
        logger.debug(
            "".join(
                [
                    f"worker embedding \n",
                    f"name: {name}\n",
                    f"namespace: {namespace}\n",
                    f"model: {model}\n",
                    f"base_url: {base_url}\n",
                ]
            )
        )

        qa_embeddings = OpenAIEmbeddings(
            api_key="fake",
            base_url=base_url,
            model=model,
        )

        remove_duplicate_loader = QARemoveDuplicate(embeddings=qa_embeddings, pool=conn_pool)
        return remove_duplicate_loader.qa_remove_duplicate(qa_list, similarity)
    else:
        endpoint = llm_spec_info.get("data").get("provider").get("endpoint")
        base_url = endpoint.get("url")
        llm_type = llm_spec_info.get("data").get("type")

        logger.debug(
            "".join(
                [
                    f"3rd_party embedding \n",
                    f"name: {name}\n",
                    f"namespace: {namespace}\n",
                    f"model: {model}\n",
                    f"llm_type: {llm_type}\n",
                ]
            )
        )

        if llm_type == "openai":
            qa_embeddings = OpenAIEmbeddings(
                api_key="fake",
                base_url=base_url,
                model=model,
            )

            remove_duplicate_loader = QARemoveDuplicate(embeddings=qa_embeddings, pool=conn_pool)
            return remove_duplicate_loader.qa_remove_duplicate(qa_list, similarity)
        else:
            return {"status": 1000, "message": f"暂时不支持{llm_type}类型的向量化模型模型", "data": ""}
