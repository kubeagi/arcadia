/*
Copyright 2024 KubeAGI.

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

// APIChainSpec defines the desired state of APIChain
type APIChainSpec struct {
	v1alpha1.CommonSpec `json:",inline"`

	LLM *v1alpha1.TypedObjectReference `json:"llm"`

	CommonChainConfig `json:",inline"`
	// APIDoc is the api doc for this chain, "api_docs"
	APIDoc string `json:"apiDoc"`
}

// APIChainStatus defines the observed state of APIChain
type APIChainStatus struct {
	// ObservedGeneration is the last observed generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// ConditionedStatus is the current status
	v1alpha1.ConditionedStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// APIChain is a chain that makes API calls and summarizes the responses to answer a question.
type APIChain struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   APIChainSpec   `json:"spec,omitempty"`
	Status APIChainStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// APIChainList contains a list of APIChain
type APIChainList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []APIChain `json:"items"`
}

func init() {
	SchemeBuilder.Register(&APIChain{}, &APIChainList{})
}

var _ node.Node = (*APIChain)(nil)

func (c *APIChain) SetRef() {
	annotations := node.SetRefAnnotations(c.GetAnnotations(), []node.Ref{node.LLMRef.Len(1), node.PromptRef.Len(1)}, []node.Ref{node.OutputRef.Len(1)})
	if c.GetAnnotations() == nil {
		c.SetAnnotations(annotations)
	}
	for k, v := range annotations {
		c.Annotations[k] = v
	}
}
