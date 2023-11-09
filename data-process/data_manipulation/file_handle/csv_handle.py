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
# CSV文件处理
# @author: wangxinbiao
# @date: 2023-11-01 16:43:01
# modify history
# ==== 2023-11-01 16:43:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###

import csv
import ulid
import pandas as pd
import os
import logging

from transform.text import (
    clean_transform,
    privacy_transform
)

from utils import (
    date_time_utils,
    file_utils
)

logger = logging.getLogger('csv_handle')

###
# 文本数据处理
# @author: wangxinbiao
# @date: 2023-11-02 14:42:01
# modify history
# ==== 2023-11-02 14:42:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###
async def text_manipulate(opt={}):
    logger.info("csv text manipulate!")

    """
        数据处理逻辑：
            处理某条数据时，如果某个方式（比如：去除不可见字符）处理失败了，则直接结束，不在处理，整个文件都视作处理失败
            
    """
    
    try:
        file_name = opt['file_name']
        support_type = opt['support_type']

        csv_file_path = await file_utils.get_temp_file_path()
        file_path = csv_file_path + 'original/' + file_name

        # 获取CSV文件的内容
        data = pd.read_csv(file_path)

        logger.info('data')

        clean_text_list = []
        logger.info("start text manipulate!")
        text_data = data['prompt']

        # 数据清洗
        clean_result = await data_clean({
            'support_type': support_type,
            'file_name': file_name,
            'data': text_data
        })

        if clean_result['status'] != 200:
            return clean_result

        text_data = clean_result['data']

        
        # 将清洗后的文件保存为final
        new_file_name = await file_utils.get_file_name({
            'file_name': file_name,
            'handle_name': 'final'
        })

        await save_csv({
            'file_name': new_file_name,
            'phase_value': 'final',
            'data': text_data
        })

        return {
            'status': 200,
            'message': '',
            'data': ''
        }
    except Exception as ex:
        return {
            'status': 400,
            'message': '',
            'data': ''
        }


###
# 数据异常清洗
# @author: wangxinbiao
# @date: 2023-11-08 09:32:01
# modify history
# ==== 2023-11-08 09:32:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###
async def data_clean(opt={}):
    logger.info("csv text data clean start!")
    support_type = opt['support_type']
    data = opt['data']

    # 去除不可见字符
    if 'remove_invisible_characters' in support_type:
        clean_data = []
        for item in data:
            result = await remove_invisible_characters({
                'text': item
            })

            if result['status'] != 200:
                return {
                    'status': 400,
                    'message': '去除不可见字符失败',
                    'data': ''
                }

            clean_data.append(result['data'])
        data = clean_data

        # 将文件存为middle
        file_name = await file_utils.get_file_name({
            'file_name': opt['file_name'],
            'handle_name': 'middle'
        })

        await save_csv({
            'file_name': file_name,
            'phase_value': 'middle',
            'data': data
        })

    logger.info("csv text data clean stop!")
    
    return {
        'status': 200,
        'message': '',
        'data': data
    }


###
# 去除不可见字符
# @author: wangxinbiao
# @date: 2023-11-02 14:42:01
# modify history
# ==== 2023-11-02 14:42:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###
async def remove_invisible_characters(opt={}):
    return await clean_transform.remove_invisible_characters({
            'text': opt['text']
        })

###
# 去除邮箱地址
# @author: wangxinbiao
# @date: 2023-11-02 14:42:01
# modify history
# ==== 2023-11-02 14:42:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###
async def remove_email(opt={}):
    return await privacy_transform.remove_email({
            'text': opt['text']
        })

###
# 将数据存到CSV中
# @author: wangxinbiao
# @date: 2023-11-02 14:42:01
# modify history
# ==== 2023-11-02 14:42:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###
async def save_csv(opt={}):
    file_name = opt['file_name']
    phase_value = opt['phase_value']
    data = opt['data']

    csv_file_path = await file_utils.get_temp_file_path()

    # 如果文件夹不存在，则创建
    directory_path = csv_file_path + phase_value
    if not os.path.exists(directory_path):
        os.makedirs(directory_path)

    file_path = directory_path + '/' + file_name

    with open(file_path, 'w', newline='') as file:
        writer = csv.writer(file)

        writer.writerow(['prompt'])

        for row in data:
            writer.writerow([row])

    return file_path
