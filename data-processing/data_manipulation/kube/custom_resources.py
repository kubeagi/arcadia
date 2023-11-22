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
# CRD Versioneddataset
arcadia_resource_versioneddatasets = CustomResource(
    arcadia_group, "versioneddatasets")
