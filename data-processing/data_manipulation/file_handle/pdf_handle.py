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

###
# PDF文件处理
# @author: wangxinbiao
# @date: 2023-11-01 16:43:01
# modify history
# ==== 2023-11-01 16:43:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###

import logging
import os

import pandas as pd
from langchain.document_loaders import PyPDFLoader
from langchain.text_splitter import SpacyTextSplitter
from pypdf import PdfReader

from common import config
from file_handle import csv_handle
from transform.text import QA_transform, clean_transform, privacy_transform
from utils import file_utils

logger = logging.getLogger('pdf_handle')

###
# 文本数据处理
# @author: wangxinbiao
# @date: 2023-11-17 16:14:01
# modify history
# ==== 2023-11-17 16:14:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###


async def text_manipulate(request, opt={}):
    logger.info("pdf text manipulate!")

    try:
        
        file_name = opt['file_name']
        support_type = opt['support_type']
        
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
            qa_data = await generate_QA(request, {
                'support_type': support_type,
                'data': content
            })
        
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

        return {
            'status': 200,
            'message': '',
            'data': ''
        }
    except Exception as ex:
        logger.error(str(ex))
        return {
            'status': 400,
            'message': '',
            'data': ''
        }

###
# 数据异常清洗
# @author: wangxinbiao
# @date: 2023-11-17 16:14:01
# modify history
# ==== 2023-11-17 16:14:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###


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


###
# 去隐私
# @author: wangxinbiao
# @date: 2023-11-17 16:14:01
# modify history
# ==== 2023-11-17 16:14:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###


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
            return {
                'status': 400,
                'message': '去邮箱',
                'data': ''
            }            
        
        data = result['data']

    logger.info("pdf text privacy erosion stop!")

    return {
        'status': 200,
        'message': '',
        'data': data
    }

###
# 获取PDF内容
# @author: wangxinbiao
# @date: 2023-11-17 16:14:01
# modify history
# ==== 2023-11-17 16:14:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###


async def get_content(opt={}):
    file_path = opt['file_path']

    reader = PdfReader(file_path)
    number_of_pages = len(reader.pages)
    pages = reader.pages
    content = ""
    for page in pages:
        content += page.extract_text()

    return content

###
# QA拆分
# @author: wangxinbiao
# @date: 2023-11-17 16:14:01
# modify history
# ==== 2023-11-17 16:14:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###


async def generate_QA(request, opt={}):
    logger.info("pdf text generate qa start!")
    request_json = request.json

    # 文本分段
    chunk_size = config.knowledge_chunk_size
    if "chunk_size" in request_json:
        chunk_size = request_json['chunk_size']

    chunk_overlap = config.knowledge_chunk_overlap
    if "chunk_overlap" in request_json:
        chunk_overlap = request_json['chunk_overlap']

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

    for item in texts:
        text = item.replace("\n", "")
        data = await QA_transform.generate_QA({
            'text': text
        })

        qa_list.extend(data)
        
    logger.info("pdf text generate qa stop!")

    return qa_list

###
# 文本分段
# @author: wangxinbiao
# @date: 2023-11-17 16:14:01
# modify history
# ==== 2023-11-17 16:14:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###


async def document_chunk(request, opt={}):

    separator = "\n\n"

    text_splitter = SpacyTextSplitter(
        separator=separator,
        pipeline="zh_core_web_sm",
        chunk_size=opt['chunk_size'],
        chunk_overlap=opt['chunk_overlap']
    )
    texts = text_splitter.split_text(opt['data'])
        
    return texts