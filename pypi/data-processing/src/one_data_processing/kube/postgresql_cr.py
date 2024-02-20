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
import traceback

import yaml

from . import client

logger = logging.getLogger(__name__)


def get_postgresql_config_in_k8s_configmap(namespace, config_map_name):
    """Get the PostgreSQL config info in the configmap.

    namespace: namespace;
    config_map_name: config map name
    """
    try:
        kube = client.KubeEnv()

        config_map = kube.read_namespaced_config_map(
            namespace=namespace, name=config_map_name
        )

        config = config_map.data.get("dataprocess")

        json_data = yaml.safe_load(config)

        return json_data["postgresql"]
    except Exception as ex:
        logger.error(
            "".join(
                [
                    f"Can not get the PostgreSQL config info. The error is: \n",
                    f"{traceback.format_exc()}\n",
                ]
            )
        )

        return None
