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


class GroupVersion:
    def __init__(self, name, version):
        self.name = name
        self.version = version


class CustomResource:
    def __init__(self, group_version, name):
        self.group_version = group_version
        self.name = name

    def get_group(self):
        return self.group_version.name

    def get_version(self):
        return self.group_version.version

    def get_name(self):
        return self.name


# Arcadia
arcadia_group = GroupVersion("arcadia.kubeagi.k8s.com.cn", "v1alpha1")
# CRD Datasource
arcadia_resource_datasources = CustomResource(arcadia_group, "datasources")
# CRD Dataset
arcadia_resource_datasets = CustomResource(arcadia_group, "datasets")
# CRD LLM
arcadia_resource_models = CustomResource(arcadia_group, "llms")
# CRD Versioneddataset
arcadia_resource_versioneddatasets = CustomResource(arcadia_group, "versioneddatasets")
# CRD Embedding
arcadia_resource_embedding = CustomResource(arcadia_group, "embedders")
