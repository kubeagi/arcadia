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

from utils import date_time_utils

from . import client

logger = logging.getLogger(__name__)


def update_dataset_k8s_cr(namespace, version_data_set_name, reason, message):
    """Update the condition info for the dataset.

    namespace: namespace;
    version_data_set_name: version dataset name;
    reason: the update reason;
    """
    try:
        kube = client.KubeEnv()

        one_cr_datasets = kube.get_versioneddatasets_status(
            namespace, version_data_set_name
        )

        conditions = one_cr_datasets["status"]["conditions"]
        now_utc_str = date_time_utils.now_utc_str()

        found_index = None
        for i in range(len(conditions)):
            item = conditions[i]
            if item["type"] == "DataProcessing":
                found_index = i
                break

        if found_index is None:
            conditions.append(
                {
                    "lastTransitionTime": now_utc_str,
                    "reason": reason,
                    "status": "True",
                    "type": "DataProcessing",
                    "message": message,
                }
            )
        else:
            conditions[found_index] = {
                "lastTransitionTime": now_utc_str,
                "reason": reason,
                "status": "True",
                "type": "DataProcessing",
                "message": message,
            }

        kube.patch_versioneddatasets_status(
            namespace, version_data_set_name, {"status": {"conditions": conditions}}
        )

        return {"status": 200, "message": "更新数据集状态成功", "data": ""}
    except Exception as ex:
        logger.error(str(ex))
        return {"status": 400, "message": "更新数据集状态失败", "data": ""}
