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
import pandas as pd
from langchain.document_loaders import PyPDFLoader
from langchain.text_splitter import SpacyTextSplitter
from pypdf import PdfReader
import ulid

from common import config, log_tag_const
from database_operate import data_process_detail_db_operate
from file_handle import csv_handle
from llm_api_service import zhi_pu_ai_service
from transform.text import clean_transform, privacy_transform
from utils import file_utils



logger = logging.getLogger(__name__)


async def text_manipulate(req_json, opt={}):
    logger.debug(f"{log_tag_const.PDF_HANDLE} Start to manipulate the text in pdf")

    try:
        
        file_name = opt['file_name']
        support_type = opt['support_type']
        conn_pool = opt['conn_pool'] # database connectionn pool
        
        # 数据量
        object_count = 0
        object_name = ''
        
        pdf_file_path = await file_utils.get_temp_file_path()
        file_path = pdf_file_path + 'original/' + file_name
        

        # 获取PDF文件的内容
        content = await get_content({
            "file_path": file_path
        })

        logger.info("start text manipulate!")

        # 数据清洗
        clean_result = await data_clean({
            'support_type': support_type,
            'file_name': file_name,
            'data': content
        })

        if clean_result['status'] != 200:
            return clean_result
        else:
            content = clean_result['data']

        # 去隐私
        clean_result = await privacy_erosion({
            'support_type': support_type,
            'file_name': file_name,
            'data': content
        })

        if clean_result['status'] != 200:
            return clean_result
        else:
            content = clean_result['data']

        # QA拆分
        if any(d.get('type') == 'qa_split' for d in support_type):

            qa_data = await generate_QA(req_json, {
                'support_type': support_type,
                'data': content
            })

            # qa_data = []

            logger.debug(f"{log_tag_const.QA_SPLIT} The QA data is: \n{qa_data}\n")

            # start to insert qa data
            for i in range(len(qa_data)):
                if i == 0:
                    continue
                qa_insert_item = {
                    'id': ulid.ulid(),
                    'task_id': opt['task_id'],
                    'file_name': file_name,
                    'question': qa_data[i][0],
                    'answer': qa_data[i][1]
                }
               
                await data_process_detail_db_operate.insert_question_answer_info(
                    qa_insert_item, {
                        'pool': opt['conn_pool']
                    }
                )
                
        
            # 将生成的QA数据保存为CSV文件
            new_file_name = await file_utils.get_file_name({
                'file_name': file_name,
                'handle_name': 'final'
            })


            file_name_without_extension = file_name.rsplit('.', 1)[0]

            await csv_handle.save_csv({
                'file_name': file_name_without_extension + '.csv',
                'phase_value': 'final',
                'data': qa_data
            })
            
            object_name = file_name_without_extension + '.csv'

            # 减 1 是为了去除表头
            object_count = len(qa_data) - 1

        return {
            'status': 200,
            'message': '',
            'data': {
                'object_name': object_name,
                'object_count': object_count
            }
        }
    except Exception as ex:
        logger.error(str(ex))
        return {
            'status': 400,
            'message': '',
            'data': ''
        }



async def data_clean(opt={}):
    logger.info("pdf text data clean start!")
    support_type = opt['support_type']
    data = opt['data']

    # 去除不可见字符
    if any(d.get('type') == 'remove_invisible_characters' for d in support_type):
        result = await clean_transform.remove_invisible_characters({
            'text': data
        })

        if result['status'] != 200:
            return {
                'status': 400,
                'message': '去除不可见字符失败',
                'data': ''
            }            
        
        data = result['data']
    
    # 空格处理
    if any(d.get('type') == 'space_standardization' for d in support_type):
        result = await clean_transform.space_standardization({
            'text': data
        })

        if result['status'] != 200:
            return {
                'status': 400,
                'message': '空格处理失败',
                'data': ''
            }            
        
        data = result['data']

    logger.info("pdf text data clean stop!")

    return {
        'status': 200,
        'message': '',
        'data': data
    }


async def privacy_erosion(opt={}):
    logger.info("pdf text privacy erosion start!")
    support_type = opt['support_type']
    data = opt['data']

    # 去邮箱
    if any(d.get('type') == 'remove_email' for d in support_type):
        result = await privacy_transform.remove_email({
            'text': data
        })

        if result['status'] != 200:
            return result            
        
        data = result['data']

    logger.info("pdf text privacy erosion stop!")

    return {
        'status': 200,
        'message': '',
        'data': data
    }



async def get_content(opt={}):
    file_path = opt['file_path']

    reader = PdfReader(file_path)
    number_of_pages = len(reader.pages)
    pages = reader.pages
    content = ""
    for page in pages:
        content += page.extract_text()

    return content


async def generate_QA(req_json, opt={}):
    logger.info("pdf text generate qa start!")
    
    # 文本分段
    chunk_size = config.knowledge_chunk_size
    if "chunk_size" in req_json:
        chunk_size = req_json['chunk_size']

    chunk_overlap = config.knowledge_chunk_overlap
    if "chunk_overlap" in req_json:
        chunk_overlap = req_json['chunk_overlap']

    separator = "\n\n"

    text_splitter = SpacyTextSplitter(
        separator=separator,
        pipeline="zh_core_web_sm",
        chunk_size=int(chunk_size),
        chunk_overlap=int(chunk_overlap),
    )
    texts = text_splitter.split_text(opt['data'])

    # 生成QA
    qa_list = [['q', 'a']]
    await zhi_pu_ai_service.init_service({
        'api_key': config.zhipuai_api_key
    })
    for item in texts:
        text = item.replace("\n", "")
        data = await zhi_pu_ai_service.generate_qa({
            'text': text
        })

        qa_list.extend(data)
        
    logger.info("pdf text generate qa stop!")

    return qa_list


async def document_chunk(req_json, opt={}):

    separator = "\n\n"

    text_splitter = SpacyTextSplitter(
        separator=separator,
        pipeline="zh_core_web_sm",
        chunk_size=opt['chunk_size'],
        chunk_overlap=opt['chunk_overlap']
    )
    texts = text_splitter.split_text(opt['data'])
        
    return texts