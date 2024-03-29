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
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/utils/env"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/pkg/utils"
)

var (
	ErrNoGPTsConfig         = fmt.Errorf("gpts config in configmap is empty")
	ErrNoGPTsConfigCategory = fmt.Errorf("gpts config Categories in comfigmap is not found")
)

// GPTsConfig is the configurations for GPT Store
type GPTsConfig struct {
	// URL is the url of gpt store
	URL string `json:"url,omitempty"`
	// PublicNamespace is the namespace which all gpt-releated resources are public
	PublicNamespace string     `json:"public_namespace,omitempty"`
	Categories      []Category `json:"categories,omitempty"`
}

// Category in gpt store
type Category struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	NameEn string `json:"nameEn,omitempty"`
}

// GetGPTsConfig gets the gpts configurations
func GetGPTsConfig(ctx context.Context, c client.Client) (gptsConfig *GPTsConfig, err error) {
	cmName := env.GetString(EnvConfigKey, EnvConfigDefaultValue)
	if cmName == "" {
		return nil, ErrNoConfigEnv
	}
	cmNamespace := utils.GetCurrentNamespace()
	cm := &corev1.ConfigMap{}
	if err = c.Get(ctx, client.ObjectKey{Name: cmName, Namespace: cmNamespace}, cm); err != nil {
		return nil, err
	}
	value, ok := cm.Data["gptsConfig"]
	if !ok || len(value) == 0 {
		return nil, ErrNoConfig
	}
	if err = yaml.Unmarshal([]byte(value), &gptsConfig); err != nil {
		return nil, err
	}

	// trim suffix /
	gptsConfig.URL = strings.TrimSuffix(gptsConfig.URL, "/")

	return gptsConfig, nil
}

// GetGPTsCategories gets the gpts Categories
func GetGPTsCategories(ctx context.Context, c client.Client) (categories []Category, err error) {
	gptsConfig, err := GetGPTsConfig(ctx, c)
	if err != nil {
		return nil, err
	}
	if gptsConfig.Categories == nil || len(gptsConfig.Categories) == 0 {
		return nil, ErrNoGPTsConfigCategory
	}
	return gptsConfig.Categories, nil
}
