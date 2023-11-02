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
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/utils/env"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/v1alpha1"
)

const (
	EnvConfigKey          = "DEFAULT_CONFIG"
	EnvConfigDefaultValue = "arcadia-config"
	EnvNamespaceKey       = "POD_NAMESPACE"

	InClusterNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

var (
	ErrNoConfigEnv    = fmt.Errorf("env:%s is not found", EnvConfigKey)
	ErrNoConfig       = fmt.Errorf("config in configmap is empty")
	ErrNoNamespaceEnv = fmt.Errorf("not in cluster and env:%s is not found", EnvNamespaceKey)
)

func GetSystemDatasource(ctx context.Context, c client.Client) (*arcadiav1alpha1.Datasource, error) {
	config, err := GetConfig(ctx, c)
	if err != nil {
		return nil, err
	}
	name := config.SystemDatasource.Name
	var namespace string
	if config.SystemDatasource.Namespace != nil {
		namespace = *config.SystemDatasource.Namespace
	} else {
		namespace, err = GetSelfNamespace()
		if err != nil {
			return nil, err
		}
	}
	source := &arcadiav1alpha1.Datasource{}
	if err = c.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, source); err != nil {
		return nil, err
	}
	return source, err
}

func GetConfig(ctx context.Context, c client.Client) (config *Config, err error) {
	cmName := env.GetString(EnvConfigKey, EnvConfigDefaultValue)
	if cmName == "" {
		return nil, ErrNoConfigEnv
	}
	cmNamespace, err := GetSelfNamespace()
	if err != nil {
		return nil, err
	}
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

func GetSelfNamespace() (string, error) {
	// Check whether the namespace file exists.
	// If not, we are not running in cluster so can't guess the namespace.
	if _, err := os.Stat(InClusterNamespacePath); os.IsNotExist(err) {
		operatorNamespace := os.Getenv(EnvNamespaceKey)
		if operatorNamespace == "" {
			return "", ErrNoNamespaceEnv
		}
		return operatorNamespace, nil
	} else if err != nil {
		return "", fmt.Errorf("error checking namespace file: %w", err)
	}

	// Load the namespace file and return its content
	namespace, err := os.ReadFile(InClusterNamespacePath)
	if err != nil {
		return "", fmt.Errorf("error reading namespace file: %w", err)
	}
	return string(namespace), nil
}
