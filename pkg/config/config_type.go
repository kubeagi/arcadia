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

	// MinIO to access MinIO api services
	MinIO *MinIO `json:"minIO,omitempty"`

	// VectorStore to access VectorStore api services
	VectorStore *arcadiav1alpha1.TypedObjectReference `json:"vectorStore,omitempty"`

	// Streamkit to get the configuration
	Streamkit *Streamkit `json:"streamkit,omitempty"`
}

// Gateway defines the way to access llm apis host by Arcadia
type Gateway struct {
	APIServer  string `json:"apiServer,omitempty"`
	Controller string `json:"controller,omitempty"`
}

// MinIO defines the way to access minio
type MinIO struct {
	MinioAddress         string `json:"minioAddress"`
	MinioAccessKeyID     string `json:"minioAccessKeyId"`
	MinioSecretAccessKey string `json:"minioSecretAccessKey"`
	MinioSecure          bool   `json:"minioSecure"`
	MinioBucket          string `json:"minioBucket"`
	MinioBasePath        string `json:"minioBasePath"`
}

// Streamkit defines the configuration of streamkit app
type Streamkit struct {
	Image            string `json:"image"`
	IngressClassName string `json:"ingressClassName"`
	Host             string `json:"host"`
	ContextPath      string `json:"contextPath"`
}
