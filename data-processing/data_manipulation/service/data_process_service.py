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
import ulid
import traceback

from common import log_tag_const
from data_store_process import minio_store_process
from database_operate import data_process_db_operate
from kube import dataset_cr
from parallel import thread_parallel
from utils import date_time_utils

logger = logging.getLogger(__name__)


async def list_by_page(req_json, opt={}):
    """Get the list data for data processing by page"""
    return await data_process_db_operate.list_by_page(req_json, opt)


async def list_by_count(req_json, opt={}):
    """Get count for the list data processing with page"""
    return await data_process_db_operate.list_by_count(req_json, opt)


async def add(req_json, opt={}):
    """Add a new data process"""
    id = ulid.ulid()
    opt['id'] = id
    res = await data_process_db_operate.add(req_json, opt)

    if res['status'] == 200:
        # update the dataset status
        update_dataset = await dataset_cr.update_dataset_k8s_cr({
            'bucket_name': req_json['bucket_name'],
            'version_data_set_name': req_json['version_data_set_name'],
            'reason': 'processing'
        })

        if update_dataset['status'] != 200:
            return update_dataset

        try:
            def execute_text_manipulate_task(loop):
                asyncio.set_event_loop(loop)
                loop.run_until_complete(minio_store_process.text_manipulate(req_json, opt))

            thread_parallel.run_async_background_task(
                execute_text_manipulate_task,
                'execute text manipuate task'
            )
        except Exception as ex:
            logger.error(''.join([
                f"{log_tag_const.MINIO_STORE_PROCESS} There is an error when ",
                f"start to run the minio store process.\n",
                f"{traceback.format_exc()}\n"
            ]))
        

    
    return res


async def delete_by_id(req_json, opt={}):
    """Delete a record with id"""
    return await data_process_db_operate.delete_by_id(req_json, opt)


async def info_by_id(req_json, opt={}):
    """Get a detail info with id"""
    id = req_json['id']

    data = _get_default_data_for_detail()
    data['id'] = id

    # Get the detail info for the data processs task
    detail_info_params = {
        'id': id
    }
    detail_info_res = await data_process_db_operate.info_by_id(detail_info_params, {
        'pool': opt['pool']
    })
    logger.debug(f"{log_tag_const.DATA_PROCESS_DETAIL} The defail info is: \n{detail_info_res}")

    if detail_info_res['status'] == 200 and len(detail_info_res['data']) > 0:
        detail_info_data = detail_info_res['data'][0]
        data['name'] = detail_info_data['name']
        data['status'] = detail_info_data['status']
        data['file_type'] = detail_info_data['file_type']
        data['file_num'] = 0 if detail_info_data['file_names'] is None else len(detail_info_data['file_names'])
        data['pre_dataset_name'] = detail_info_data['pre_data_set_name']
        data['pre_dataset_version'] = detail_info_data['pre_data_set_version']
        data['post_dataset_name'] = detail_info_data['post_data_set_name']
        data['post_dataset_version'] = detail_info_data['post_data_set_version']
        data['start_time'] = detail_info_data['start_datetime']
        data['end_time'] = detail_info_data['end_datetime']

    logger.debug(f"{log_tag_const.DATA_PROCESS_DETAIL} The response data is: \n{data}")

   
    return  {
        'status': 200,
        'message': '',
        'data': data
    }



def _get_default_data_for_detail():
    """Get the data for the detail"""
    return {
        "id": '',
        "name": "数据处理任务1", 
        "status": "processing",
        "file_type": "text",
        "pre_dataset_name": "def",
        "pre_dataset_version": "v1",
        "post_dataset_name": "def",
        "post_dataset_version": "v1",
        "file_num": 20,
        "start_time": date_time_utils.now_str(),
        "end_time": date_time_utils.now_str(),
        "config": [
            {
                "name": "chunk_processing",
                "description": "拆分处理",
                "status": "succeed",
                "children": [
                    {
                        "name'": "qa_split",
                        "enable": "true",
                        "zh_name": "QA拆分",
                        "description": "根据文件中的文章与图表标题，自动将文件做 QA 拆分处理。",
                        "preview": []
                    }
                ]
            },
            {
                "name": "clean",
                "description": "异常清洗配置",
                "status": "succeed",
                "children": [
                    {
                        "name": "remove_invisible_characters",
                        "enable": "true",
                        "zh_name": "移除不可见字符",
                        "description": "移除ASCII中的一些不可见字符, 如0-32 和127-160这两个范围",
                        "preview": [
                            {
                                "file_name": "xxx_001",
                                "content": [
                                    {
                                        "pre": "全然不知对方身份，不断反转的剧情即将揭开层层真相。",
                                        "post": "全然不知对方身份，不断反转的剧情即将揭开层层真相。"
                                    }
                                ]
                            }
                        ]
                    },
                    {
                        "name": "space_standardization",
                        "enable": "true",
                        "zh_name": "空格处理",
                        "description": "将不同的unicode空格比如u2008, 转成正常的空格",
                        "preview": [
                            {
                                "file_name": "xxx_001",
                                "content": [
                                    {
                                        "pre": "全然不知对方身份，不断反转的剧情即将揭开层层真相。",
                                        "post": "全然不知对方身份，不断反转的剧情即将揭开层层真相。"
                                    }
                                ]
                            }
                        ]
                    }
                ]
            }
        ]
    }