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

	Knowledgebase *v1alpha1.TypedObjectReference `json:"knowledgebase"`

	CommonRetrieverConfig `json:",inline"`
}

type CommonRetrieverConfig struct {
	// ScoreThreshold is the cosine distance float score threshold. Lower score represents more similarity.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1
	// +kubebuilder:default=0.3
	ScoreThreshold float32 `json:"scoreThreshold,omitempty"`
	// NumDocuments is the max number of documents to return.
	// +kubebuilder:default=5
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	NumDocuments int `json:"numDocuments,omitempty"`
	// DocNullReturn is the return statement when the query result is empty from the retriever.
	// +kubebuilder:default="未找到您询问的内容，请详细描述您的问题"
	DocNullReturn string `json:"docNullReturn,omitempty"`
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

var _ node.Node = (*KnowledgeBaseRetriever)(nil)

func (c *KnowledgeBaseRetriever) SetRef() {
	annotations := node.SetRefAnnotations(c.GetAnnotations(), []node.Ref{node.KnowledgeBaseRef.Len(1)}, []node.Ref{node.RetrievalQAChainRef.Len(1)})
	if c.GetAnnotations() == nil {
		c.SetAnnotations(annotations)
	}
	for k, v := range annotations {
		c.Annotations[k] = v
	}
}
