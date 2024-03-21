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

// DocumentLoaderSpec defines the desired state of DocumentLoader
type DocumentLoaderSpec struct {
	// CommonSpec
	v1alpha1.CommonSpec `json:",inline"`
	// ChunkSize for text splitter
	// +kubebuilder:default=512
	ChunkSize int `json:"chunkSize,omitempty"`
	// ChunkOverlap for text splitter
	// +kubebuilder:default=100
	ChunkOverlap *int `json:"chunkOverlap,omitempty"`
	// BatchSize for text splitter
	// +kubebuilder:default=10
	BatchSize int `json:"batchSize,omitempty"`
	// FileExtName the type of documents, can be .pdf, .txt, .mp3, etc ...
	FileExtName string `json:"fileExtName,omitempty"`
	// LoaderConfig defines the config of loader tools
	LoaderConfig `json:",inline"`
}

// LoaderConfig defines the config of the loader
type LoaderConfig struct {
	Params map[string]string `json:"params,omitempty"`
}

// LoaderStatus defines the observed state of loader
type LoaderStatus struct {
	// ObservedGeneration is the last observed generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// ConditionedStatus is the current status
	v1alpha1.ConditionedStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DocumentLoader is the Schema for the DocumentLoader
type DocumentLoader struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DocumentLoaderSpec `json:"spec,omitempty"`
	Status LoaderStatus       `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DocumentLoaderList contains a list of DocumentLoader
type DocumentLoaderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DocumentLoader `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DocumentLoader{}, &DocumentLoaderList{})
}

var _ node.Node = (*DocumentLoader)(nil)

func (c *DocumentLoader) SetRef() {
	annotations := node.SetRefAnnotations(c.GetAnnotations(), []node.Ref{node.InputRef.Len(1)}, []node.Ref{node.CommonRef.Len(1)})
	if c.GetAnnotations() == nil {
		c.SetAnnotations(annotations)
	}
	for k, v := range annotations {
		c.Annotations[k] = v
	}
}
