import os

from custom_resources import (arcadia_resource_datasets,
                              arcadia_resource_datasources,
                              arcadia_resource_versioneddatasets)
from kubernetes import client, config
from kubernetes.client import CustomObjectsApi


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
        self.pod_namespace = os.environ.get('POD_NAMESPACE')
        self.kubeconfig_path = os.environ.get('KUBECONFIG')
        if self.kubeconfig_path:
            print("load kubeconfig from ", self.kubeconfig_path)
            config.load_kube_config(self.kubeconfig_path)
        else:
            try:
                print("try load kubeconfig from incluster config")
                config.load_incluster_config()
            except config.ConfigException:
                raise RuntimeError(
                    "Failed to load incluster config. Make sure the code is running inside a Kubernetes cluster.")

    def list_datasources(self, namespace: str, **kwargs):
        return CustomObjectsApi().list_namespaced_custom_object(
            arcadia_resource_datasources.get_group(),
            arcadia_resource_datasources.get_version(),
            namespace,
            arcadia_resource_datasources.get_name(),
            **kwargs
        )

    def list_datasets(self, namespace: str, **kwargs):
        return CustomObjectsApi().list_namespaced_custom_object(
            arcadia_resource_datasets.get_group(),
            arcadia_resource_datasets.get_version(),
            namespace, arcadia_resource_datasets.get_name(),
            **kwargs
        )

    def list_versioneddatasets(self, namespace: str, **kwargs):
        return CustomObjectsApi().list_namespaced_custom_object(
            arcadia_resource_versioneddatasets.get_group(),
            arcadia_resource_versioneddatasets.get_version(),
            namespace, arcadia_resource_versioneddatasets.get_name(),
            **kwargs
        )
