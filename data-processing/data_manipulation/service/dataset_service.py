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

import kubernetes.client
from kubernetes.client.rest import ApiException
from kube import client, custom_resources
from utils import date_time_utils

logger = logging.getLogger('dataset_service')

async def update_dataset_k8s_cr(opt={}):
    try:
        kube = client.KubeEnv()

        cr_datasets = kube.list_versioneddatasets(opt['bucket_name'])

        one_cr_datasets = kube.get_versioneddatasets_status(opt['bucket_name'], opt['version_data_set_name'])

        conditions = one_cr_datasets['status']['conditions']


        found_index = None
        for i in range(len(conditions)):
            item = conditions[i]
            if item['type'] == 'DataProcessing':
                found_index = i
                break

        
        result = None
        if found_index is None:
            conditions.append({
                'lastTransitionTime': date_time_utils.now_utc_str(),
                'reason': opt['reason'],
                'status': "True",
                "type": "DataProcessing"
            })
        else:
            conditions[found_index] = {
                'lastTransitionTime': date_time_utils.now_utc_str(),
                'reason': opt['reason'],
                'status': "True",
                "type": "DataProcessing"
            }

        kube.patch_versioneddatasets_status(
            opt['bucket_name'], 
            opt['version_data_set_name'],
            {
            'status': {
                'conditions': conditions
            }
            }
        )

        return {
            'status': 200,
            'message': '更新数据集状态成功',
            'data': ''
        }
    except Exception as ex:
        logger.error(str(ex))
        return {
            'status': 400,
            'message': '更新数据集状态失败',
            'data': ''
        }