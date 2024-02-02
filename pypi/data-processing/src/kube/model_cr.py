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


def get_spec_for_llms_k8s_cr(name, namespace):
    """get worker model.

    name: model name;
    namespace: namespace;
    """
    try:
        kube = client.KubeEnv()

        one_cr_llm = kube.get_versionedmodels_status(namespace=namespace, name=name)

        provider = one_cr_llm["spec"]

        return {"status": 200, "message": "获取llms中的provider成功", "data": provider}
    except Exception as ex:
        logger.error(str(ex))
        return {"status": 400, "message": "获取llms中的provider失败", "data": ""}


def get_worker_base_url_k8s_configmap(name, namespace):
    """get base url for configmap.

    name: model name;
    namespace: namespace;
    """
    try:
        kube = client.KubeEnv()

        config_map = kube.read_namespaced_config_map(name=name, namespace=namespace)

        config = config_map.data.get("config")

        json_data = yaml.safe_load(config)
        external_api_server = json_data.get("gateway", {}).get("apiServer")

        return external_api_server
    except Exception as ex:
        logger.error(str(ex))
        return None


def get_secret_info(name, namespace):
    """get secret info by name and namespace.

    name: model name;
    namespace: namespace;
    """
    try:
        kube = client.KubeEnv()

        return kube.get_secret_info(namespace=namespace, name=name)
    except Exception as ex:
        logger.error(str(ex))
        return None


def get_llm_qa_retry_count_in_k8s_configmap(namespace, config_map_name):
    """Get the llm QA retry count in the configmap.

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

        return json_data["llm"]["qa_retry_count"]
    except Exception as ex:
        logger.error(
            "".join(
                [
                    f"Can not the llm QA retry count. The error is: \n",
                    f"{traceback.format_exc()}\n",
                ]
            )
        )

        return None

def get_spec_for_embedding_k8s_cr(name, namespace):
    """get embedding.

    name: model name;
    namespace: namespace;
    """
    try:
        kube = client.KubeEnv()

        one_cr_llm = kube.get_versionedembedding_status(namespace=namespace, name=name)

        provider = one_cr_llm["spec"]

        return {"status": 200, "message": "获取embedding中的provider成功", "data": provider}
    except Exception as ex:
        logger.error(str(ex))
        return {"status": 400, "message": "获取embedding中的provider失败", "data": ""}