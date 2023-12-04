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

	node "github.com/kubeagi/arcadia/api/app-node"
	"github.com/kubeagi/arcadia/api/base/v1alpha1"
)

// KnowledgeBaseRetrieverSpec defines the desired state of KnowledgeBaseRetriever
type KnowledgeBaseRetrieverSpec struct {
	v1alpha1.CommonSpec `json:",inline"`
	Input               Input  `json:"input,omitempty"`
	Output              Output `json:"output,omitempty"`
}

type Input struct {
	node.KnowledgeBaseRef `json:",inline"`
}

type Output struct {
	node.CommonOrInPutOrOutputRef `json:",inline"`
}

// KnowledgeBaseRetrieverStatus defines the observed state of KnowledgeBaseRetriever
type KnowledgeBaseRetrieverStatus struct {
	// ObservedGeneration is the last observed generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// ConditionedStatus is the current status
	v1alpha1.ConditionedStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// KnowledgeBaseRetriever is the Schema for the KnowledgeBaseRetriever API
type KnowledgeBaseRetriever struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KnowledgeBaseRetrieverSpec   `json:"spec,omitempty"`
	Status KnowledgeBaseRetrieverStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KnowledgeBaseRetrieverList contains a list of KnowledgeBaseRetriever
type KnowledgeBaseRetrieverList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KnowledgeBaseRetriever `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KnowledgeBaseRetriever{}, &KnowledgeBaseRetrieverList{})
}
