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
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/pkg/llms"
)

func (llm LLM) AuthAPIKey(ctx context.Context, c client.Client) (string, error) {
	if llm.Spec.Endpoint == nil {
		return "", nil
	}
	return llm.Spec.Endpoint.AuthAPIKey(ctx, llm.GetNamespace(), c)
}

func (llmStatus LLMStatus) LLMReady() (string, bool) {
	if len(llmStatus.Conditions) == 0 {
		return "No conditions yet", false
	}
	if !llmStatus.IsReady() {
		return "Bad condition", false
	}
	return "", true
}

// GetLLMBaseURL returns the llm's url
func (llm LLM) Get3rdPartyLLMBaseURL() string {
	return llm.Spec.Endpoint.URL
}

// GetModelList returns a model list provided by this LLM based on different provider
func (llm LLM) GetModelList() []string {
	switch llm.Spec.Provider.GetType() {
	case ProviderTypeWorker:
		return llm.GetWorkerModels()
	case ProviderType3rdParty:
		return llm.Get3rdPartyModels()
	}
	return []string{}
}

// GetWorkerModels returns a model list which provided by this worker provider
func (llm LLM) GetWorkerModels() []string {
	// Get the worker's uid from owner reference as the model id
	ownerObj := llm.GetOwnerReferences()
	if len(ownerObj) > 0 {
		if ownerObj[0].Kind == "Worker" {
			return []string{string(ownerObj[0].UID)}
		}
	}
	return []string{}
}

// Get3rdPartyModels returns a model list which provided by the 3rd party provider
func (llm LLM) Get3rdPartyModels() []string {
	if llm.Spec.Provider.GetType() != ProviderType3rdParty {
		return []string{}
	}

	//  if models(customized) are provided,then return it
	if llm.Spec.Models != nil && len(llm.Spec.Models) != 0 {
		return llm.Spec.Models
	}

	switch llm.Spec.Type {
	case llms.ZhiPuAI:
		return llms.ZhiPuAIModels
	case llms.OpenAI:
		return llms.OpenAIModels
	}
	return []string{}
}

func (llm LLM) ReadyCondition(msg string) Condition {
	currCon := llm.Status.GetCondition(TypeReady)
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

func (llm LLM) ErrorCondition(msg string) Condition {
	currCon := llm.Status.GetCondition(TypeReady)
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
