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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WorkerSpec defines the desired state of Worker
type WorkerSpec struct {
	CommonSpec `json:",inline"`

	// Type for this worker
	Type WorkerType `json:"type,omitempty"`

	// Model this worker wants to use
	Model *TypedObjectReference `json:"model"`

	// Resource request&limits including
	// - CPU or GPU
	// - Memory
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Storage claimed to store model files
	Storage *corev1.PersistentVolumeClaimSpec `json:"storage,omitempty"`
}

// WorkerStatus defines the observed state of Worker
type WorkerStatus struct {
	// PodStatus is the observed stated of Worker pod
	// +optional
	PodStatus corev1.PodStatus `json:"podStatus,omitempty"`

	// ConditionedStatus is the current status
	ConditionedStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="type",type=string,JSONPath=`.spec.type`
//+kubebuilder:printcolumn:name="model",type=string,JSONPath=`.spec.model.name`

// Worker is the Schema for the workers API
type Worker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkerSpec   `json:"spec,omitempty"`
	Status WorkerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WorkerList contains a list of Worker
type WorkerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Worker `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Worker{}, &WorkerList{})
}
