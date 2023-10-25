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

package vectorstore

import (
	"github.com/kubeagi/arcadia/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ChromaSpec defines the desired state of Chroma
type ChromaSpec struct {
	v1alpha1.CommonSpec `json:",inline"`
	// Endpoint represents the endpoint used to communicate with the VectorStore.
	ProxyEndpoint v1alpha1.Endpoint `json:"proxyEndpoint"`
	// +optional
	CollactionName string `json:"collactionName,omitempty"`
	// +optional
	StorePVC string `json:"storePVC,omitempty"`
	// +optional
	Image string `json:"image,omitempty"`
	// +optional
	Deploy *appsv1.DeploymentSpec `json:"deploy,omitempty"`
	// +optional
	Service *corev1.ServiceSpec `json:"service,omitempty"`
}

// ChromaStatus defines the observed state of Chroma
type ChromaStatus struct {
	// ConditionedStatus is the current status
	// +optional
	v1alpha1.ConditionedStatus `json:",inline"`

	// ObservedGeneration is the latest generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Chroma is the Schema for the Chroma API
type Chroma struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChromaSpec   `json:"spec,omitempty"`
	Status ChromaStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ChromaList contains a list of Chroma
type ChromaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Chroma `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Chroma{}, &ChromaList{})
}
