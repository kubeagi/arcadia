/*
Copyright 2023 KubeAGI.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type FileGroup struct {
	// From defines the datasource which provides this `File`
	Datasource *TypedObjectReference `json:"datasource"`
	// Paths defines the detail paths to get objects from above datasource
	Paths []string `json:"paths"`
}

// VersionedDatasetSpec defines the desired state of VersionedDataset
type VersionedDatasetSpec struct {
	// Dataset which this `VersionedDataset` belongs to
	Dataset *TypedObjectReference `json:"dataset"`

	// Version
	Version string `json:"version"`

	// FileGroups included in this `VersionedDataset`
	// Grouped by Datasource
	FileGroups []FileGroup `json:"fileGroups,omitempty"`

	// +kubebuilder:validation:Enum=0;1
	// +kubebuilder:default=0
	Released uint8 `json:"released"`
}

type DatasourceFileStatus struct {
	DatasourceName      string        `json:"datasourceName"`
	DatasourceNamespace string        `json:"datasourceNamespace"`
	Status              []FileDetails `json:"status,omitempty"`
}

// VersionedDatasetStatus defines the observed state of VersionedDataset
type VersionedDatasetStatus struct {
	// ConditionedStatus is the current status
	ConditionedStatus `json:",inline"`

	// DatasourceFiles record the process and results of file processing for each data source
	DatasourceFiles []DatasourceFileStatus `json:"datasourceFiles,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="dataset",type=string,JSONPath=`.spec.dataset.name`
//+kubebuilder:printcolumn:name="version",type=string,JSONPath=`.spec.version`

// VersionedDataset is the Schema for the versioneddatasets API
type VersionedDataset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VersionedDatasetSpec   `json:"spec,omitempty"`
	Status VersionedDatasetStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VersionedDatasetList contains a list of VersionedDataset
type VersionedDatasetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VersionedDataset `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VersionedDataset{}, &VersionedDatasetList{})
}
