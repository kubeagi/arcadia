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

import base64
import logging
import traceback

import yaml

from . import client

logger = logging.getLogger(__name__)


def get_minio_config_in_k8s_configmap(namespace, config_map_name):
    """Get the MinIO config info in the configmap.

    namespace: namespace;
    config_map_name: config map name
    """
    try:
        kube = client.KubeEnv()

        config_map = kube.read_namespaced_config_map(
            namespace=namespace, name=config_map_name
        )

        config = config_map.data.get("config")

        json_data = yaml.safe_load(config)

        datasource = json_data["systemDatasource"]

        minio_cr_object = kube.get_datasource_object(
            namespace=datasource["namespace"], name=datasource["name"]
        )

        minio_api_url = minio_cr_object["spec"]["endpoint"]["url"]

        minio_secure = True
        insecure = minio_cr_object["spec"]["endpoint"].get("insecure")
        if insecure is None:
            minio_secure = True
        elif str(insecure).lower() == "true":
            minio_secure = False

        secret_info = kube.get_secret_info(
            namespace=namespace,
            name=minio_cr_object["spec"]["endpoint"]["authSecret"]["name"],
        )

        return {
            "minio_api_url": minio_api_url,
            "minio_secure": minio_secure,
            "minio_access_key": base64.b64decode(secret_info["rootUser"]).decode(
                "utf-8"
            ),
            "minio_secret_key": base64.b64decode(secret_info["rootPassword"]).decode(
                "utf-8"
            ),
        }
    except Exception as ex:
        logger.error(
            "".join(
                [
                    f"Can not get the MinIO config info. The error is: \n",
                    f"{traceback.format_exc()}\n",
                ]
            )
        )

        return None
