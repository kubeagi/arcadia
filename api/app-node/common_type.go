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

package appnode

import (
	"encoding/json"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
)

const (
	InputLengthAnnotationKey  = v1alpha1.Group + `/input-rules`
	OutputLengthAnnotationKey = v1alpha1.Group + `/output-rules`
)

type Memory struct {
	// MaxTokenLimit is the maximum number of tokens to keep in memory. Can only use MaxTokenLimit or ConversionWindowSize.
	MaxTokenLimit int `json:"maxTokenLimit,omitempty"`
	// ConversionWindowSize is the maximum number of conversation rounds in memory.Can only use MaxTokenLimit or ConversionWindowSize.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=30
	// +kubebuilder:default=5
	ConversionWindowSize int `json:"conversionWindowSize,omitempty"`
}

type Ref struct {
	Kind   string `json:"kind,omitempty"`
	Group  string `json:"group,omitempty"`
	Length int    `json:"length,omitempty"`
}

func (r *Ref) Len(i int) Ref {
	r.Length = i
	return *r
}

type Node interface {
	SetRef()
}

var (
	ChainRef = Ref{
		Group: "chain.arcadia.kubeagi.k8s.com.cn",
	}
	PromptRef = Ref{
		Kind:  "prompt",
		Group: "prompt.arcadia.kubeagi.k8s.com.cn",
	}
	LLMRef = Ref{
		Kind:  "LLM",
		Group: "arcadia.kubeagi.k8s.com.cn",
	}
	RetrieverRef = Ref{
		Group: "retriever.arcadia.kubeagi.k8s.com.cn",
	}
	KnowledgeBaseRef = Ref{
		Kind:  "KnowledgeBase",
		Group: "arcadia.kubeagi.k8s.com.cn",
	}
	InputRef = Ref{
		Kind: "Input",
	}
	OutputRef = Ref{
		Kind: "Output",
	}
	RetrievalQAChainRef = Ref{
		Group: "chain.arcadia.kubeagi.k8s.com.cn",
		Kind:  "RetrievalQAChain",
	}
	CommonRef = Ref{}
)

func SetRefAnnotations(annotations map[string]string, inputRef []Ref, outputRef []Ref) map[string]string {
	if annotations == nil {
		annotations = make(map[string]string)
	}
	input, _ := json.Marshal(inputRef)
	annotations[InputLengthAnnotationKey] = string(input)
	output, _ := json.Marshal(outputRef)
	annotations[OutputLengthAnnotationKey] = string(output)
	return annotations
}
