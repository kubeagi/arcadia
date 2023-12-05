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
	"crypto/sha256"
	"encoding/hex"

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
)

func (worker Worker) Type() WorkerType {
	if worker.Spec.Type == "" {
		// use `fastchat` by default
		return WorkerTypeFastchatNormal
	}
	return worker.Spec.Type
}

func (worker Worker) PendingCondition() Condition {
	return Condition{
		Type:               TypeReady,
		Status:             corev1.ConditionFalse,
		Reason:             "Pending",
		Message:            "Worker is pending",
		LastTransitionTime: metav1.Now(),
		LastSuccessfulTime: metav1.Now(),
	}
}

func (worker Worker) ReadyCondition() Condition {
	return Condition{
		Type:               TypeReady,
		Status:             corev1.ConditionTrue,
		Reason:             "WorkerRunning",
		Message:            "Work has been actively running",
		LastTransitionTime: metav1.Now(),
		LastSuccessfulTime: metav1.Now(),
	}
}

func (worker Worker) ErrorCondition(msg string) Condition {
	return Condition{
		Type:               TypeReady,
		Status:             corev1.ConditionFalse,
		Reason:             "Error",
		Message:            msg,
		LastTransitionTime: metav1.Now(),
		LastSuccessfulTime: metav1.Now(),
	}
}

func (worker Worker) generateUniqueName() string {
	// Create a new SHA-256 hasher
	hasher := sha256.New()

	// Write the input string to the hasher
	hasher.Write([]byte(worker.Name))

	// Calculate the hash sum
	hashSum := hasher.Sum(nil)

	// Convert the hash sum to a hexadecimal string
	hashString := hex.EncodeToString(hashSum)

	return hashString[:10]
}

func (worker Worker) BuildEmbedder() *Embedder {
	return &Embedder{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: worker.Namespace,
			Name:      worker.generateUniqueName() + "-worker",
		},
		Spec: EmbedderSpec{
			CommonSpec: CommonSpec{
				Creator:     worker.Spec.Creator,
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
			Name:      worker.generateUniqueName() + "-worker",
		},
		Spec: LLMSpec{
			CommonSpec: CommonSpec{
				Creator:     worker.Spec.Creator,
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
