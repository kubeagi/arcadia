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

// LLMChainSpec defines the desired state of LLMChain
type LLMChainSpec struct {
	v1alpha1.CommonSpec `json:",inline"`

	CommonChainConfig `json:",inline"`

	Input  Input  `json:"input"`
	Output Output `json:"output"`
}

type Input struct {
	LLM    node.LLMRef    `json:"llm"`
	Prompt node.PromptRef `json:"prompt"`
}
type Output struct {
	node.CommonOrInPutOrOutputRef `json:",inline"`
}

type CommonChainConfig struct {
	// 记忆相关参数
	Memory Memory `json:"memory,omitempty"`
}

type Memory struct {
	// 能记住的最大 token 数
	MaxTokenLimit int `json:"maxTokenLimit,omitempty"`
}

// LLMChainStatus defines the observed state of LLMChain
type LLMChainStatus struct {
	// ObservedGeneration is the last observed generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// ConditionedStatus is the current status
	v1alpha1.ConditionedStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// LLMChain is the Schema for the LLMChains API
type LLMChain struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LLMChainSpec   `json:"spec,omitempty"`
	Status LLMChainStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LLMChainList contains a list of LLMChain
type LLMChainList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LLMChain `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LLMChain{}, &LLMChainList{})
}
