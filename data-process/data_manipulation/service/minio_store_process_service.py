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
# MinIO
# @author: wangxinbiao
# @date: 2023-11-01 16:43:01
# modify history
# ==== 2023-11-01 16:43:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###

from sanic.response import json, raw
from minio import Minio
from minio.commonconfig import Tags
from minio.error import S3Error
import pandas as pd
import io
import os

import logging

from file_handle import (
    csv_handle
)

from utils import (
    minio_utils,
    file_utils
)

logger = logging.getLogger('minio_store_process_service')

###
# 文本数据处理
# @author: wangxinbiao
# @date: 2023-11-01 10:44:01
# modify history
# ==== 2023-11-01 10:44:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###
async def text_manipulate(request):

    request_json = request.json
    bucket_name = request_json['bucket_name']
    support_type = request_json['type']
    folder_prefix = request_json['folder_prefix']

    # create minio client
    minio_client = await minio_utils.create_client()
    
    # 查询存储桶下的所有对象
    objects = minio_client.list_objects(bucket_name, prefix=folder_prefix)

    # 将文件都下载到本地
    file_names = await download({
        'minio_client': minio_client,
        'bucket_name': bucket_name,
        'folder_prefix': folder_prefix,
        'objects': objects
    })

    # 文件处理
    for item in file_names:
        file_extension = item.split('.')[-1].lower()
        if file_extension in ['csv']:
            # 处理CSV文件
            result = await csv_handle.text_manipulate({
                    'file_name': item,
                    'support_type': support_type
                })

    # 将清洗后的文件上传到MinIO中
    # 上传middle文件夹下的文件，并添加tag
    tags = Tags(for_object=True)
    tags["phase"] = "middle"
    file_path = await file_utils.get_temp_file_path()
    await upload_files_to_minio_with_tags(
        minio_client,
        file_path + 'middle',
        bucket_name,
        minio_prefix=folder_prefix,
        tags=tags
    )

    # 上传final文件夹下的文件，并添加tag
    tags["phase"] = "final"
    await upload_files_to_minio_with_tags(
        minio_client,
        file_path + 'final',
        bucket_name,
        minio_prefix=folder_prefix,
        tags=tags
    )

    # 将本地临时文件删除
    for item in file_names:
        remove_file_path = await file_utils.get_temp_file_path()
        await file_utils.delete_file(remove_file_path + 'original/' + item)
    
    return json({
        'status': 200,
        'message': '',
        'data': ''
    })

###
# 下载MinIO中的文件
# @author: wangxinbiao
# @date: 2023-11-07 17:42:01
# modify history
# ==== 2023-11-07 17:42:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###
async def download(opt={}):
    objects = opt['objects']
    minio_client = opt['minio_client']
    bucket_name = opt['bucket_name']
    folder_prefix = opt['folder_prefix']
    file_names = []
    for obj in objects:
        file_name = obj.object_name[len(folder_prefix):]

        data = minio_client.get_object(bucket_name, obj.object_name)
        df = pd.read_csv(data)

        await csv_handle.save_csv({
            'file_name': file_name,
            'phase_value': 'original',
            'data': df['prompt']
        })
        file_names.append(file_name)

    return file_names

###
# 文件上传至MinIO，添加Tags
# @author: wangxinbiao
# @date: 2023-11-08 14:02:01
# modify history
# ==== 2023-11-08 14:02:01 ====
# author: wangxinbiao
# content:
# 1) 基本功能实现
###
async def upload_files_to_minio_with_tags(minio_client, local_folder, minio_bucket, minio_prefix="", tags=None):
    for root, dirs, files in os.walk(local_folder):
        for file in files:
            local_file_path = os.path.join(root, file)
            minio_object_name = os.path.join(minio_prefix, os.path.relpath(local_file_path, local_folder))
            
            try:
                minio_client.fput_object(minio_bucket, minio_object_name, local_file_path, tags=tags)
                
                # 删除本地文件
                await file_utils.delete_file(local_file_path)
            except S3Error as e:
                logger.error(f"Error uploading {minio_object_name} to {minio_bucket}: {e}")

