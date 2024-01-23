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
}

type CommonChainConfig struct {
	// for memory
	Memory Memory `json:"memory,omitempty"`

	// Model is the model to use in an llm call.like `gpt-3.5-turbo` or `chatglm_turbo`
	// Usually this value is just empty
	Model string `json:"model,omitempty"`
	// MaxTokens is the maximum number of tokens to generate to use in a llm call.
	// +kubebuilder:default=2048
	MaxTokens int `json:"maxTokens,omitempty"`
	// Temperature is the temperature for sampling to use in a llm call, between 0 and 1.
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:Maximum=1
	//+kubebuilder:default=0.7
	Temperature float64 `json:"temperature,omitempty"`
	// StopWords is a list of words to stop on to use in a llm call.
	StopWords []string `json:"stopWords,omitempty"`
	// TopK is the number of tokens to consider for top-k sampling in a llm call.
	TopK int `json:"topK,omitempty"`
	// TopP is the cumulative probability for top-p sampling in a llm call.
	TopP float64 `json:"topP,omitempty"`
	// Seed is a seed for deterministic sampling in a llm call.
	Seed int `json:"seed,omitempty"`
	// MinLength is the minimum length of the generated text in a llm call.
	MinLength int `json:"minLength,omitempty"`
	// MaxLength is the maximum length of the generated text in a llm call.
	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:default=2048
	MaxLength int `json:"maxLength,omitempty"`
	// RepetitionPenalty is the repetition penalty for sampling in a llm call.
	RepetitionPenalty float64 `json:"repetitionPenalty,omitempty"`
}

type Memory struct {
	// MaxTokenLimit is the maximum number of tokens to keep in memory. Can only use MaxTokenLimit or ConversionWindowSize.
	MaxTokenLimit int `json:"maxTokenLimit,omitempty"`
	// ConversionWindowSize is the maximum number of conversation rounds in memory.Can only use MaxTokenLimit or ConversionWindowSize.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=30
	// +kubebuilder:default=5
	ConversionWindowSize int `json:"conversionWindowSize,omitempty"`
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

var _ node.Node = (*LLMChain)(nil)

func (c *LLMChain) SetRef() {
	annotations := node.SetRefAnnotations(c.GetAnnotations(), []node.Ref{node.LLMRef.Len(1), node.PromptRef.Len(1)}, []node.Ref{node.OutputRef.Len(1)})
	if c.GetAnnotations() == nil {
		c.SetAnnotations(annotations)
	}
	for k, v := range annotations {
		c.Annotations[k] = v
	}
}
