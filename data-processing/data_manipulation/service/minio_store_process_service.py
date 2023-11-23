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

import io
import logging
import os

import pandas as pd
from minio import Minio
from minio.commonconfig import Tags
from minio.error import S3Error
from sanic.response import json, raw

from common import config
from db import data_process_task
from file_handle import csv_handle, pdf_handle
# from kube import client
from utils import file_utils, minio_utils

logger = logging.getLogger('minio_store_process_service')

# kube = client.KubeEnv()

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


async def text_manipulate(request, opt={}):

    request_json = request.json
    bucket_name = request_json['bucket_name']
    support_type = request_json['data_process_config_info']
    file_names = request_json['file_names']

    # minio 数据集统一前缀
    minio_dataset_prefix = config.minio_dataset_prefix

    folder_prefix = minio_dataset_prefix + '/' + request_json['pre_data_set_name'] + '/' + request_json['pre_data_set_version']

    # create minio client
    minio_client = await minio_utils.create_client()

    # 将文件都下载到本地
    for file_name in file_names:
        await download({
            'minio_client': minio_client,
            'bucket_name': bucket_name,
            'folder_prefix': folder_prefix,
            'file_name': file_name['name']
        })

    # 文件处理
    for item in file_names:
        file_name = item['name']
        file_extension = file_name.split('.')[-1].lower()
        if file_extension in ['csv']:
            # 处理CSV文件
            result = await csv_handle.text_manipulate({
                'file_name': file_name,
                'support_type': support_type
            })

        elif file_extension in ['pdf']:
            # 处理PDF文件
            result = await pdf_handle.text_manipulate(request, {
                'file_name': file_name,
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
    if any(d.get('type') == 'qa_split' for d in support_type):
        tags["object_type"] = "QA"
    else:
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
        local_file_path = remove_file_path + 'original/' + item['name']
        await file_utils.delete_file(local_file_path)

    # 数据库更新任务状态
    await data_process_task.update_status_by_id({
        'id': opt['id'],
        'status': 'process_complete',
        'conn': opt['conn']
    })

    # 更新数据集CR状态
    # kube.patch_versioneddatasets_status(request_json[])

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
    folder_prefix = opt['folder_prefix']
    minio_client = opt['minio_client']
    bucket_name = opt['bucket_name']
    file_name = opt['file_name']
    file_path = await file_utils.get_temp_file_path()

    # 如果文件夹不存在，则创建
    directory_path = file_path + 'original'
    if not os.path.exists(directory_path):
        os.makedirs(directory_path)

    file_path = directory_path + '/' + file_name

    minio_client.fget_object(bucket_name, folder_prefix + '/' + file_name, file_path)



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
            minio_object_name = os.path.join(
                minio_prefix, os.path.relpath(local_file_path, local_folder)
            )

            try:
                minio_client.fput_object(
                    minio_bucket, minio_object_name, local_file_path, tags=tags
                )
                
                # 删除本地文件
                await file_utils.delete_file(local_file_path)
            except S3Error as e:
                logger.error(
                    f"Error uploading {minio_object_name} to {minio_bucket}: {e}")
