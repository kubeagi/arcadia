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
	CommonSpec `json:",inline"`

	// Endpoint defines connection info
	Endpoint Endpoint `json:"endpoint"`

	// OSS defines info for object storage service
	OSS *OSS `json:"oss,omitempty"`

	// RDMA configure RDMA pulls the model file directly from the remote service to the host node.
	RDMA *RDMA `json:"rdma,omitempty"`
}

type RDMA struct {
	// Path on a model storage server, the usual storage path is /path/ns/mode-name, and the path field is /path/, which must end in /.
	// example: /opt/kubeagi/, /opt/, /
	// +kubebuilder:validation:Pattern=(^\/$)|(^\/[a-zA-Z0-9\_.@-]+(\/[a-zA-Z0-9\_.@-]+)*\/$)
	Path string `json:"path"`
}

// OSS defines info for object storage service as datasource
type OSS struct {
	Bucket string `json:"bucket,omitempty"`
	// Object must end with a slash "/" if it is a directory
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
//+kubebuilder:printcolumn:name="display-name",type=string,JSONPath=`.spec.displayName`
//+kubebuilder:printcolumn:name="type",type=string,JSONPath=`.metadata.labels.arcadia\.kubeagi\.k8s\.com\.cn/datasource-type`

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
