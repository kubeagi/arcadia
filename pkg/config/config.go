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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/env"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
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
	ErrNoConfigVectorstore = fmt.Errorf("config Vectorstore in comfigmap is not found")
	ErrNoConfigStreamlit   = fmt.Errorf("config Streamlit in comfigmap is not found")
	ErrNoConfigRayClusters = fmt.Errorf("config RayClusters in comfigmap is not found")
)

func GetSystemDatasource(ctx context.Context, c client.Client, cli dynamic.Interface) (*arcadiav1alpha1.Datasource, error) {
	config, err := GetConfig(ctx, c, cli)
	if err != nil {
		return nil, err
	}
	name := config.SystemDatasource.Name
	var namespace string
	if config.SystemDatasource.Namespace != nil {
		namespace = *config.SystemDatasource.Namespace
	} else {
		namespace = utils.GetCurrentNamespace()
	}
	source := &arcadiav1alpha1.Datasource{}
	if c != nil {
		if err = c.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, source); err != nil {
			return nil, err
		}
	} else {
		obj, err := cli.Resource(schema.GroupVersionResource{Group: arcadiav1alpha1.GroupVersion.Group, Version: arcadiav1alpha1.GroupVersion.Version, Resource: "datasources"}).
			Namespace(namespace).Get(ctx, name, v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, source)
		if err != nil {
			return nil, err
		}
	}
	return source, err
}

func GetGateway(ctx context.Context, c client.Client, cli dynamic.Interface) (*Gateway, error) {
	config, err := GetConfig(ctx, c, cli)
	if err != nil {
		return nil, err
	}
	if config.Gateway == nil {
		return nil, ErrNoConfigGateway
	}
	return config.Gateway, nil
}

func GetConfig(ctx context.Context, c client.Client, cli dynamic.Interface) (config *Config, err error) {
	if err := utils.ValidateClient(c, cli); err != nil {
		return nil, err
	}
	cmName := env.GetString(EnvConfigKey, EnvConfigDefaultValue)
	if cmName == "" {
		return nil, ErrNoConfigEnv
	}
	cmNamespace := utils.GetCurrentNamespace()
	cm := &corev1.ConfigMap{}
	if c != nil {
		if err = c.Get(ctx, client.ObjectKey{Name: cmName, Namespace: cmNamespace}, cm); err != nil {
			return nil, err
		}
	} else {
		obj, err := cli.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}).
			Namespace(cmNamespace).Get(ctx, cmName, v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), cm)
		if err != nil {
			return nil, err
		}
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

func GetVectorStore(ctx context.Context, c dynamic.Interface) (*arcadiav1alpha1.TypedObjectReference, error) {
	config, err := GetConfig(ctx, nil, c)
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
	config, err := GetConfig(ctx, c, nil)
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
	config, err := GetConfig(ctx, c, nil)
	if err != nil {
		return nil, err
	}
	if config.RayClusters == nil {
		return nil, ErrNoConfigRayClusters
	}
	return config.RayClusters, nil
}
