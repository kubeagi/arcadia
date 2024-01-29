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


import asyncio
import logging
import traceback

import ulid

from common import log_tag_const
from data_store_process import minio_store_process
from database_operate import (data_process_db_operate,
                              data_process_detail_db_operate,
                              data_process_detail_preview_db_operate,
                              data_process_document_chunk_db_operate,
                              data_process_document_db_operate,
                              data_process_log_db_operate,
                              data_process_stage_log_db_operate)
from parallel import thread_parallel

logger = logging.getLogger(__name__)


def list_by_page(req_json, pool):
    """Get the list data for data processing by page"""
    return data_process_db_operate.list_by_page(req_json, pool=pool)


def list_by_count(req_json, pool):
    """Get count for the list data processing with page"""
    return data_process_db_operate.list_by_count(req_json, pool=pool)


def add(req_json, pool):
    """Add a new data process task.

    req_json is a dictionary object. for example:
    {
        "name": "小T_test_0201",
        "file_type": "text",
        "pre_data_set_name": "dataset1",
        "pre_data_set_version": "v2",
        "post_data_set_name": "dataset1",
        "post_data_set_version": "v2",
        "version_data_set_name": "dataset1-v2",
        "namespace": "system-tce",
        "file_names": [
            {
                "name": "数据处理文件_小T.pdf"
            }
        ],
        "data_process_config_info": []
    }

    pool: database connection pool.
    """
    id = ulid.ulid()
    res = data_process_db_operate.add(req_json, pool=pool, id=id)

    if res["status"] == 200:
        try:

            async def async_text_manipulate(req_json, pool, id):
                await minio_store_process.text_manipulate(req_json, pool=pool, id=id)

            def execute_text_manipulate_task(loop):
                asyncio.set_event_loop(loop)
                loop.run_until_complete(
                    async_text_manipulate(req_json, pool=pool, id=id)
                )

            thread_parallel.run_async_background_task(
                execute_text_manipulate_task, "execute text manipuate task"
            )
        except Exception as ex:
            logger.error(
                "".join(
                    [
                        f"{log_tag_const.MINIO_STORE_PROCESS} There is an error when ",
                        f"start to run the minio store process.\n",
                        f"{traceback.format_exc()}\n",
                    ]
                )
            )

    return res


def delete_by_id(req_json, pool):
    """Delete a record with id"""
    # 删除需要在详情中预览的信息
    data_process_detail_db_operate.delete_transform_by_task_id(req_json, pool=pool)
    # 删除生成的QA信息
    data_process_detail_db_operate.delete_qa_by_task_id(req_json, pool=pool)
    data_process_detail_preview_db_operate.delete_qa_by_task_id(req_json, pool=pool)
    data_process_detail_db_operate.delete_qa_clean_by_task_id(req_json, pool=pool)
    # 删除对应的进度信息
    data_process_document_db_operate.delete_by_task_id(req_json, pool=pool)
    # 删除chunk的信息
    data_process_document_chunk_db_operate.delete_by_task_id(req_json, pool=pool)
    # 删除log信息
    data_process_log_db_operate.delete_by_task_id(req_json, pool=pool)
    data_process_stage_log_db_operate.delete_by_task_id(req_json, pool=pool)

    return data_process_db_operate.delete_by_id(req_json, pool=pool)


def info_by_id(req_json, pool):
    """Get a detail info with id.

    req_json is a dictionary object. for example:
    {
        "id": "01HGWBE48DT3ADE9ZKA62SW4WS"
    }
    """
    id = req_json["id"]

    data = _get_default_data_for_detail()
    _get_and_set_basic_detail_info(data, task_id=id, conn_pool=pool)

    if data["id"] == "":
        return {"status": 200, "message": "", "data": data}

    process_cofig_map = _convert_config_info_to_map(
        data.get("data_process_config_info")
    )

    config_map_for_result = {}
    _set_basic_info_for_config_map_for_result(
        config_map_for_result, process_cofig_map, task_id=id, conn_pool=pool
    )

    _set_children_info_for_config_map_for_result(
        config_map_for_result, process_cofig_map, task_id=id, conn_pool=pool
    )

    # convert the config resule from map to list
    config_list_for_result = []
    for value in config_map_for_result.values():
        config_list_for_result.append(value)

    data["config"] = config_list_for_result

    logger.debug(f"{log_tag_const.DATA_PROCESS_DETAIL} The response data is: \n{data}")

    return {"status": 200, "message": "", "data": data}


def check_task_name(req_json, pool):
    # 判断名称是否已存在
    count = data_process_db_operate.count_by_name(req_json, pool=pool)

    if count.get("data") > 0:
        return {"status": 1000, "message": "任务名称已存在，请重新输入！", "data": ""}

    return {"status": 200, "message": "", "data": ""}


def get_log_info(req_json, pool):
    # 获取任务日志信息
    log_list = data_process_stage_log_db_operate.list_by_task_id(req_json, pool=pool)

    log_dict = []
    for log_info in log_list.get("data"):
        log_dict.append(log_info.get("stage_detail"))

    separator = "=" * 100
    log_detail = ("\n" + separator + "\n").join(log_dict)

    return {"status": 200, "message": "", "data": log_detail}


def get_log_by_file_name(req_json, pool):
    try:
        stage_log_info = data_process_stage_log_db_operate.info_by_stage_and_file_name(
            req_json, pool=pool
        )

        if stage_log_info.get("status") != 200:
            return stage_log_info

        stage_detail = stage_log_info.get("data")[0].get("stage_detail")

        return {"status": 200, "message": "", "data": stage_detail}
    except Exception as ex:
        return {"status": 400, "message": str(ex), "data": traceback.format_exc()}


def retry(req_json, pool):
    """When a task fails, attempt a retry."""
    try:
        logger.debug(f"{log_tag_const.DATA_PROCESS_SERVICE} The task retry start")

        async def async_text_manipulate_retry(req_json, pool):
            minio_store_process.text_manipulate_retry(req_json, pool=pool)

        def execute_text_manipulate_task_retry(loop):
            asyncio.set_event_loop(loop)
            loop.run_until_complete(async_text_manipulate_retry(req_json, pool=pool))

        thread_parallel.run_async_background_task(
            execute_text_manipulate_task_retry, "execute text manipuate task retry"
        )

        return {"status": 200, "message": "任务开始重试!", "data": ""}
    except Exception as ex:
        return {"status": 400, "message": str(ex), "data": traceback.format_exc()}


def _get_default_data_for_detail():
    """Get the data for the detail"""
    return {
        "id": "",
        "name": "",
        "status": "",
        "file_type": "",
        "pre_dataset_name": "",
        "pre_dataset_version": "",
        "post_dataset_name": "",
        "post_dataset_version": "",
        "file_num": 0,
        "start_time": "",
        "end_time": "",
        "create_user": "",
        "data_process_config_info": [],
        "config": [],
    }


def _get_and_set_basic_detail_info(from_result, task_id, conn_pool):
    """Get and set the basic detail info.

    from_result: the from result, it's content will be changed;

    task_id: task id;
    conn_pool: database connection pool
    """
    # step 1
    # Get the detail info from the database.
    detail_info_params = {"id": task_id}
    detail_info_res = data_process_db_operate.info_by_id(
        detail_info_params, pool=conn_pool
    )
    if detail_info_res["status"] == 200 and len(detail_info_res["data"]) > 0:
        detail_info_data = detail_info_res["data"][0]

        file_num = 0
        if detail_info_data.get("file_names"):
            file_num = len(detail_info_data["file_names"])

        from_result["id"] = task_id
        from_result["name"] = detail_info_data["name"]
        from_result["status"] = detail_info_data["status"]
        from_result["file_type"] = detail_info_data["file_type"]
        from_result["file_num"] = file_num
        from_result["pre_dataset_name"] = detail_info_data["pre_data_set_name"]
        from_result["pre_dataset_version"] = detail_info_data["pre_data_set_version"]
        from_result["post_dataset_name"] = detail_info_data["post_data_set_name"]
        from_result["post_dataset_version"] = detail_info_data["post_data_set_version"]
        from_result["start_time"] = detail_info_data["start_datetime"]
        from_result["end_time"] = detail_info_data["end_datetime"]
        from_result["creator"] = detail_info_data["create_user"]
        from_result["error_msg"] = detail_info_data["error_msg"]
        from_result["data_process_config_info"] = detail_info_data[
            "data_process_config_info"
        ]
    else:
        from_result["id"] = ""


def _convert_config_info_to_map(config_info_list):
    """Convert the config info to map.

    config_info_list: a list for example
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
    for item in config_info_list:
        result[item["type"]] = item

    return result


def _set_basic_info_for_config_map_for_result(
    from_result, process_cofig_map, task_id, conn_pool
):
    """Set basic info for the config map for result.

    from_result: the from result, it's content will be changed.
    process_config_map: process config map
    """
    # chunk processing
    if process_cofig_map.get("qa_split"):
        if from_result.get("chunk_processing") is None:
            from_result["chunk_processing"] = {
                "name": "chunk_processing",
                "description": "拆分处理",
                "file_num": _get_qa_process_file_num(
                    task_id=task_id, conn_pool=conn_pool
                ),
                "status": _get_qa_split_status(task_id=task_id, conn_pool=conn_pool),
                "children": [],
            }

    # data clean
    if (
        process_cofig_map.get("remove_invisible_characters")
        or process_cofig_map.get("space_standardization")
        or process_cofig_map.get("remove_garbled_text")
        or process_cofig_map.get("traditional_to_simplified")
        or process_cofig_map.get("remove_html_tag")
        or process_cofig_map.get("remove_emojis")
    ):
        if from_result.get("clean") is None:
            from_result["clean"] = {
                "name": "clean",
                "description": "异常清洗配置",
                "file_num": _get_clean_process_file_num(
                    task_id=task_id, conn_pool=conn_pool
                ),
                "status": "success",
                "children": [],
            }

    # remove privacy
    if (
        process_cofig_map.get("remove_email")
        or process_cofig_map.get("remove_ip_address")
        or process_cofig_map.get("remove_number")
    ):
        if from_result.get("privacy") is None:
            from_result["privacy"] = {
                "name": "privacy",
                "description": "数据隐私处理",
                "file_num": _get_privacy_process_file_num(
                    task_id=task_id, conn_pool=conn_pool
                ),
                "status": "success",
                "children": [],
            }


def _set_children_info_for_config_map_for_result(
    from_result, process_cofig_map, task_id, conn_pool
):
    """Set child list for the config for result

    from_result: the from result, it's content will be changed.
    process_config_map: process config map;
    task_id: task id,
    conn_pool: database connection pool
    """
    # insert the qa list
    if process_cofig_map.get("qa_split"):
        from_result["chunk_processing"]["children"].append(
            {
                "name": "qa_split",
                "enable": "true",
                "zh_name": "QA拆分",
                "description": "根据文件中的文档内容，自动将文件做 QA 拆分处理。",
                "llm_config": process_cofig_map.get("qa_split").get("llm_config"),
                "preview": _get_qa_list_preview(task_id=task_id, conn_pool=conn_pool),
                "file_progress": _get_file_progress(
                    task_id=task_id, conn_pool=conn_pool
                ),
            }
        )

    # remove invisible characters
    if process_cofig_map.get("remove_invisible_characters"):
        from_result["clean"]["children"].append(
            {
                "name": "remove_invisible_characters",
                "enable": "true",
                "zh_name": "移除不可见字符",
                "description": "移除ASCII中的一些不可见字符, 如0-32 和127-160这两个范围",
                "preview": _get_transform_preview_list(
                    task_id=task_id,
                    transform_type="remove_invisible_characters",
                    conn_pool=conn_pool,
                ),
            }
        )

    # space standardization
    if process_cofig_map.get("space_standardization"):
        from_result["clean"]["children"].append(
            {
                "name": "space_standardization",
                "enable": "true",
                "zh_name": "空格处理",
                "description": "将不同的unicode空格比如u2008, 转成正常的空格",
                "preview": _get_transform_preview_list(
                    task_id=task_id,
                    transform_type="space_standardization",
                    conn_pool=conn_pool,
                ),
            }
        )

    # remove garbled text
    if process_cofig_map.get("remove_garbled_text"):
        from_result["clean"]["children"].append(
            {
                "name": "remove_garbled_text",
                "enable": "true",
                "zh_name": "去除乱码",
                "description": "去除乱码和无意义的unicode",
                "preview": _get_transform_preview_list(
                    task_id=task_id,
                    transform_type="remove_garbled_text",
                    conn_pool=conn_pool,
                ),
            }
        )

    # traditional to simplified
    if process_cofig_map.get("traditional_to_simplified"):
        from_result["clean"]["children"].append(
            {
                "name": "traditional_to_simplified",
                "enable": "true",
                "zh_name": "繁转简",
                "description": "繁体转简体，如“不經意，妳的笑容”清洗成“不经意，你的笑容”",
                "preview": _get_transform_preview_list(
                    task_id=task_id,
                    transform_type="traditional_to_simplified",
                    conn_pool=conn_pool,
                ),
            }
        )

    # remove html tag
    if process_cofig_map.get("remove_html_tag"):
        from_result["clean"]["children"].append(
            {
                "name": "remove_html_tag",
                "enable": "true",
                "zh_name": "去除网页标识符",
                "description": "移除文档中的html标签, 如<html>,<dev>,<p>等",
                "preview": _get_transform_preview_list(
                    task_id=task_id,
                    transform_type="remove_html_tag",
                    conn_pool=conn_pool,
                ),
            }
        )

    # remove emojis
    if process_cofig_map.get("remove_emojis"):
        from_result["clean"]["children"].append(
            {
                "name": "remove_emojis",
                "enable": "true",
                "zh_name": "去除表情",
                "description": "去除文档中的表情，如‘🐰’, ‘🧑🏼’等",
                "preview": _get_transform_preview_list(
                    task_id=task_id, transform_type="remove_emojis", conn_pool=conn_pool
                ),
            }
        )

    # remove email
    if process_cofig_map.get("remove_email"):
        from_result["privacy"]["children"].append(
            {
                "name": "remove_email",
                "enable": "true",
                "zh_name": "去除Email",
                "description": "去除email地址",
                "preview": _get_transform_preview_list(
                    task_id=task_id, transform_type="remove_email", conn_pool=conn_pool
                ),
            }
        )

    # remove ip address
    if process_cofig_map.get("remove_ip_address"):
        from_result["privacy"]["children"].append(
            {
                "name": "remove_ip_address",
                "enable": "true",
                "zh_name": "去除IP地址",
                "description": "去除IPv4 或者 IPv6 地址",
                "preview": _get_transform_preview_list(
                    task_id=task_id,
                    transform_type="remove_ip_address",
                    conn_pool=conn_pool,
                ),
            }
        )

    # remove number
    if process_cofig_map.get("remove_number"):
        from_result["privacy"]["children"].append(
            {
                "name": "remove_number",
                "enable": "true",
                "zh_name": "去除数字",
                "description": "去除数字和字母数字标识符，如电话号码、信用卡号、十六进制散列等，同时跳过年份和简单数字的实例",
                "preview": _get_transform_preview_list(
                    task_id=task_id, transform_type="remove_number", conn_pool=conn_pool
                ),
            }
        )


def _get_transform_preview_list(task_id, transform_type, conn_pool):
    """ "Get transofm preview list.

    task_id: task id;
    transform_type: transform type
    conn_pool: database connection pool;
    """
    transform_preview = []
    # step 1
    # list file name in transform
    list_file_name_params = {"task_id": task_id, "transform_type": transform_type}
    list_file_name_res = data_process_detail_db_operate.list_file_name_for_transform(
        list_file_name_params, pool=conn_pool
    )
    if list_file_name_res["status"] == 200:
        for item in list_file_name_res["data"]:
            transform_preview.append({"file_name": item["file_name"], "content": []})
    # step 2
    # iterate the transform preview
    for item in transform_preview:
        list_transform_params = {
            "task_id": task_id,
            "transform_type": transform_type,
            "file_name": item["file_name"],
        }
        list_transform_res = (
            data_process_detail_db_operate.top_n_list_transform_for_preview(
                list_transform_params, pool=conn_pool
            )
        )
        if list_transform_res["status"] == 200:
            for item_transform in list_transform_res["data"]:
                item["content"].append(
                    {
                        "pre": item_transform["pre_content"],
                        "post": item_transform["post_content"],
                    }
                )

    return transform_preview


def _get_qa_list_preview(task_id, conn_pool):
    """Get the QA list preview.

    task_id: task od;
    conn_pool: database connection pool
    """
    logger.debug("".join([f"{log_tag_const.MINIO_STORE_PROCESS} Get preview for QA "]))
    qa_list_preview = []
    # step 1
    # list file name in QA
    list_file_name_params = {"task_id": task_id, "transform_type": "qa_split"}
    list_file_name_res = (
        data_process_detail_preview_db_operate.list_file_name_by_task_id(
            list_file_name_params, pool=conn_pool
        )
    )
    if list_file_name_res["status"] == 200:
        for item in list_file_name_res["data"]:
            qa_list_preview.append({"file_name": item["file_name"], "content": []})

    # step 2
    # iterate the QA list preview
    list_qa_params = {"task_id": task_id, "transform_type": "qa_split"}
    list_qa_res = data_process_detail_preview_db_operate.list_for_preview(
        list_qa_params, pool=conn_pool
    )
    for item in qa_list_preview:
        for item_qa in list_qa_res["data"]:
            if item.get("file_name") == item_qa.get("file_name"):
                item["content"].append(
                    {"pre": item_qa["pre_content"], "post": item_qa["post_content"]}
                )

    return qa_list_preview


def _get_file_progress(task_id, conn_pool):
    """Get file progress.

    task_id: task id;
    conn_pool: database connection pool
    """
    # Get the detail info from the database.
    detail_info_params = {"task_id": task_id}
    list_file = data_process_document_db_operate.list_file_by_task_id(
        detail_info_params, pool=conn_pool
    )

    return list_file.get("data")


def _get_qa_split_status(task_id, conn_pool):
    """Get file progress.

    task_id: task id;
    conn_pool: database connection pool
    """
    # Get the detail info from the database.
    status = "doing"
    detail_info_params = {"task_id": task_id}
    list_file = data_process_document_db_operate.list_file_by_task_id(
        detail_info_params, pool=conn_pool
    )

    if list_file.get("status") != 200 or len(list_file.get("data")) == 0:
        return "fail"

    file_dict = list_file.get("data")

    # 当所有文件状态都为success，则status为success
    all_success = all(item["status"] == "success" for item in file_dict)
    if all_success:
        return "success"

    # 当所有文件状态都为not_start，则status为not_start
    all_success = all(item["status"] == "not_start" for item in file_dict)
    if all_success:
        return "not_start"

    # 只要有一个文件状态为fail，则status为fail
    status_fail = any(item["status"] == "fail" for item in file_dict)
    if status_fail:
        return "fail"

    return status


def _get_qa_process_file_num(task_id, conn_pool):
    list_file_name_params = {"task_id": task_id}
    list_file_name_res = data_process_detail_db_operate.list_file_name_in_qa_by_task_id(
        list_file_name_params, pool=conn_pool
    )

    if list_file_name_res.get("status") == 200:
        return len(list_file_name_res.get("data"))

    logger.error(
        "".join(
            [
                f"{log_tag_const.MINIO_STORE_PROCESS} Get the number of files processed after QA "
            ]
        )
    )
    return 0


def _get_clean_process_file_num(task_id, conn_pool):
    list_file_name_params = {"task_id": task_id}
    list_file_name_res = data_process_detail_db_operate.list_file_name_for_clean(
        list_file_name_params, pool=conn_pool
    )

    if list_file_name_res.get("status") == 200:
        return len(list_file_name_res.get("data"))

    logger.error(
        "".join(
            [
                f"{log_tag_const.MINIO_STORE_PROCESS} Get the number of files processed after cleaning "
            ]
        )
    )
    return 0


def _get_privacy_process_file_num(task_id, conn_pool):
    list_file_name_params = {"task_id": task_id}
    list_file_name_res = data_process_detail_db_operate.list_file_name_for_privacy(
        list_file_name_params, pool=conn_pool
    )

    if list_file_name_res.get("status") == 200:
        return len(list_file_name_res.get("data"))

    logger.error(
        "".join(
            [
                f"{log_tag_const.MINIO_STORE_PROCESS} Get the number of files processed after privacy "
            ]
        )
    )
    return 0
