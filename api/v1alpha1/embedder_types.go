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

	"github.com/kubeagi/arcadia/pkg/embeddings"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EmbedderSpec defines the desired state of Embedder
type EmbedderSpec struct {
	CommonSpec `json:",inline"`

	// ServiceType indicates the source type of embedding service
	ServiceType embeddings.EmbeddingType `json:"serviceType,omitempty"`

	// Enpoint defines connection info
	Enpoint *Endpoint `json:"endpoint,omitempty"`

	// Worker defines the worker instance that this embedder comes from
	Worker *TypedObjectReference `json:"worker,omitempty"`
}

// EmbeddingsStatus defines the observed state of Embedder
type EmbedderStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// ConditionedStatus is the current status
	ConditionedStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="display-name",type=string,JSONPath=`.spec.displayName`

// Embedder is the Schema for the embeddings API
type Embedder struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EmbedderSpec   `json:"spec,omitempty"`
	Status EmbedderStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EmbedderList contains a list of Embedder
type EmbedderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Embedder `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Embedder{}, &EmbedderList{})
}
