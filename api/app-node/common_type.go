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

// +kubebuilder:object:generate=true
type CommonRef struct {
	// Kind is the type of resource being referenced
	// +optional
	Kind string `json:"kind,omitempty"`
	// Name is the name of resource being referenced
	// +optional
	Name string `json:"name,omitempty"`
}

// +kubebuilder:object:generate=true
type ChainRef struct {
	CommonRef `json:",inline"`
	// +kubebuilder:validation:Enum="chain.arcadia.kubeagi.k8s.com.cn"
	// kubebuilder:default="chain.arcadia.kubeagi.k8s.com.cn"
	// APIGroup is the group for the resource being referenced.
	APIGroup string `json:"apiGroup"`
}

// +kubebuilder:object:generate=true
type PromptRef struct {
	CommonRef `json:",inline"`
	// +kubebuilder:validation:Enum="prompt.arcadia.kubeagi.k8s.com.cn"
	// kubebuilder:default="prompt.arcadia.kubeagi.k8s.com.cn"
	// APIGroup is the group for the resource being referenced.
	APIGroup string `json:"apiGroup"`
}

// +kubebuilder:object:generate=true
type LLMRef struct {
	// +kubebuilder:validation:Enum="LLM"
	// kubebuilder:default="LLM"
	// Kind is the type of resource being referenced
	Kind string `json:"kind"`
	// Name is the name of resource being referenced
	// +optional
	Name string `json:"name,omitempty"`
	// +kubebuilder:validation:Enum="arcadia.kubeagi.k8s.com.cn"
	// kubebuilder:default="arcadia.kubeagi.k8s.com.cn"
	// APIGroup is the group for the resource being referenced.
	APIGroup string `json:"apiGroup"`
}

// +kubebuilder:object:generate=true
type CommonOrInPutOrOutputRef struct {
	// APIGroup is the group for the resource being referenced.
	// If APIGroup is not specified, the specified Kind must be in the core API group.
	// For any other third-party types, APIGroup is required.
	APIGroup *string `json:"apiGroup,omitempty"`
	// Kind is the type of resource being referenced
	//+kubebuilder:default=Input
	//+kubebuilder:example=Input;Output
	// +optional
	Kind string `json:"kind,omitempty"`
	// Name is the name of resource being referenced
	Name string `json:"name,omitempty"`
}
