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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeagi/arcadia/pkg/embeddings"
	"github.com/kubeagi/arcadia/pkg/llms"
)

type WorkerType string

const (
	WorkerTypeFastchatNormal WorkerType = "fastchat"
	WorkerTypeFastchatVLLM   WorkerType = "fastchat-vllm"
	WorkerTypeUnknown        WorkerType = "unknown"
)

const (
	LabelWorkerType = Group + "/worker-type"

	// Labels for worker's Pod
	WorkerPodSelectorLabel = "app.kubernetes.io/name"
	WorkerPodLabel         = Group + "/worker"
)

func DefaultWorkerType() WorkerType {
	return WorkerTypeFastchatNormal
}

func (worker Worker) Type() WorkerType {
	if worker.Spec.Type == "" {
		// use `fastchat` by default
		return WorkerTypeFastchatNormal
	}
	return worker.Spec.Type
}

func (worker Worker) Model() TypedObjectReference {
	if worker.Spec.Model == nil {
		return TypedObjectReference{}
	}
	modelNs := worker.Namespace
	if worker.Spec.Model.Namespace != nil {
		modelNs = *worker.Spec.Model.Namespace
	}
	return TypedObjectReference{
		Kind:      "Model",
		Name:      worker.Spec.Model.Name,
		Namespace: &modelNs,
	}
}

// MakeRegistrationModelName generates a model name used to register itself into fastchat controller
func (worker Worker) MakeRegistrationModelName() string {
	return string(worker.UID)
}

func (worker Worker) PendingCondition() Condition {
	currCon := worker.Status.GetCondition(TypeReady)
	// return current condition if condition not changed
	if currCon.Status == corev1.ConditionFalse && currCon.Reason == "Pending" {
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
		Reason:             "Pending",
		Message:            "Worker is pending",
		LastTransitionTime: metav1.Now(),
		LastSuccessfulTime: lastSuccessfulTime,
	}
}

func (worker Worker) ReadyCondition() Condition {
	currCon := worker.Status.GetCondition(TypeReady)
	// return current condition if condition not changed
	if currCon.Status == corev1.ConditionTrue && currCon.Reason == "Running" {
		return currCon
	}
	return Condition{
		Type:               TypeReady,
		Status:             corev1.ConditionTrue,
		Reason:             "Running",
		Message:            "Work has been actively running",
		LastTransitionTime: metav1.Now(),
		LastSuccessfulTime: metav1.Now(),
	}
}

func (worker Worker) OfflineCondition() Condition {
	currCon := worker.Status.GetCondition(TypeReady)
	// return current condition if condition not changed
	if currCon.Status == corev1.ConditionTrue && currCon.Reason == "Offline" {
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
		Reason:             "Offline",
		Message:            "Work is offline",
		LastTransitionTime: metav1.Now(),
		LastSuccessfulTime: lastSuccessfulTime,
	}
}

func (worker Worker) ErrorCondition(msg string) Condition {
	currCon := worker.Status.GetCondition(TypeReady)
	// return current condition if condition not changed
	if currCon.Status == corev1.ConditionFalse && currCon.Reason == "Error" && currCon.Message == msg {
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
		Reason:             "Error",
		Message:            msg,
		LastSuccessfulTime: lastSuccessfulTime,
		LastTransitionTime: metav1.Now(),
	}
}

func (worker Worker) BuildEmbedder() *Embedder {
	return &Embedder{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: worker.Namespace,
			Name:      worker.Name,
		},
		Spec: EmbedderSpec{
			CommonSpec: CommonSpec{
				Creator: worker.Spec.Creator,
				// Use the model name as the displayname
				DisplayName: worker.Spec.Model.Name,
				Description: "Embedder created by Worker(OpenAI compatible)",
			},
			Type: embeddings.OpenAI,
			Provider: Provider{
				Worker: &TypedObjectReference{
					Kind:      "Worker",
					Namespace: &worker.Namespace,
					Name:      worker.Name,
				},
			},
		},
	}
}

func (worker Worker) BuildLLM() *LLM {
	return &LLM{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: worker.Namespace,
			Name:      worker.Name,
		},
		Spec: LLMSpec{
			CommonSpec: CommonSpec{
				Creator: worker.Spec.Creator,
				// Use the model name as the displayname
				DisplayName: worker.Spec.Model.Name,
				Description: "LLM created by Worker(OpenAI compatible)",
			},
			Type: llms.OpenAI,
			Provider: Provider{
				Worker: &TypedObjectReference{
					Kind:      "Worker",
					Namespace: &worker.Namespace,
					Name:      worker.Name,
				},
			},
		},
	}
}
