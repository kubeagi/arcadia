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
import os

from common import log_tag_const
from kube import minio_cr, model_cr, postgresql_cr
from utils.class_utils import Singleton

logger = logging.getLogger(__name__)


class Config(metaclass=Singleton):
    """Configuration class to store the env values."""

    def __init__(self):
        logger.debug(f"{log_tag_const.CONFIG} start to load config file.")
        self.__set_property_value()

    def __set_property_value(self):
        """设置属性的值"""
        # kubernetes
        # namespace
        k8s_pod_namespace = os.getenv("POD_NAMESPACE", "arcadia")
        self.k8s_pod_namespace = k8s_pod_namespace
        # config
        k8s_default_config = os.getenv("DEFAULT_CONFIG", "arcadia-config")
        self.k8s_default_config = k8s_default_config

        minio_config = minio_cr.get_minio_config_in_k8s_configmap(
            namespace=k8s_pod_namespace, config_map_name=k8s_default_config
        )
        if minio_config is None:
            minio_config = {}

        # minio access key
        self.minio_access_key = minio_config.get("minio_access_key")
        # minio secret key
        self.minio_secret_key = minio_config.get("minio_secret_key")
        # minio api url
        self.minio_api_url = minio_config.get("minio_api_url")
        # minio secure
        # if use HTTP, secure = False;
        # if use HTTPS, secure = True;
        self.minio_secure = minio_config.get("minio_secure")
        # minio data set prefix
        self.minio_dataset_prefix = "dataset"

        llm_qa_retry_count = model_cr.get_llm_qa_retry_count_in_k8s_configmap(
            namespace=k8s_pod_namespace, config_map_name=k8s_default_config
        )

        if llm_qa_retry_count is None:
            llm_qa_retry_count = 5

        self.llm_qa_retry_count = int(llm_qa_retry_count)

        # knowledge
        # chunk size
        self.knowledge_chunk_size = 500
        # chunk overlap
        self.knowledge_chunk_overlap = 50

        # backend PostgreSQL
        postgresql_config = postgresql_cr.get_postgresql_config_in_k8s_configmap(
            namespace=k8s_pod_namespace, config_map_name=k8s_default_config
        )
        if postgresql_config is None:
            postgresql_config = {}

        # host
        self.pg_host = postgresql_config.get("host")
        # port
        self.pg_port = postgresql_config.get("port")
        # user
        self.pg_user = postgresql_config.get("user")
        # password
        self.pg_password = postgresql_config.get("password")
        # database name
        self.pg_database = postgresql_config.get("database")


config = Config()
