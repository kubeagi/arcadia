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
import traceback

from kubernetes import config
from kubernetes.client import CoreV1Api, CustomObjectsApi

from common import log_tag_const

from .custom_resources import (arcadia_resource_datasets,
                               arcadia_resource_datasources,
                               arcadia_resource_embedding,
                               arcadia_resource_models,
                               arcadia_resource_versioneddatasets)

logger = logging.getLogger(__name__)


class NamespacedName:
    def __init__(self, namespace, name):
        self.namespace = namespace
        self.name = name

    def get_namespace(self):
        return self.namespace

    def get_name(self):
        return self.name


class KubeEnv:
    def __init__(self):
        self.pod_namespace = os.environ.get("POD_NAMESPACE")
        self.kubeconfig_path = os.environ.get("KUBECONFIG")
        if self.kubeconfig_path:
            config.load_kube_config(self.kubeconfig_path)
            logger.debug(
                f"{log_tag_const.KUBERNETES} Load kubeconfig from {self.kubeconfig_path}"
            )
        else:
            try:
                logger.debug(
                    f"{log_tag_const.KUBERNETES} Try to loading kubeconfig from in cluster config"
                )
                config.load_incluster_config()
            except config.ConfigException:
                logger.error(
                    f"{log_tag_const.KUBERNETES} There is an error ",
                    f"when load kubeconfig from in cluster config.\n {traceback.format_exc()}",
                )
                raise RuntimeError(
                    "".join(
                        [
                            "Failed to load incluster config. ",
                            "Make sure the code is running inside a Kubernetes cluster.",
                        ]
                    )
                )

    def list_datasources(self, namespace: str, **kwargs):
        return CustomObjectsApi().list_namespaced_custom_object(
            arcadia_resource_datasources.get_group(),
            arcadia_resource_datasources.get_version(),
            namespace,
            arcadia_resource_datasources.get_name(),
            **kwargs,
        )

    def list_datasets(self, namespace: str, **kwargs):
        return CustomObjectsApi().list_namespaced_custom_object(
            arcadia_resource_datasets.get_group(),
            arcadia_resource_datasets.get_version(),
            namespace,
            arcadia_resource_datasets.get_name(),
            **kwargs,
        )

    def list_versioneddatasets(self, namespace: str, **kwargs):
        return CustomObjectsApi().list_namespaced_custom_object(
            arcadia_resource_versioneddatasets.get_group(),
            arcadia_resource_versioneddatasets.get_version(),
            namespace,
            arcadia_resource_versioneddatasets.get_name(),
            **kwargs,
        )

    def patch_versioneddatasets_status(self, namespace: str, name: str, status: any):
        CustomObjectsApi().patch_namespaced_custom_object_status(
            arcadia_resource_versioneddatasets.get_group(),
            arcadia_resource_versioneddatasets.get_version(),
            namespace,
            arcadia_resource_versioneddatasets.get_name(),
            name,
            status,
        )

    def get_versioneddatasets_status(self, namespace: str, name: str):
        return CustomObjectsApi().get_namespaced_custom_object_status(
            arcadia_resource_versioneddatasets.get_group(),
            arcadia_resource_versioneddatasets.get_version(),
            namespace,
            arcadia_resource_versioneddatasets.get_name(),
            name,
        )

    def get_versionedmodels_status(self, namespace: str, name: str):
        return CustomObjectsApi().get_namespaced_custom_object_status(
            arcadia_resource_models.get_group(),
            arcadia_resource_models.get_version(),
            namespace,
            arcadia_resource_models.get_name(),
            name,
        )

    def read_namespaced_config_map(self, namespace: str, name: str):
        return CoreV1Api().read_namespaced_config_map(namespace=namespace, name=name)

    def get_secret_info(self, namespace: str, name: str):
        """Get the secret info."""
        data = CoreV1Api().read_namespaced_secret(namespace=namespace, name=name)
        return data.data

    def get_datasource_object(self, namespace: str, name: str):
        """Get the Datasource object."""
        return CustomObjectsApi().get_namespaced_custom_object(
            group=arcadia_resource_models.get_group(),
            version=arcadia_resource_models.get_version(),
            namespace=namespace,
            plural=arcadia_resource_datasources.get_name(),
            name=name,
        )

    def get_versionedembedding_status(self, namespace: str, name: str):
        return CustomObjectsApi().get_namespaced_custom_object_status(
            arcadia_resource_embedding.get_group(),
            arcadia_resource_embedding.get_version(),
            namespace,
            arcadia_resource_embedding.get_name(),
            name,
        )
