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
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/utils/env"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/datasource"
	"github.com/kubeagi/arcadia/pkg/utils"
)

const (
	EnvConfigKey          = "DEFAULT_CONFIG"
	EnvConfigDefaultValue = "arcadia-config"
)

var (
	ErrNoConfigEnv         = fmt.Errorf("env:%s is not found", EnvConfigKey)
	ErrNoConfig            = fmt.Errorf("config in configmap is empty")
	ErrNoConfigGateway     = fmt.Errorf("config Gateway in configmap is not found")
	ErrNoConfigMinIO       = fmt.Errorf("config MinIO in comfigmap is not found")
	ErrNoConfigEmbedder    = fmt.Errorf("config Embedder in comfigmap is not found")
	ErrNoConfigVectorstore = fmt.Errorf("config Vectorstore in comfigmap is not found")
	ErrNoConfigStreamlit   = fmt.Errorf("config Streamlit in comfigmap is not found")
	ErrNoConfigRayClusters = fmt.Errorf("config RayClusters in comfigmap is not found")
	ErrNoConfigRerank      = fmt.Errorf("config rerankDefaultEndpoint in comfigmap is not found")
)

func getDatasource(ctx context.Context, ref arcadiav1alpha1.TypedObjectReference, c client.Client) (ds *arcadiav1alpha1.Datasource, err error) {
	name := ref.Name
	namespace := ref.GetNamespace(utils.GetCurrentNamespace())
	source := &arcadiav1alpha1.Datasource{}
	if err = c.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, source); err != nil {
		return nil, err
	}
	return source, err
}

func GetSystemDatasource(ctx context.Context, c client.Client) (*arcadiav1alpha1.Datasource, error) {
	config, err := GetConfig(ctx, c)
	if err != nil {
		return nil, err
	}
	return getDatasource(ctx, config.SystemDatasource, c)
}

func GetRelationalDatasource(ctx context.Context, c client.Client) (*arcadiav1alpha1.Datasource, error) {
	config, err := GetConfig(ctx, c)
	if err != nil {
		return nil, err
	}
	return getDatasource(ctx, config.RelationalDatasource, c)
}

func GetGateway(ctx context.Context, c client.Client) (*Gateway, error) {
	config, err := GetConfig(ctx, c)
	if err != nil {
		return nil, err
	}
	if config.Gateway == nil {
		return nil, ErrNoConfigGateway
	}
	return config.Gateway, nil
}

func GetConfig(ctx context.Context, c client.Client) (config *Config, err error) {
	cmName := env.GetString(EnvConfigKey, EnvConfigDefaultValue)
	if cmName == "" {
		return nil, ErrNoConfigEnv
	}
	cmNamespace := utils.GetCurrentNamespace()
	cm := &corev1.ConfigMap{}
	if err = c.Get(ctx, client.ObjectKey{Name: cmName, Namespace: cmNamespace}, cm); err != nil {
		return nil, err
	}
	value, ok := cm.Data["config"]
	if !ok || len(value) == 0 {
		return nil, ErrNoConfig
	}
	if err = yaml.Unmarshal([]byte(value), &config); err != nil {
		return nil, err
	}
	return config, nil
}

// GetEmbedder get the default embedder from config
func GetEmbedder(ctx context.Context, c client.Client) (*arcadiav1alpha1.TypedObjectReference, error) {
	config, err := GetConfig(ctx, c)
	if err != nil {
		return nil, err
	}
	if config.Embedder == nil {
		return nil, ErrNoConfigEmbedder
	}
	return config.Embedder, nil
}

// GetVectorStore get the default vector store from config
func GetVectorStore(ctx context.Context, c client.Client) (*arcadiav1alpha1.TypedObjectReference, error) {
	config, err := GetConfig(ctx, c)
	if err != nil {
		return nil, err
	}
	if config.VectorStore == nil {
		return nil, ErrNoConfigVectorstore
	}
	return config.VectorStore, nil
}

// Get the configuration of streamlit tool
func GetStreamlit(ctx context.Context, c client.Client) (*Streamlit, error) {
	config, err := GetConfig(ctx, c)
	if err != nil {
		return nil, err
	}
	if config.Streamlit == nil {
		return nil, ErrNoConfigStreamlit
	}
	return config.Streamlit, nil
}

// Get the ray cluster that can be used a resource pool
func GetRayClusters(ctx context.Context, c client.Client) ([]RayCluster, error) {
	config, err := GetConfig(ctx, c)
	if err != nil {
		return nil, err
	}
	if config.RayClusters == nil {
		return nil, ErrNoConfigRayClusters
	}
	return config.RayClusters, nil
}

// GetDefaultRerankModel gets the default reranking model which is recommended by kubeagi
func GetDefaultRerankModel(ctx context.Context, c client.Client) (*arcadiav1alpha1.TypedObjectReference, error) {
	config, err := GetConfig(ctx, c)
	if err != nil {
		return nil, err
	}
	if config.Rerank == nil {
		return nil, ErrNoConfigRerank
	}
	return config.Rerank, nil
}

func GetSystemDatasourceOSS(ctx context.Context, mgrClient client.Client) (*datasource.OSS, error) {
	systemDatasource, err := GetSystemDatasource(ctx, mgrClient)
	if err != nil {
		return nil, err
	}
	endpoint := systemDatasource.Spec.Endpoint.DeepCopy()
	if endpoint.AuthSecret != nil && endpoint.AuthSecret.Namespace == nil {
		endpoint.AuthSecret.WithNameSpace(systemDatasource.Namespace)
	}
	return datasource.NewOSS(ctx, mgrClient, endpoint)
}
