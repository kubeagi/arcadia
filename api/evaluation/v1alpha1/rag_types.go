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
}

// RAGStatus defines the observed state of RAG
type RAGStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
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
