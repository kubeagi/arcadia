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

// DatasourceSpec defines the desired state of Datasource
type DatasourceSpec struct {
	// Creator defines datasource creator(AUTO-FILLED by webhook)
	Creator string `json:"creator,omitempty"`

	// DisplayName defines datasource display name
	DiplayName string `json:"displayName,omitempty"`

	// Description defines datasource description
	Description string `json:"description,omitempty"`

	// Enpoint defines connection info
	Enpoint Endpoint `json:"endpoint"`

	// OSS defines info for object storage service
	OSS *OSS `json:"oss,omitempty"`
}

// OSS defines info for object storage service as datasource
type OSS struct {
	Bucket string `json:"bucket,omitempty"`
	Object string `json:"object,omitempty"`
}

// DatasourceStatus defines the observed state of Datasource
type DatasourceStatus struct {
	// ConditionedStatus is the current status
	ConditionedStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Namespaced

// Datasource is the Schema for the datasources API
type Datasource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatasourceSpec   `json:"spec,omitempty"`
	Status DatasourceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DatasourceList contains a list of Datasource
type DatasourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Datasource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Datasource{}, &DatasourceList{})
}
