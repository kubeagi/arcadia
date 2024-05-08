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

// MergerRetrieverSpec defines the desired state of MergerRetriever
type MergerRetrieverSpec struct {
	v1alpha1.CommonSpec `json:",inline"`
}

// MergerRetrieverStatus defines the observed state of MergerRetriever
type MergerRetrieverStatus struct {
	// ObservedGeneration is the last observed generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// ConditionedStatus is the current status
	v1alpha1.ConditionedStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MergerRetriever is the Schema for the MergerRetriever API
type MergerRetriever struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MergerRetrieverSpec   `json:"spec,omitempty"`
	Status MergerRetrieverStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MergerRetrieverList contains a list of MergerRetriever
type MergerRetrieverList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MergerRetriever `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MergerRetriever{}, &MergerRetrieverList{})
}

var _ node.Node = (*MergerRetriever)(nil)

func (c *MergerRetriever) SetRef() {
	annotations := node.SetRefAnnotations(c.GetAnnotations(), []node.Ref{node.RetrieverRef.Len(1)}, []node.Ref{node.RetrievalQAChainRef.Len(1)})
	if c.GetAnnotations() == nil {
		c.SetAnnotations(annotations)
	}
	for k, v := range annotations {
		c.Annotations[k] = v
	}
}
