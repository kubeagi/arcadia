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
# 数据处理
# @author: wangxinbiao
# @date: 2023-11-21 11:35:01
# modify history
# ==== 2023-11-21 11:35:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###

import asyncio
import logging

import ulid
from sanic.response import json

from db import data_process_task
from service import minio_store_process_service, dataset_service

logger = logging.getLogger('data_process_service')

###
# 分页查询列表
# @author: wangxinbiao
# @date: 2023-11-21 11:31:01
# modify history
# ==== 2023-11-21 11:31:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###


async def list_by_page(request, opt={}):
    return await data_process_task.list_by_page(request, opt)

###
# 查询列表总记录数
# @author: wangxinbiao
# @date: 2023-11-21 15:45:01
# modify history
# ==== 2023-11-21 15:45:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###


async def list_by_count(request, opt={}):
    return await data_process_task.list_by_count(request, opt)

###
# 新增
# @author: wangxinbiao
# @date: 2023-11-21 15:45:01
# modify history
# ==== 2023-11-21 15:45:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###


async def add(request, opt={}):
    id = ulid.ulid()
    opt['id'] = id
    res = await data_process_task.add(request, opt)

    if res['status'] == 200:
        request_json = request.json

        # 更新数据集状态
        update_dataset = await dataset_service.update_dataset_k8s_cr({
            'bucket_name': request_json['bucket_name'],
            'version_data_set_name': request_json['version_data_set_name'],
            'reason': 'processing'
        })

        if update_dataset['status'] != 200:
            return json(update_dataset)

        # 进行数据处理
        asyncio.create_task(
            minio_store_process_service.text_manipulate(request, opt)
        )
    
    return json(res)


###
# 删除
# @author: wangxinbiao
# @date: 2023-11-21 17:47:01
# modify history
# ==== 2023-11-21 17:47:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###


async def delete_by_id(request, opt={}):

    return await data_process_task.delete_by_id(request, opt)

