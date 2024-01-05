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
	chromago "github.com/amikos-tech/chroma-go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VectorStoreSpec defines the desired state of VectorStore
type VectorStoreSpec struct {
	CommonSpec `json:",inline"`

	// Endpoint defines connection info
	Endpoint *Endpoint `json:"endpoint,omitempty"`

	Chroma *Chroma `json:"chroma,omitempty"`

	PGVector *PGVector `json:"pgvector,omitempty"`
}

// Chroma defines the configuration of Chroma
type Chroma struct {
	DistanceFunction chromago.DistanceFunction `json:"distanceFunction,omitempty"`
}

type PGVector struct {
	// PreDeleteCollection defines if the collection should be deleted before creating.
	PreDeleteCollection bool `json:"preDeleteCollection,omitempty"`
	// CollectionName defines the name of the collection
	CollectionName string `json:"collectionName,omitempty"`
	// EmbeddingTableName defines the name of the embedding table. if empty, use `langchain_pg_embedding`
	EmbeddingTableName string `json:"embeddingTableName,omitempty"`
	// CollectionTableName defines the name of the collection table. if empty, use `langchain_pg_collection`
	CollectionTableName string `json:"collectionTableName,omitempty"`
	// DataSourceRef defines the reference of the data source
	DataSourceRef *TypedObjectReference `json:"dataSourceRef,omitempty"`
}

// VectorStoreStatus defines the observed state of VectorStore
type VectorStoreStatus struct {
	// ConditionedStatus is the current status
	ConditionedStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="display-name",type=string,JSONPath=`.spec.displayName`

// VectorStore is the Schema for the vectorstores API
type VectorStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VectorStoreSpec   `json:"spec,omitempty"`
	Status VectorStoreStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VectorStoreList contains a list of VectorStore
type VectorStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VectorStore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VectorStore{}, &VectorStoreList{})
}
