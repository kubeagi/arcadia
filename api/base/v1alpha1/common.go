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
	"github.com/kubeagi/arcadia/pkg/utils"
)

const (
	// Finalizer is the key of the finalizer
	Finalizer = Group + "/finalizer"
)

// Tags for file object
const (
	ObjectTypeTag  = "object_type"
	ObjectCountTag = "object_count"
	ObjectTypeQA   = "QA"
)

type ProviderType string

const (
	ProviderTypeUnknown  ProviderType = "unknown"
	ProviderType3rdParty ProviderType = "3rd_party"
	ProviderTypeWorker   ProviderType = "worker"
)

// Provider defines how to prvoide the service
type Provider struct {
	// Enpoint defines connection info
	Enpoint *Endpoint `json:"endpoint,omitempty"`

	// Worker defines the worker info
	// Means this LLM is provided by a arcadia worker
	Worker *TypedObjectReference `json:"worker,omitempty"`
}

// GetType returnes the type of this provider
func (p Provider) GetType() ProviderType {
	// if endpoint provided,then 3rd_party
	if p.Enpoint != nil {
		return ProviderType3rdParty
	}
	// if worker provided,then worker
	if p.Worker != nil {
		return ProviderTypeWorker
	}

	return ProviderTypeUnknown
}

// After we use k8s.io/api v1.26.0, we can remove this types to use corev1.TypedObjectReference
// that types is introduced in https://github.com/kubernetes/kubernetes/pull/113186

type TypedObjectReference struct {
	// APIGroup is the group for the resource being referenced.
	// If APIGroup is not specified, the specified Kind must be in the core API group.
	// For any other third-party types, APIGroup is required.
	// +optional
	APIGroup *string `json:"apiGroup" protobuf:"bytes,1,opt,name=apiGroup"`
	// Kind is the type of resource being referenced
	Kind string `json:"kind" protobuf:"bytes,2,opt,name=kind"`
	// Name is the name of resource being referenced
	Name string `json:"name" protobuf:"bytes,3,opt,name=name"`
	// Namespace is the namespace of resource being referenced
	// +optional
	Namespace *string `json:"namespace,omitempty" protobuf:"bytes,4,opt,name=namespace"`
}

func (in *TypedObjectReference) WithAPIGroup(apiGroup string) {
	if in == nil {
		in = &TypedObjectReference{}
	}
	in.APIGroup = &apiGroup
}

func (in *TypedObjectReference) WithKind(kind string) {
	if in == nil {
		in = &TypedObjectReference{}
	}
	in.Kind = kind
}

func (in *TypedObjectReference) WithName(name string) {
	if in == nil {
		in = &TypedObjectReference{}
	}
	in.Name = name
}

func (in *TypedObjectReference) WithNameSpace(namespace string) {
	if in == nil {
		in = &TypedObjectReference{}
	}
	in.Namespace = &namespace
}

func (in *TypedObjectReference) GetNamespace() string {
	if in.Namespace == nil {
		return utils.GetCurrentNamespace()
	}
	return *in.Namespace
}

// Endpoint represents a reachable API endpoint.
type Endpoint struct {
	// URL chart repository address
	// +kubebuilder:validation:Required
	URL string `json:"url,omitempty"`

	// AuthSecret if the chart repository requires auth authentication,
	// set the username and password to secret, with the field user and password respectively.
	AuthSecret *TypedObjectReference `json:"authSecret,omitempty"`

	// Insecure if the endpoint needs a secure connection
	Insecure bool `json:"insecure,omitempty"`
}

type CommonSpec struct {
	// Creator defines datasource creator (AUTO-FILLED by webhook)
	Creator string `json:"creator,omitempty"`

	// DisplayName defines datasource display name
	DisplayName string `json:"displayName,omitempty"`

	// Description defines datasource description
	Description string `json:"description,omitempty"`
}
