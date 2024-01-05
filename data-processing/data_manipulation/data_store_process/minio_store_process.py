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
import ulid
import traceback
import ujson

import pandas as pd
from common import log_tag_const, const
from common.config import config
from data_store_clients import minio_store_client
from database_operate import (data_process_db_operate,
                              data_process_document_db_operate,
                              data_process_detail_db_operate,
                              data_process_detail_preview_db_operate,
                              data_process_log_db_operate,
                              data_process_stage_log_db_operate)
from file_handle import csv_handle, pdf_handle, word_handle
from kube import dataset_cr
from utils import file_utils, date_time_utils, json_utils
from pathlib import Path

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

    # 新增数据处理任务日志
    log_id = ulid.ulid()
    insert_log_item = {
        'id': log_id,
        'task_id': id,
        'type': 'NOW',
        'error_msg': '',
        'creator': req_json.get('creator')
    }
    data_process_log_db_operate.add(
        insert_log_item,
        pool=pool
    )

    # update the dataset status
    update_dataset = _update_dateset_status(
        bucket_name=req_json['bucket_name'],
        version_data_set_name=req_json['version_data_set_name'],
        reason='processing',
        task_id=id,
        log_id=log_id,
        creator=req_json.get('creator')
    )
    if update_dataset['status'] != 200:
        return update_dataset

    # minio 数据集统一前缀
    minio_dataset_prefix = config.minio_dataset_prefix

    folder_prefix = '/'.join([
        minio_dataset_prefix,
        req_json['pre_data_set_name'],
        req_json['pre_data_set_version']
    ])

    # get a minio client
    minio_client = minio_store_client.get_minio_client()

    # 将文件信息存入data_process_task_document表中
    for file_name in file_names:
        # 新增文档处理进度信息
        document_id = ulid.ulid()
        extension = file_utils.get_file_extension(file_name['name'])
        document_insert_item = {
            'id': document_id,
            'task_id': id,
            'file_name': file_name['name'],
            'status': 'not_start',
            'progress': '0',
            'creator': req_json['creator'],
            'from_source_type': 'MinIO',
            'from_source_path': config.minio_api_url,
            'document_type': extension
        }
        data_process_document_db_operate.add(
            document_insert_item,
            pool=pool
        )
        file_name['document_id']=document_id

    # 文件处理
    task_status = 'process_complete'
    error_msg = ''
    # 存放每个文件对应的数据量
    data_volumes_file = []
    
    for item in file_names:
        result = None
        file_name = item['name']

        # 将文件下载到本地
        minio_store_client.download(
            minio_client,
            bucket_name=bucket_name,
            folder_prefix=folder_prefix,
            file_name=file_name
        )

        # 新增阶段性日志-开始
        start_stage_detail = _get_stage_detail(
            req_json,
            pool=pool,
            task_id=id,
            document_id=item.get('document_id'),
            stage='start'
        )
        insert_stage_log_params = {
            'task_id': id,
            'log_id': log_id,
            'file_name': file_name,
            'stage_name': 'start',
            'stage_status': 'success',
            'stage_detail': start_stage_detail.get('data'),
            'creator': req_json.get('creator')
        }
        data_process_stage_log_db_operate.insert(
            insert_stage_log_params,
            pool=pool
        )

        file_extension = file_utils.get_file_extension(file_name)
        if file_extension in ['pdf']:
            # 处理PDF文件
            result = pdf_handle.text_manipulate(
                chunk_size=req_json.get('chunk_size'),
                chunk_overlap=req_json.get('chunk_overlap'),
                file_name=file_name,
                document_id=item.get('document_id'),
                support_type=support_type,
                conn_pool=pool,
                task_id=id,
                create_user=req_json['creator']
            )
        
        elif file_extension in ['docx']:
            # 处理.docx文件
            result = word_handle.docx_text_manipulate(
                chunk_size=req_json.get('chunk_size'),
                chunk_overlap=req_json.get('chunk_overlap'),
                file_name=file_name,
                document_id=item.get('document_id'),
                support_type=support_type,
                conn_pool=pool,
                task_id=id,
                create_user=req_json['creator']
            )
        
        # 将下载的本地文件删除
        _remove_local_file(file_name)

        # 判断是否存在qa拆分
        has_qa_split = any(item.get('type') == 'qa_split' for item in support_type)

        if result is None:
            logger.error(''.join([
                f"{log_tag_const.MINIO_STORE_PROCESS} The file type is not supported \n",
                f"The current file type is: {file_extension}"
            ]))
            # 任务失败
            task_status = 'process_fail'
            error_msg = f"{file_extension} file type is not currently supported."
            break

                # 新增阶段性日志-clean
        clean_stage_detail=_get_stage_detail(
            req_json,
            pool=pool,
            task_id=id,
            document_id=item.get('document_id'),
            stage='clean',
            file_name=file_name
        )
        if clean_stage_detail.get('status') == 200:
            insert_stage_log_params = {
                'task_id': id,
                'log_id': log_id,
                'file_name': file_name,
                'stage_name': 'clean',
                'stage_status': 'success',
                'stage_detail': clean_stage_detail.get('data'),
                'creator': req_json.get('creator')
            }
            data_process_stage_log_db_operate.insert(
                insert_stage_log_params,
                pool=pool
            )

        # 新增阶段性日志-privacy
        privacy_stage_detail=_get_stage_detail(
            req_json,
            pool=pool,
            task_id=id,
            document_id=item.get('document_id'),
            stage='privacy',
            file_name=file_name
        )
        if privacy_stage_detail.get('status') == 200:
            insert_stage_log_params = {
                'task_id': id,
                'log_id': log_id,
                'file_name': file_name,
                'stage_name': 'privacy',
                'stage_status': 'success',
                'stage_detail': privacy_stage_detail.get('data'),
                'creator': req_json.get('creator')
            }
            data_process_stage_log_db_operate.insert(
                insert_stage_log_params,
                pool=pool
            )
        
        if result.get('status') != 200:
            # 任务失败
            logger.error(''.join([
                f"{log_tag_const.MINIO_STORE_PROCESS} Data process fail \n",
                f"The file name: {file_name}\n",
                f"The error is: {result.get('message')}\n"
            ]))
            task_status = 'process_fail'
            error_msg = result.get('message')

            # 新增阶段性日志-qa_split
            if has_qa_split:
                _get_qa_stage_detail(
                    task_id=id,
                    log_id=log_id,
                    status='fail',
                    file_name=file_name,
                    creator=req_json.get('creator'),
                    result=result,
                    pool=pool
                )
            break

        data_volumes_file.append(result['data'])

        # 新增阶段性日志-qa_split
        if has_qa_split:
            _get_qa_stage_detail(
                task_id=id,
                log_id=log_id,
                status='success',
                file_name=file_name,
                creator=req_json.get('creator'),
                result=result,
                pool=pool
            )

    # 新增阶段性日志-finish
    finish_now = date_time_utils.now_str()
    finish_stage_detail=f"{finish_now} Task Finished!!!"
    
    insert_stage_log_params = {
        'task_id': id,
        'log_id': log_id,
        'file_name': file_name,
        'stage_name': 'finish',
        'stage_status': 'success',
        'stage_detail': finish_stage_detail,
        'creator': req_json.get('creator')
    }
    data_process_stage_log_db_operate.insert(
        insert_stage_log_params,
        pool=pool
    )
    
    # insert QA list to detail preview
    logger.debug(f"{log_tag_const.MINIO_STORE_PROCESS} Insert QA list for detail preview.")
    list_qa_params = {
        'task_id': id
    }
    list_qa_res = data_process_detail_db_operate.top_n_list_qa_for_preview(
        list_qa_params,
        pool=pool
    )

    for item in list_qa_res.get('data'):
        item['transform_type']='qa_split'
        item['pre_content']=item['question']
        item['post_content']=item['answer']
        data_process_detail_preview_db_operate.insert(
            item,
            pool=pool
        )

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

    # update the dataset status
    update_dataset = _update_dateset_status(
        bucket_name=req_json['bucket_name'],
        version_data_set_name=req_json['version_data_set_name'],
        reason=task_status,
        task_id=id,
        log_id=log_id,
        creator=req_json.get('creator')
    )
    if update_dataset['status'] != 200:
        return update_dataset

    # 更新数据处理任务日志
    update_log_item = {
        'id': log_id,
        'status': task_status,
        'error_msg': error_msg,
        'creator': req_json['creator']
    }
    data_process_log_db_operate.update_status_by_id(
        update_log_item,
        pool=pool
    )

    # 数据库更新任务状态
    update_params = {
        'id': id,
        'current_log_id': log_id,
        'status': task_status,
        'user': req_json['creator']
    }
    data_process_db_operate.update_status_by_id(
        update_params,
        pool=pool
    )

    return {
        'status': 200,
        'message': '',
        'data': ''
    }


def _remove_local_file(file_name):
    try:
        remove_file_path = file_utils.get_temp_file_path()
        local_file_path = remove_file_path + 'original/' + file_name
        file_utils.delete_file(local_file_path)
        return {
            'status': 200,
            'message': '删除成功',
            'data': ''
        }
    except Exception as ex:
        logger.error(''.join([
            f"{log_tag_const.MINIO_STORE_PROCESS} remove local file fail \n",
            f"the error. \n{traceback.format_exc()}"
        ]))
        return {
            'status': 400,
            'message': str(ex),
            'data': traceback.format_exc()
        }

def _update_dateset_status(
    bucket_name,
    version_data_set_name,
    reason,
    task_id,
    log_id,
    creator
):
    update_dataset = dataset_cr.update_dataset_k8s_cr(
        bucket_name=bucket_name,
        version_data_set_name=version_data_set_name,
        reason=reason
    )

    if update_dataset['status'] != 200:
        # 更新数据处理任务日志
        update_log_item = {
            'id': log_id,
            'status': 'process_fail',
            'error_msg': update_dataset.get('message'),
            'creator': creator
        }
        data_process_log_db_operate.update_status_by_id(
            update_log_item,
            pool=pool
        )

        # 数据库更新任务状态
        update_params = {
            'id': task_id,
            'current_log_id': log_id,
            'status': 'process_fail',
            'user': creator
        }
        data_process_db_operate.update_status_by_id(
            update_params,
            pool=pool
        )

    return update_dataset


def _get_stage_detail(
    req_json,
    task_id,
    document_id,
    pool,
    stage,
    file_name=None
):
    now = date_time_utils.now_str()
    stage_detail = ''
    operations = req_json.get('data_process_config_info')

    if stage == 'start':
        received_task={
            'task_id': task_id,
            'pre_dataset_name': req_json.get('pre_data_set_name'),
            'pre_dataset_version': req_json.get('pre_data_set_version'),
            'file_names': req_json.get('file_names')
        }

        stage_detail = '\n'.join([
            f"{now} Data Processing Task Starts!!!",
            f"Received Task: {json_utils.dumps(received_task)}",
            f"Operations: {json_utils.dumps(operations)}"
        ])
    elif stage == 'clean':
        clean_stage_detail = _get_stage_detail_for_transform_type(
            task_id=task_id,
            document_id=document_id,
            transform_type=operations,
            support_type=const.clean_support_type,
            pool=pool
        )

        if clean_stage_detail.get('status') != 200:
            return clean_stage_detail

        stage_detail='\n'.join([
            f"{now} Current Execution Stage: {stage}, File Name: {file_name}",
            f"Current Result: {json_utils.dumps(clean_stage_detail.get('data'))}"
        ])
    elif stage == 'privacy':
        privacy_stage_detail = _get_stage_detail_for_transform_type(
            task_id=task_id,
            document_id=document_id,
            transform_type=operations,
            support_type=const.privacy_support_type,
            pool=pool
        )

        if privacy_stage_detail.get('status') != 200:
            return privacy_stage_detail

        stage_detail='\n'.join([
            f"{now} Current Execution Stage: {stage}, File Name: {file_name}",
            f"Current Result: {json_utils.dumps(privacy_stage_detail.get('data'))}"
        ])
    
    return {
        'status': 200,
        'message': '',
        'data': stage_detail
    }


def _get_stage_detail_for_transform_type(
    task_id,
    document_id,
    transform_type,
    support_type,
    pool
):
    """获取阶段详情日志"""
    # 处理结果
    operator_result=[]

    stage_support_type = []
    for item in transform_type:
        if item.get('type') in support_type:
            stage_support_type.append(item.get('type'))

    if len(stage_support_type) == 0:
        return {
            'status': 1000,
            'message': '用户没有选择数据异常清洗',
            'data': ''
        }

    detail_list = _list_for_transform_type(
        task_id=task_id,
        document_id=document_id,
        transform_type=stage_support_type,
        pool=pool
    )
    if len(detail_list.get('data')) == 0:
        for item in stage_support_type:
            operator_result.append({
                'type': item,
                'processed_count': 0
            })
        return {
            'status': 200,
            'message': '',
            'data': {
                'status': 'sucess',
                'operator_count': len(stage_support_type),
                'operator_result': operator_result
            }
        }  

    # 判断是否存在状态为fail的数据
    status='success'
    has_fail = any(item.get('status') == 'fail' for item in detail_list.get('data'))
    if has_fail:
        status='fail'

    for item in stage_support_type:
        list_for_support_type = _list_for_transform_type(
            task_id=task_id,
            document_id=document_id,
            transform_type=[item],
            pool=pool
        )

        # 判断该类型状态为fail的数据
        data_dict = list_for_support_type.get('data')
        has_fail = any(item.get('status') == 'fail' for item in data_dict)
        if has_fail:
            operator_result.append({
                'type': item,
                'processed_count': len(data_dict),
                'message': data_dict[0].get('error_message')
            })
        else:
            operator_result.append({
                'type': item,
                'processed_count': len(data_dict)
            })
    
    current_result = {
        'status': status,
        'operator_count': len(stage_support_type),
        'operator_result': operator_result
    }

    return {
        'status': 200,
        'message': '',
        'data': current_result
    }  

def _get_qa_stage_detail(
    task_id,
    log_id,
    status,
    file_name,
    creator,
    result,
    pool
):
    """获取QA阶段详情日志"""
    now = date_time_utils.now_str()
    current_result = None
    if status == 'fail':
        current_result = {
            'status': status,
            'qa_count': 0,
            'message': result.get('message')
        }
    else:
        current_result = {
            'status': status,
            'qa_count': result.get('data').get('object_count')
        }

    qa_stage_detail='\n'.join([
        f"{now} Current Execution Stage: qa_split, File Name: {file_name}",
        f"Current Result: {json_utils.dumps(current_result)}"
    ])
    
    insert_stage_log_params = {
        'task_id': task_id,
        'log_id': log_id,
        'file_name': file_name,
        'stage_name': 'qa_split',
        'stage_status': status,
        'stage_detail': qa_stage_detail,
        'creator': creator
    }
    data_process_stage_log_db_operate.insert(
        insert_stage_log_params,
        pool=pool
    )

    return {
        'status': 200,
        'message': '',
        'data': current_result
    }  


def _list_for_transform_type(
    task_id,
    document_id,
    transform_type,
    pool
):
    params={
        'task_id': task_id,
        'document_id': document_id,
        'transform_type': transform_type
    }
    return data_process_detail_db_operate.list_for_transform_type(
        params,
        pool=pool
    )
