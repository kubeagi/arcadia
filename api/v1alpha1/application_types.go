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

// ApplicationSpec defines the desired state of Application
type ApplicationSpec struct {
	Nodes       []Node       `json:"nodes,omitempty"`
	Connections []Connection `json:"connections,omitempty"`
}

type Node struct {
	Name string `json:"name"`
	// LLMRef is a reference to a llm.
	// +optional
	LLMRef *TypedObjectReference `json:"llmRef,omitempty"`

	// LLMSpec is a specification of a llm.
	// +optional
	LLMSpec *LLMSpec `json:"llmSpec,omitempty"`

	// LLMRef is a reference to a llm.
	// +optional
	EmbedderRef *TypedObjectReference `json:"embedderRef,omitempty"`

	// EmbedderSpec is a specification of a Embedder.
	// +optional
	EmbedderSpec *EmbedderSpec `json:"EmbedderSpec,omitempty"`

	// LLMRef is a reference to a llm.
	// +optional
	VectorStoreRef *TypedObjectReference `json:"vectorStoreRef,omitempty"`

	// VectorStoreSpec is a specification of a VectorStore.
	// +optional
	VectorStoreSpec *VectorStoreSpec `json:"VectorStoreSpec,omitempty"`

	// Chain is a reference to a chain.
	// +optional
	Chain Chain `json:"chain,omitempty"`
}

type Connection struct {
	SourceNode string `json:"sourceNode"`
	TargetNode string `json:"targetNode"`
}

type Chain struct {
	LLMChain LLMChain `json:"llmChain,omitempty"`
}

type LLMChain struct {
}

// ApplicationStatus defines the observed state of Application
type ApplicationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Application is the Schema for the applications API
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec,omitempty"`
	Status ApplicationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ApplicationList contains a list of Application
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}
