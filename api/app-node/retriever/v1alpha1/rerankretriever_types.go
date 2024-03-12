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

// RerankRetrieverSpec defines the desired state of RerankRetriever
type RerankRetrieverSpec struct {
	v1alpha1.CommonSpec   `json:",inline"`
	CommonRetrieverConfig `json:",inline"`
	// the model of the rerank
	Model *v1alpha1.TypedObjectReference `json:"model,omitempty"`
}

// RerankRetrieverStatus defines the observed state of RerankRetriever
type RerankRetrieverStatus struct {
	// ObservedGeneration is the last observed generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// ConditionedStatus is the current status
	v1alpha1.ConditionedStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RerankRetriever is the Schema for the RerankRetriever API
type RerankRetriever struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RerankRetrieverSpec   `json:"spec,omitempty"`
	Status RerankRetrieverStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RerankRetrieverList contains a list of RerankRetriever
type RerankRetrieverList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RerankRetriever `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RerankRetriever{}, &RerankRetrieverList{})
}

var _ node.Node = (*RerankRetriever)(nil)

func (c *RerankRetriever) SetRef() {
	annotations := node.SetRefAnnotations(c.GetAnnotations(), []node.Ref{node.RetrieverRef.Len(1)}, []node.Ref{node.RetrievalQAChainRef.Len(1)})
	if c.GetAnnotations() == nil {
		c.SetAnnotations(annotations)
	}
	for k, v := range annotations {
		c.Annotations[k] = v
	}
}
