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

// RetrievalQAChainSpec defines the desired state of RetrievalQAChain
type RetrievalQAChainSpec struct {
	v1alpha1.CommonSpec `json:",inline"`

	LLM *v1alpha1.TypedObjectReference `json:"llm"`

	CommonChainConfig `json:",inline"`
}

// RetrievalQAChainStatus defines the observed state of RetrievalQAChain
type RetrievalQAChainStatus struct {
	// ObservedGeneration is the last observed generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// ConditionedStatus is the current status
	v1alpha1.ConditionedStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RetrievalQAChain is a chain used for question-answering against a retriever.
// First the chain gets documents from the retriever, then the documents
// and the query are used as input to another chain.Typically, that chain
// combines the documents into a prompt that is sent to a llm.
type RetrievalQAChain struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RetrievalQAChainSpec   `json:"spec,omitempty"`
	Status RetrievalQAChainStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RetrievalQAChainList contains a list of RetrievalQAChain
type RetrievalQAChainList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RetrievalQAChain `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RetrievalQAChain{}, &RetrievalQAChainList{})
}

var _ node.Node = (*RetrievalQAChain)(nil)

func (c *RetrievalQAChain) SetRef() {
	annotations := node.SetRefAnnotations(c.GetAnnotations(), []node.Ref{node.LLMRef.Len(1), node.PromptRef.Len(1), node.RetrieverRef.Len(1)}, []node.Ref{node.OutputRef.Len(1)})
	if c.GetAnnotations() == nil {
		c.SetAnnotations(annotations)
	}
	for k, v := range annotations {
		c.Annotations[k] = v
	}
}
