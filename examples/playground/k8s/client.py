import os
from kubernetes import config
from kubernetes.client import CoreV1Api, CustomObjectsApi


class NamespacedName:
    def __init__(self, namespace, name):
        self.namespace = namespace
        self.name = name

    def get_namespace(self):
        return self.namespace

    def get_name(self):
        return self.name


class KubeEnv:
    def __init__(self, namespace: str):
        # configure namespace
        self.namespace = os.environ.get('NAMESPACE')
        if namespace != '':
            self.namespace == namespace

        # configure kubeconfig
        self.kubeconfig_path = os.environ.get('KUBECONFIG')
        if self.kubeconfig_path:
            config.load_kube_config(self.kubeconfig_path)
        else:
            try:
                print("try load kubeconfig from incluster config")
                config.load_incluster_config()
            except config.ConfigException:
                raise RuntimeError(
                    "Failed to load incluster config. Make sure the code is running inside a Kubernetes cluster.")
