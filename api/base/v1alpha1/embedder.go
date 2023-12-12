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
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/pkg/utils"
)

func (e Embedder) AuthAPIKey(ctx context.Context, c client.Client, cli dynamic.Interface) (string, error) {
	if e.Spec.Enpoint == nil || e.Spec.Enpoint.AuthSecret == nil {
		return "", nil
	}
	if err := utils.ValidateClient(c, cli); err != nil {
		return "", err
	}
	authSecret := &corev1.Secret{}
	if c != nil {
		if err := c.Get(ctx, types.NamespacedName{Name: e.Spec.Enpoint.AuthSecret.Name, Namespace: e.Namespace}, authSecret); err != nil {
			return "", err
		}
	} else {
		obj, err := cli.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}).
			Namespace(e.GetNamespace()).Get(ctx, e.Spec.Enpoint.AuthSecret.Name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), authSecret)
		if err != nil {
			return "", err
		}
	}
	return string(authSecret.Data["apiKey"]), nil
}

type EmbeddingType string

const (
	OpenAI  EmbeddingType = "openai"
	ZhiPuAI EmbeddingType = "zhipuai"
)
