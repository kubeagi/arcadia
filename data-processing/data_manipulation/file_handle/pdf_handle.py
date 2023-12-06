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


import logging
import os
import traceback

import pandas as pd
import ulid
from common import log_tag_const
from common.config import config
from database_operate import data_process_detail_db_operate
from langchain.text_splitter import SpacyTextSplitter
from llm_api_service.qa_provider_open_ai import QAProviderOpenAI
from llm_api_service.qa_provider_zhi_pu_ai_online import \
    QAProviderZhiPuAIOnline
from transform.text import clean_transform, privacy_transform
from utils import csv_utils, file_utils, pdf_utils

logger = logging.getLogger(__name__)


def text_manipulate(
    file_name,
    support_type,
    conn_pool,
    task_id,
    create_user,
    chunk_size,
    chunk_overlap
):
    """Manipulate the text content from a pdf file.
    
    file_name: file name;
    support_type: support type;
    conn_pool: database connection pool;
    task_id: data process task id;
    chunk_size: chunk size;
    chunk_overlap: chunk overlap;
    """
    
    logger.debug(f"{log_tag_const.PDF_HANDLE} Start to manipulate the text in pdf")

    try:
        pdf_file_path = file_utils.get_temp_file_path()
        file_path = pdf_file_path + 'original/' + file_name
        
        # step 1
        # Get the content from the pdf fild.
        content = pdf_utils.get_content(file_path)
        logger.debug(f"{log_tag_const.PDF_HANDLE} The pdf content is\n {content}")

        support_type_map = _convert_support_type_to_map(support_type)
        
        # step 2
        # Clean the data such as removing invisible characters.
        clean_result = _data_clean(
            support_type_map=support_type_map,
            file_name=file_name,
            data=content,
            conn_pool=conn_pool,
            task_id=task_id,
            create_user=create_user
        )

        if clean_result['status'] == 200:
            content = clean_result['data']

        # step 3
        # Remove the privacy info such as removing email.
        clean_result = _remove_privacy_info(
            support_type_map=support_type_map,
            file_name=file_name,
            data=content,
            conn_pool=conn_pool,
            task_id=task_id,
            create_user=create_user
        )

        if clean_result['status'] == 200:
            content = clean_result['data']


        
        # 数据量
        object_count = 0
        object_name = ''
        if support_type_map.get('qa_split'):
            logger.debug(f"{log_tag_const.QA_SPLIT} Start to split QA.")

            qa_data = _generate_qa_list(
                chunk_size=chunk_size,
                chunk_overlap=chunk_overlap,
                data=content
            )

            logger.debug(f"{log_tag_const.QA_SPLIT} The QA data is: \n{qa_data}\n")

            # start to insert qa data
            for i in range(len(qa_data)):
                if i == 0:
                    continue
                qa_insert_item = {
                    'id': ulid.ulid(),
                    'task_id': task_id,
                    'file_name': file_name,
                    'question': qa_data[i][0],
                    'answer': qa_data[i][1],
                    'create_user': create_user
                }
               
                data_process_detail_db_operate.insert_question_answer_info(
                    qa_insert_item,
                    pool=conn_pool
                )

            # Save the csv file.        
            file_name_without_extension = file_name.rsplit('.', 1)[0] + '_final'
            csv_utils.save_csv(
                file_name=file_name_without_extension + '.csv',
                phase_value='final',
                data=qa_data
            )
            
            object_name = file_name_without_extension + '.csv'
            # 减 1 是为了去除表头
            object_count = len(qa_data) - 1
            
            logger.debug(f"{log_tag_const.QA_SPLIT} Finish splitting QA.")
        
        logger.debug(f"{log_tag_const.PDF_HANDLE} Finish manipulating the text in pdf")
        return {
            'status': 200,
            'message': '',
            'data': {
                'object_name': object_name,
                'object_count': object_count
            }
        }
    except Exception as ex:
        logger.error(''.join([
            f"{log_tag_const.PDF_HANDLE} There is an error when manipulate ",
            f"the text in pdf handler. \n{traceback.format_exc()}"
        ]))
        logger.debug(f"{log_tag_const.PDF_HANDLE} Finish manipulating the text in pdf")
        return {
            'status': 400,
            'message': str(ex),
            'data': traceback.format_exc()
        }

def _data_clean(
    support_type_map,
    data,
    task_id,
    file_name,
    create_user,
    conn_pool
):
    """Clean the data.
    
    support_type_map: example
        {
            "qa_split": 1, 
            "remove_invisible_characters": 1, 
            "space_standardization": 1, 
            "remove_email": 1
        }
    data: data;
    file_name: file name;
    conn_pool: database connection pool;
    task_id: data process task id;
    """
    # remove invisible characters
    if support_type_map.get('remove_invisible_characters'):
        result = clean_transform.remove_invisible_characters(
            text=data
        )
        if result['status'] == 200:
            clean_data = result['data']['clean_data']
            if len(clean_data) > 0:
                for item in clean_data:
                    task_detail_item = {
                        'id': ulid.ulid(),
                        'task_id': task_id,
                        'file_name': file_name,
                        'transform_type': 'remove_invisible_characters',
                        'pre_content': item['pre_content'],
                        'post_content': item['post_content'],
                        'create_user': create_user
                    }
                    data_process_detail_db_operate.insert_transform_info(
                        task_detail_item,
                        pool=conn_pool
                    )
            data = result['data']['text']

    
    # process for space standardization
    if support_type_map.get('space_standardization'):
        result = clean_transform.space_standardization(
            text=data
        )
        if result['status'] == 200:
            clean_data = result['data']['clean_data']
            if len(clean_data) > 0:
                for item in clean_data:
                    task_detail_item = {
                        'id': ulid.ulid(),
                        'task_id': task_id,
                        'file_name': file_name,
                        'transform_type': 'space_standardization',
                        'pre_content': item['pre_content'],
                        'post_content': item['post_content'],
                        'create_user': create_user
                    }
                    data_process_detail_db_operate.insert_transform_info(
                        task_detail_item,
                        pool=conn_pool
                    )
            data = result['data']['text']


    # process for remove garbled text
    if support_type_map.get('remove_garbled_text'):
        result = clean_transform.remove_garbled_text(
            text=data
        )
        if result['status'] == 200:
            if result['data']['found'] > 0:
                task_detail_item = {
                    'id': ulid.ulid(),
                    'task_id': task_id,
                    'file_name': file_name,
                    'transform_type': 'remove_garbled_text',
                    'pre_content': data,
                    'post_content': result['data']['text'],
                        'create_user': create_user
                }
                data_process_detail_db_operate.insert_transform_info(
                    task_detail_item,
                    pool=conn_pool
                )
            data = result['data']['text']


    # process for Traditional Chinese to Simplified Chinese
    if support_type_map.get('traditional_to_simplified'):
        result = clean_transform.traditional_to_simplified(
            text=data
        )
        if result['status'] == 200:
            if result['data']['found'] > 0:
                task_detail_item = {
                    'id': ulid.ulid(),
                    'task_id': task_id,
                    'file_name': file_name,
                    'transform_type': 'traditional_to_simplified',
                    'pre_content': data,
                    'post_content': result['data']['text'],
                        'create_user': create_user
                }
                data_process_detail_db_operate.insert_transform_info(
                    task_detail_item,
                    pool=conn_pool
                )
            data = result['data']['text']


    # process for clean html code in text samples
    if support_type_map.get('remove_html_tag'):
        result = clean_transform.remove_html_tag(
            text=data
        )
        if result['status'] == 200:
            if result['data']['found'] > 0:
                task_detail_item = {
                    'id': ulid.ulid(),
                    'task_id': task_id,
                    'file_name': file_name,
                    'transform_type': 'remove_html_tag',
                    'pre_content': data,
                    'post_content': result['data']['text'],
                    'create_user': create_user
                }
                data_process_detail_db_operate.insert_transform_info(
                    task_detail_item,
                    pool=conn_pool
                )
            data = result['data']['text']
    

    # process for remove emojis
    if support_type_map.get('remove_emojis'):
        result = clean_transform.remove_emojis(
            text=data
        )
        if result['status'] == 200:
            clean_data = result['data']['clean_data']
            if len(clean_data) > 0:
                for item in clean_data:
                    task_detail_item = {
                        'id': ulid.ulid(),
                        'task_id': task_id,
                        'file_name': file_name,
                        'transform_type': 'remove_emojis',
                        'pre_content': item['pre_content'],
                        'post_content': item['post_content'],
                        'create_user': create_user
                    }
                    data_process_detail_db_operate.insert_transform_info(
                        task_detail_item,
                        pool=conn_pool
                    )
            data = result['data']['text']

    return {
        'status': 200,
        'message': '',
        'data': data
    }


def _remove_privacy_info(
    support_type_map,
    data,
    task_id,
    file_name,
    create_user,
    conn_pool
):
    """"Remove the privacy info such as removing email.
    
    support_type_map: example
        {
            "qa_split": 1, 
            "remove_invisible_characters": 1, 
            "space_standardization": 1, 
            "remove_email": 1
        }
    data: data;
    file_name: file name;
    conn_pool: database connection pool;
    task_id: data process task id;
    """
    # remove email
    if support_type_map.get('remove_email'):
        result = privacy_transform.remove_email(
            text=data
        )
        if result['status'] == 200:
            clean_data = result['data']['clean_data']
            if len(clean_data) > 0:
                for item in clean_data:
                    task_detail_item = {
                        'id': ulid.ulid(),
                        'task_id': task_id,
                        'file_name': file_name,
                        'transform_type': 'remove_email',
                        'pre_content': item['pre_content'],
                        'post_content': item['post_content'],
                        'create_user': create_user
                    }
                    data_process_detail_db_operate.insert_transform_info(
                        task_detail_item,
                        pool=conn_pool
                    )
            data = result['data']['text']
        

    # remove ip addresses
    if support_type_map.get('remove_ip_address'):
        result = privacy_transform.remove_ip_address(
            text=data
        )
        if result['status'] == 200:
            clean_data = result['data']['clean_data']
            if len(clean_data) > 0:
                for item in clean_data:
                    task_detail_item = {
                        'id': ulid.ulid(),
                        'task_id': task_id,
                        'file_name': file_name,
                        'transform_type': 'remove_ip_address',
                        'pre_content': item['pre_content'],
                        'post_content': item['post_content'],
                        'create_user': create_user
                    }
                    data_process_detail_db_operate.insert_transform_info(
                        task_detail_item,
                        pool=conn_pool
                    )
            data = result['data']['text']

    # remove number
    if support_type_map.get('remove_number'):
        # remove phone
        result = privacy_transform.remove_phone(
            text=data
        )
        if result['status'] == 200:
            clean_data = result['data']['clean_data']
            if len(clean_data) > 0:
                for item in clean_data:
                    task_detail_item = {
                        'id': ulid.ulid(),
                        'task_id': task_id,
                        'file_name': file_name,
                        'transform_type': 'remove_number',
                        'pre_content': item['pre_content'],
                        'post_content': item['post_content'],
                        'create_user': create_user
                    }
                    data_process_detail_db_operate.insert_transform_info(
                        task_detail_item,
                        pool=conn_pool
                    )
            data = result['data']['text']
        
        # remove id card
        result = privacy_transform.remove_id_card(
            text=data
        )
        if result['status'] == 200:
            clean_data = result['data']['clean_data']
            if len(clean_data) > 0:
                for item in clean_data:
                    task_detail_item = {
                        'id': ulid.ulid(),
                        'task_id': task_id,
                        'file_name': file_name,
                        'transform_type': 'remove_number',
                        'pre_content': item['pre_content'],
                        'post_content': item['post_content'],
                        'create_user': create_user
                    }
                    data_process_detail_db_operate.insert_transform_info(
                        task_detail_item,
                        pool=conn_pool
                    )
            data = result['data']['text']

        # remove weixin
        result = privacy_transform.remove_weixin(
            text=data
        )
        if result['status'] == 200:
            clean_data = result['data']['clean_data']
            if len(clean_data) > 0:
                for item in clean_data:
                    task_detail_item = {
                        'id': ulid.ulid(),
                        'task_id': task_id,
                        'file_name': file_name,
                        'transform_type': 'remove_number',
                        'pre_content': item['pre_content'],
                        'post_content': item['post_content'],
                        'create_user': create_user
                    }
                    data_process_detail_db_operate.insert_transform_info(
                        task_detail_item,
                        pool=conn_pool
                    )
            data = result['data']['text']

        # remove bank card
        result = privacy_transform.remove_bank_card(
            text=data
        )
        if result['status'] == 200:
            clean_data = result['data']['clean_data']
            if len(clean_data) > 0:
                for item in clean_data:
                    task_detail_item = {
                        'id': ulid.ulid(),
                        'task_id': task_id,
                        'file_name': file_name,
                        'transform_type': 'remove_number',
                        'pre_content': item['pre_content'],
                        'post_content': item['post_content'],
                        'create_user': create_user
                    }
                    data_process_detail_db_operate.insert_transform_info(
                        task_detail_item,
                        pool=conn_pool
                    )
            data = result['data']['text']
   
    return {
        'status': 200,
        'message': '',
        'data': data
    }



def _generate_qa_list(
    chunk_size,
    chunk_overlap,
    data
):
    """Generate the Question and Answer list.
    
    chunk_size: chunck size;
    chunk_overlap: chunk overlap;
    data: the text used to generate QA;
    """
    # step 1
    # Split the text.
    if chunk_size is None:
        chunk_size = config.knowledge_chunk_size

    if chunk_overlap is None:
        chunk_overlap = config.knowledge_chunk_overlap

    text_splitter = SpacyTextSplitter(
        separator="\n\n",
        pipeline="zh_core_web_sm",
        chunk_size=int(chunk_size),
        chunk_overlap=int(chunk_overlap)
    )
    texts = text_splitter.split_text(data)
    
    logger.debug(''.join([
        f"original text is: \n{data}\n",
        f"splitted text is: \n{texts}\n"
    ]))


    # step 2
    # Generate the QA list.
    qa_list = [['q', 'a']]
    if config.llm_use_type == 'open_ai':
        qa_provider = QAProviderOpenAI()
        for item in texts:
            text = item.replace("\n", "")
            data = qa_provider.generate_qa_list(text)
            qa_list.extend(data)
    elif config.llm_use_type == 'zhi_pu_online':
        qa_provider = QAProviderZhiPuAIOnline()
        for item in texts:
            text = item.replace("\n", "")
            data = qa_provider.generate_qa_list(text)
            qa_list.extend(data)

    return qa_list


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
        result[item['type']] = 1

    return result