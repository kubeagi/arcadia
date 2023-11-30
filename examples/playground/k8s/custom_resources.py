import re
import os
from kubernetes.client import CustomObjectsApi
from .client import KubeEnv


class GroupVersion:
    def __init__(self, name, version):
        self.name = name
        self.version = version


arcadia_group = GroupVersion("arcadia.kubeagi.k8s.com.cn", "v1alpha1")


class CustomResource:
    def __init__(self, group_version, name):
        self.group_version = group_version
        self.name = name

    def get_group(self):
        return self.group_version.name

    def get_version(self):
        return self.group_version.version

    def api_group(self):
        return self.group_version.name + '/' + self.group_version.version

    def get_name(self):
        return self.name


class Datasource(CustomResource):
    def __init__(self):
        self.namespace = os.environ.get('NAMESPACE')
        KubeEnv(namespace=self.namespace)
        super().__init__(arcadia_group, "datasources")

    def get_datasource(self, namespace: str, name: str):
        if namespace == '':
            namespace = self.namespace
        if name == '':
            raise ValueError("Name cannot be empty")
        return CustomObjectsApi().get_namespaced_custom_object(
            self.get_group(),
            self.get_version(),
            namespace,
            "datasources",
            name
        )


class Dataset(CustomResource):
    def __init__(self):
        self.namespace = os.environ.get('NAMESPACE')
        KubeEnv(namespace=self.namespace)

        super().__init__(arcadia_group, "datasets")

    def create(self, namespace: str, name: str, **kwargs):
        if namespace == '':
            namespace = self.namespace
        if name == '':
            raise ValueError("Name cannot be empty")

        dataset = {
            "apiVersion": self.api_group(),
            "kind": "Dataset",
            "metadata": {
                "namespace": namespace,
                "name": name
            },
            "spec": {
                "displayName": name,
                **kwargs
            },
        }

        return CustomObjectsApi().create_namespaced_custom_object(
            self.get_group(),
            self.get_version(),
            namespace,
            "datasets",
            dataset
        )

    def list(self, namespace: str, **kwargs):
        if namespace == '':
            namespace = self.namespace

        ds_list = CustomObjectsApi().list_namespaced_custom_object(
            self.get_group(),
            self.get_version(),
            namespace,
            self.get_name(),
            **kwargs
        )

        datasets = []
        for ds in ds_list.get('items', []):
            datasets.append(ds.get('metadata', {}).get('name'))

        return datasets


class VersionedDataset(CustomResource):
    def __init__(self):
        self.namespace = os.environ.get('NAMESPACE')

        KubeEnv(namespace=self.namespace)
        super().__init__(arcadia_group, "versioneddatasets")

    def create(self, namespace: str, dataset: str, name: str, version: str, **kwargs):
        if namespace == '':
            namespace = self.namespace
        if dataset == '':
            raise ValueError("Dataset cannot be empty")
        if version == '':
            raise ValueError("Version cannot be empty")
        if name == '':
            name = dataset + "-" + version
        versioneddataset = {
            "apiVersion": self.api_group(),
            "kind": "VersionedDataset",
            "metadata": {
                "namespace": namespace,
                "name": name
            },
            "spec": {
                "displayName": name,
                "version": version,
                "dataset": {
                    "kind": "Dataset",
                    "name": dataset
                },
                ** kwargs
            },
        }

        return CustomObjectsApi().create_namespaced_custom_object(
            self.get_group(),
            self.get_version(),
            namespace,
            "versioneddatasets",
            versioneddataset
        )

    def list(self, namespace: str, **kwargs):
        if namespace == '':
            namespace = self.namespace
        vds_list = CustomObjectsApi().list_namespaced_custom_object(
            self.get_group(),
            self.get_version(),
            namespace, self.get_name(),
            **kwargs
        )

        versioneddatasets = []
        for vds in vds_list.get('items', []):
            versioneddatasets.append(vds.get('metadata', {}).get('name'))

        return versioneddatasets


class Knowledgebase(CustomResource):
    def __init__(self):
        self.namespace = os.environ.get('NAMESPACE')
        KubeEnv(namespace=self.namespace)
        super().__init__(arcadia_group, "knowledgebases")


class Vectorstore(CustomResource):
    def __init__(self):
        self.namespace = os.environ.get('NAMESPACE')
        KubeEnv(namespace=self.namespace)
        super().__init__(arcadia_group, "vectorstores")

    def list(self, namespace: str, **kwargs):
        if namespace == '':
            namespace = self.namespace

        vs_list = CustomObjectsApi().list_namespaced_custom_object(
            self.get_group(),
            self.get_version(),
            namespace, self.get_name(),
            **kwargs
        )

        vectorstores = []
        for vs in vs_list.get('items', []):
            vs_name = vs.get('metadata', {}).get('name')
            vectorstores.append(vs.get('metadata', {}).get('name'))

        return vectorstores


class Model(CustomResource):
    def __init__(self):
        self.namespace = os.environ.get('NAMESPACE')
        KubeEnv(namespace=self.namespace)
        super().__init__(arcadia_group, "models")


class Worker(CustomResource):
    def __init__(self):
        self.namespace = os.environ.get('NAMESPACE')
        KubeEnv(namespace=self.namespace)
        super().__init__(arcadia_group, "workers")


class Embedder(CustomResource):
    def __init__(self):
        self.namespace = os.environ.get('NAMESPACE')
        KubeEnv(namespace=self.namespace)
        super().__init__(arcadia_group, "embedders")

    def list(self, namespace: str, **kwargs):
        if namespace == '':
            namespace = self.namespace

        embedder_list = CustomObjectsApi().list_namespaced_custom_object(
            self.get_group(),
            self.get_version(),
            namespace, self.get_name(),
            **kwargs
        )

        embedders = []
        for embedder in embedder_list.get('items', []):
            embedders.append(embedder.get('metadata', {}).get('name'))

        return embedders


class LLM(CustomResource):
    def __init__(self):
        self.namespace = os.environ.get('NAMESPACE')
        KubeEnv(namespace=self.namespace)
        super().__init__(arcadia_group, "llms")
