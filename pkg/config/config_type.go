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

package config

import (
	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
)

// Config defines the configuration for the Arcadia controller
type Config struct {
	// SystemDatasource specifies the built-in datasource for Arcadia to host data files and model files
	SystemDatasource arcadiav1alpha1.TypedObjectReference `json:"systemDatasource,omitempty"`

	// Gateway to access LLM api services
	Gateway *Gateway `json:"gateway,omitempty"`

	// VectorStore to access VectorStore api services
	VectorStore *arcadiav1alpha1.TypedObjectReference `json:"vectorStore,omitempty"`

	// Streamlit to get the Streamlit configuration
	Streamlit *Streamlit `json:"streamlit,omitempty"`

	// Resource pool managed by Ray cluster
	RayClusters []RayCluster `json:"rayClusters,omitempty"`
}

// Gateway defines the way to access llm apis host by Arcadia
type Gateway struct {
	ExternalAPIServer string `json:"externalApiServer,omitempty"`
	APIServer         string `json:"apiServer,omitempty"`
	Controller        string `json:"controller,omitempty"`
}

// Streamlit defines the configuration of streamlit app
type Streamlit struct {
	Image            string `json:"image"`
	IngressClassName string `json:"ingressClassName"`
	Host             string `json:"host"`
	ContextPath      string `json:"contextPath"`
}

// RayCluster defines configuration of existing ray cluster that manage GPU resources
type RayCluster struct {
	// Name of this ray cluster
	Name string `json:"name,omitempty"`
	// Address of ray head address
	HeadAddress string `json:"headAddress,omitempty"`
	// Management dashboard of ray cluster, optional to configure it using ingress
	DashboardHost string `json:"dashboardHost,omitempty"`
	// Overwrite the python version in the woker
	PythonVersion string `json:"pythonVersion,omitempty"`
}
