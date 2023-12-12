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


import io
import logging
import os

import pandas as pd
from common import log_tag_const
from common.config import config
from data_store_clients import minio_store_client
from database_operate import data_process_db_operate
from file_handle import csv_handle, pdf_handle
from kube import dataset_cr
from utils import file_utils

logger = logging.getLogger(__name__)


def text_manipulate(
    req_json,
    pool,
    id,
):
    """Manipulate the text content.
    
    req_json is a dictionary object. 
    """
    
    bucket_name = req_json['bucket_name']
    support_type = req_json['data_process_config_info']
    file_names = req_json['file_names']

    # minio 数据集统一前缀
    minio_dataset_prefix = config.minio_dataset_prefix

    folder_prefix = '/'.join([
        minio_dataset_prefix,
        req_json['pre_data_set_name'],
        req_json['pre_data_set_version']
    ])

    # get a minio client
    minio_client = minio_store_client.get_minio_client()

    # 将文件都下载到本地
    for file_name in file_names:
        minio_store_client.download(
            minio_client,
            bucket_name=bucket_name,
            folder_prefix=folder_prefix,
            file_name=file_name['name']
        )

    # 文件处理
    # 存放每个文件对应的数据量
    data_volumes_file = []
    
    for item in file_names:
        file_name = item['name']
        file_extension = file_name.split('.')[-1].lower()
        if file_extension in ['csv']:
            # 处理CSV文件
            result = csv_handle.text_manipulate({
                'file_name': file_name,
                'support_type': support_type
            })

        elif file_extension in ['pdf']:
            # 处理PDF文件
            result = pdf_handle.text_manipulate(
                chunk_size=req_json.get('chunk_size'),
                chunk_overlap=req_json.get('chunk_overlap'),
                file_name=file_name,
                support_type=support_type,
                conn_pool=pool,
                task_id=id,
                create_user=req_json['creator']
            )

        data_volumes_file.append(result['data'])

    # 将清洗后的文件上传到MinIO中
    # 上传final文件夹下的文件，并添加tag
    file_path = file_utils.get_temp_file_path()
    minio_store_client.upload_files_to_minio_with_tags(
        minio_client=minio_client,
        local_folder=file_path + 'final',
        minio_bucket=bucket_name,
        minio_prefix=folder_prefix,
        support_type=support_type,
        data_volumes_file=data_volumes_file
    )

    # 将本地临时文件删除
    for item in file_names:
        remove_file_path = file_utils.get_temp_file_path()
        local_file_path = remove_file_path + 'original/' + item['name']
        file_utils.delete_file(local_file_path)

    # 数据库更新任务状态
    update_params = {
        'id': id,
        'status': 'process_complete',
        'create_user': req_json['creator']
    }
    data_process_db_operate.update_status_by_id(
        update_params,
        pool=pool
    )

    # 更新数据集CR状态
    dataset_cr.update_dataset_k8s_cr(
        bucket_name=req_json['bucket_name'],
        version_data_set_name=req_json['version_data_set_name'],
        reason='process_complete'
    )

    return {
        'status': 200,
        'message': '',
        'data': ''
    }







