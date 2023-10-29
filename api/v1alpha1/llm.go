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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (llm LLM) AuthAPIKey(ctx context.Context, c client.Client) (string, error) {
	if llm.Spec.Enpoint == nil || llm.Spec.Enpoint.AuthSecret == nil {
		return "", nil
	}
	authSecret := &corev1.Secret{}
	err := c.Get(ctx, types.NamespacedName{Name: llm.Spec.Enpoint.AuthSecret.Name, Namespace: llm.Namespace}, authSecret)
	if err != nil {
		return "", err
	}
	return string(authSecret.Data["apiKey"]), nil
}

func (llmStatus LLMStatus) LLMReady() (string, bool) {
	if len(llmStatus.Conditions) == 0 {
		return "No conditions yet", false
	}
	if llmStatus.Conditions[0].Type != TypeReady || llmStatus.Conditions[0].Status != corev1.ConditionTrue {
		return "Bad condition", false
	}
	return "", true
}
