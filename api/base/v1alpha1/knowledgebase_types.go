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

// KnowledgeBaseSpec defines the desired state of KnowledgeBase
type KnowledgeBaseSpec struct {
	CommonSpec `json:",inline"`

	// Embedder defines the embedder to embedding files
	Embedder *TypedObjectReference `json:"embedder,omitempty"`

	// VectorStore defines the vectorstore to store results
	VectorStore *TypedObjectReference `json:"vectorStore,omitempty"`

	// FileGroups included files Grouped by VersionedDataset
	FileGroups []FileGroup `json:"fileGroups,omitempty"`
}

type FileGroupDetail struct {
	// From defines the datasource which provides these files
	Source *TypedObjectReference `json:"source,omitempty"`

	// FileDetails is the detail files
	FileDetails []FileDetails `json:"fileDetails,omitempty"`
}

type FileDetails struct {
	// Path defines the detail path to get objects from above datasource
	Path string `json:"path,omitempty"`

	// Checksum defines the checksum of the file
	Checksum string `json:"checksum,omitempty"`

	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`

	// Phase defines the process phase
	Phase FileProcessPhase `json:"phase,omitempty"`

	// ErrMessage defines the error message
	ErrMessage string `json:"errMessage,omitempty"`
}

type FileProcessPhase string

const (
	FileProcessPhasePending    FileProcessPhase = "Pending"
	FileProcessPhaseProcessing FileProcessPhase = "Processing"
	FileProcessPhaseSucceeded  FileProcessPhase = "Succeeded"
	FileProcessPhaseFailed     FileProcessPhase = "Failed"
	FileProcessPhaseSkipped    FileProcessPhase = "Skipped"
)

// KnowledgeBaseStatus defines the observed state of KnowledgeBase
type KnowledgeBaseStatus struct {
	// ObservedGeneration is the last observed generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// FileGroupDetail is the detail of these files
	FileGroupDetail []FileGroupDetail `json:"fileGroupDetail,omitempty"`

	// ConditionedStatus is the current status
	ConditionedStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="display-name",type=string,JSONPath=`.spec.displayName`

// KnowledgeBase is the Schema for the knowledgebases API
type KnowledgeBase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KnowledgeBaseSpec   `json:"spec,omitempty"`
	Status KnowledgeBaseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KnowledgeBaseList contains a list of KnowledgeBase
type KnowledgeBaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KnowledgeBase `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KnowledgeBase{}, &KnowledgeBaseList{})
}
