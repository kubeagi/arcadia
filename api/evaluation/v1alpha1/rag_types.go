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
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	basev1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
)

// Dataset stands for the files used to generate ragas test dataset
type Dataset struct {
	// From defines the source which provides this QA Files for test dataset
	// Only `VersionedDataset` allowed
	Source *basev1alpha1.TypedObjectReference `json:"source,omitempty"`
	// Files retrieved from Source and used in this testdataset
	// - For file with tag `object_type: QA`, will be used directly
	// - TODO: For file without special tags, will use `QAGenerationChain` to generate QAs (Not Supported Yet)
	Files []string `json:"files,omitempty"`
}

// RAGSpec defines the desired state of RAG
type RAGSpec struct {
	// CommonSpec
	basev1alpha1.CommonSpec `json:",inline"`

	// Application(required) defines the target of this RAG evaluation
	Application *basev1alpha1.TypedObjectReference `json:"application"`

	// Datasets defines the dataset which will be used to generate test datasets
	Datasets []Dataset `json:"datasets"`

	// JudgeLLM(required) defines the judge which is a LLM to evaluate RAG application against test dataset
	JudgeLLM *basev1alpha1.TypedObjectReference `json:"judge_llm"`

	// Metrics that this rag evaluation will do
	Metrics []Metric `json:"metrics"`

	// Report defines the evaluation report configurations
	Report Report `json:"report,omitempty"`

	// Storage storage must be provided and data needs to be saved throughout the evaluation phase.
	Storage *corev1.PersistentVolumeClaimSpec `json:"storage"`

	// ServiceAccountName define the user when the job is run
	// +kubebuilder:default=default
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Suspend suspension of the evaluation process
	// +kubebuilder:default=false
	Suspend bool `json:"suspend,omitempty"`
}

// RAGStatus defines the observed state of RAG
type RAGStatus struct {
	// CompletionTime Evaluation completion time
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Phase evaluation current stage,
	// init,download,generate,judge,upload,complete
	Phase RAGPhase `json:"phase,omitempty"`

	// Conditions show the status of the job in the current stage
	Conditions []v1.JobCondition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// RAG is the Schema for the rags API
type RAG struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RAGSpec   `json:"spec,omitempty"`
	Status RAGStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RAGList contains a list of RAG
type RAGList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RAG `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RAG{}, &RAGList{})
}
