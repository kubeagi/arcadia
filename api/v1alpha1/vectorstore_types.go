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

// VectorStoreSpec defines the desired state of VectorStore
type VectorStoreSpec struct {
	CommonSpec `json:",inline"`
	// Endpoint represents the endpoint used to communicate with the VectorStore.
	// +kubebuilder:validation:Required
	Endpoint Endpoint `json:"endpoint"`

	// InfrastructureRef is a reference to a provider-specific resource that holds the details
	// for provisioning infrastructure for a VectorStore in said provider.
	InfrastructureRef *TypedObjectReference `json:"infrastructureRef,omitempty"`
}

// VectorStoreStatus defines the observed state of VectorStore
type VectorStoreStatus struct {
	// Phase represents the current phase of cluster actuation.
	// Pending, Running, Terminating, Failed etc.
	// +optional
	Phase string `json:"phase,omitempty"`

	// InfrastructureReady is the state of the infrastructure provider.
	// +optional
	InfrastructureReady bool `json:"infrastructureReady"`

	// ConditionedStatus is the current status
	// +optional
	ConditionedStatus `json:",inline"`

	// ObservedGeneration is the latest generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// VectorStore is the Schema for the vectorstores API
type VectorStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VectorStoreSpec   `json:"spec,omitempty"`
	Status VectorStoreStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VectorStoreList contains a list of VectorStore
type VectorStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VectorStore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VectorStore{}, &VectorStoreList{})
}
