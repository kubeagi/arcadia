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

type NodeConfig struct {
	// +kubebuilder:validation:Required
	Name        string                `json:"name,omitempty"`
	DisplayName string                `json:"displayName,omitempty"`
	Description string                `json:"description,omitempty"`
	Ref         *TypedObjectReference `json:"ref,omitempty"`
}

// ApplicationSpec defines the desired state of Application
type ApplicationSpec struct {
	CommonSpec `json:",inline"`
	// 开场白，只给客户看的内容，引导客户首次输入
	Prologue string `json:"prologue,omitempty"`
	// 节点
	// +kubebuilder:validation:Required
	Nodes []Node `json:"nodes"`
}

type Node struct {
	NodeConfig   `json:",inline"`
	NextNodeName []string `json:"nextNodeName,omitempty"`
}

// ApplicationStatus defines the observed state of Application
type ApplicationStatus struct {
	// ObservedGeneration is the last observed generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// ConditionedStatus is the current status
	ConditionedStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Application is the Schema for the applications API
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec,omitempty"`
	Status ApplicationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ApplicationList contains a list of Application
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}
