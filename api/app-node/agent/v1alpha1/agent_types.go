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

// AgentSpec defines the desired state of Agent
type AgentSpec struct {
	v1alpha1.CommonSpec `json:",inline"`

	AgentConfig `json:",inline"`
}

type AgentConfig struct {
	// type, can be zeroShot or conversational
	//+kubebuilder:default="zeroShot"
	Type string `json:"type,omitempty"`
	// list of allowed tools for this agent
	AllowedTools []Tool `json:"allowedTools,omitempty"`
	// http action like get/post
	Options Options `json:"options,omitempty"`
}

// Options defines the options to be used by agent
type Options struct {
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:default=5
	MaxIterations int `json:"maxIterations,omitempty"`

	// The options below might be used later
	// prompt prompts.PromptTemplate
	// outputKey               string
	// promptPrefix            string
	// formatInstructions      string
	// promptSuffix            string
	// returnIntermediateSteps bool
	// memory schema.Memory
}

// Tool/Capability that this agent will use
type Tool struct {
	// Name of the tool
	Name string `json:"name,omitempty"`
	// Map of key/value that will be passed to the tool
	Params map[string]string `json:"params,omitempty"`
}

// AgentStatus defines the observed state of Agent
type AgentStatus struct {
	// ObservedGeneration is the last observed generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// ConditionedStatus is the current status
	v1alpha1.ConditionedStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Agent is the Schema for the Agent API
type Agent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentSpec   `json:"spec,omitempty"`
	Status AgentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AgentList contains a list of Agent
type AgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Agent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Agent{}, &AgentList{})
}

var _ node.Node = (*Agent)(nil)

func (c *Agent) SetRef() {
	annotations := node.SetRefAnnotations(c.GetAnnotations(), []node.Ref{node.InputRef.Len(1)}, []node.Ref{node.CommonRef.Len(1)})
	if c.GetAnnotations() == nil {
		c.SetAnnotations(annotations)
	}
	for k, v := range annotations {
		c.Annotations[k] = v
	}
}
