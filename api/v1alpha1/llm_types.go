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

// LLMSpec defines the desired state of LLM
type LLMSpec struct {
	DisplayName string `json:"displayName,omitempty"`
	// Type defines the type of llm
	Type LLMType `json:"type"`
	// URL keeps the URL of the llm service(required)
	URL string `json:"url"`
	// Auth keeps the authentication credentials when access llm
	// keeps in k8s secret
	Auth string `json:"auth,omitempty"`
}

type AuthInfo struct {
	APIKey string `json:"apiKey,omitempty"`
}

// LLMStatus defines the observed state of LLM
type LLMStatus struct {
	// ConditionedStatus is the current status
	ConditionedStatus `json:",inline"`
	// URL keeps the URL of the llm service(required)
	URL string `json:"url"`
	// Auth keeps the authentication credentials when access llm
	// keeps in k8s secret
	Auth string `json:"auth,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// LLM is the Schema for the llms API
type LLM struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LLMSpec   `json:"spec,omitempty"`
	Status LLMStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LLMList contains a list of LLM
type LLMList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LLM `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LLM{}, &LLMList{})
}
