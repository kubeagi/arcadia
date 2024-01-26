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
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/pkg/embeddings"
)

func (e Embedder) AuthAPIKey(ctx context.Context, c client.Client, cli dynamic.Interface) (string, error) {
	if e.Spec.Endpoint == nil {
		return "", nil
	}
	return e.Spec.Endpoint.AuthAPIKey(ctx, e.GetNamespace(), c, cli)
}

// GetEmbedderBaseUrl returns the embedder's url
func (e Embedder) Get3rdPartyEmbedderBaseURL() string {
	return e.Spec.Endpoint.URL
}

// GetModelList returns a model list provided by this LLM based on different provider
func (e Embedder) GetModelList() []string {
	switch e.Spec.Provider.GetType() {
	case ProviderTypeWorker:
		return e.GetWorkerModels()
	case ProviderType3rdParty:
		return e.Get3rdPartyModels()
	}
	return []string{}
}

// GetWorkerModels returns a model list which provided by this worker provider
func (e Embedder) GetWorkerModels() []string {
	// Get the worker's uid from owner reference as the model id
	ownerObj := e.GetOwnerReferences()
	if len(ownerObj) > 0 {
		if ownerObj[0].Kind == "Worker" {
			return []string{string(ownerObj[0].UID)}
		}
	}
	return []string{}
}

// Get3rdPartyModels returns a model list which provided by the 3rd party provider
func (e Embedder) Get3rdPartyModels() []string {
	if e.Spec.Provider.GetType() != ProviderType3rdParty {
		return []string{}
	}

	//  if models(customized) are provided,then return it
	if e.Spec.Models != nil && len(e.Spec.Models) != 0 {
		return e.Spec.Models
	}

	switch e.Spec.Type {
	case embeddings.ZhiPuAI:
		return embeddings.ZhiPuAIModels
	case embeddings.OpenAI:
		return embeddings.OpenAIModels
	}

	return []string{}
}

func (e Embedder) ReadyCondition(msg string) Condition {
	currCon := e.Status.GetCondition(TypeReady)
	// return current condition if condition not changed
	if currCon.Status == corev1.ConditionTrue && currCon.Reason == ReasonAvailable && currCon.Message == msg {
		return currCon
	}
	return Condition{
		Type:               TypeReady,
		Status:             corev1.ConditionTrue,
		Reason:             ReasonAvailable,
		Message:            msg,
		LastTransitionTime: metav1.Now(),
		LastSuccessfulTime: metav1.Now(),
	}
}

func (e Embedder) ErrorCondition(msg string) Condition {
	currCon := e.Status.GetCondition(TypeReady)
	// return current condition if condition not changed
	if currCon.Status == corev1.ConditionFalse && currCon.Reason == ReasonUnavailable && currCon.Message == msg {
		return currCon
	}
	// keep original LastSuccessfulTime if have
	lastSuccessfulTime := metav1.Now()
	if currCon.LastSuccessfulTime.IsZero() {
		lastSuccessfulTime = currCon.LastSuccessfulTime
	}
	return Condition{
		Type:               TypeReady,
		Status:             corev1.ConditionFalse,
		Reason:             ReasonUnavailable,
		Message:            msg,
		LastSuccessfulTime: lastSuccessfulTime,
		LastTransitionTime: metav1.Now(),
	}
}
