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
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// LabelModelEmbedding indicates this is a embedding model
	LabelModelEmbedding = Group + "/embedding"
	// LabelModelLLM indicates this is a llm model
	LabelModelLLM = Group + "/llm"
	// LabelModelReranking indicates this is a reranking model
	LabelModelReranking = Group + "/reranking"
	// LabelModelFullPath indicates the full path in storage
	LabelModelFullPath = Group + "/full-path"
)

// IsLLMModel checks whether this model is a llm model
func (model Model) IsLLMModel() bool {
	for _, t := range strings.Split(model.Spec.Types, ",") {
		if strings.ToLower(t) == "llm" {
			return true
		}
	}
	return false
}

// IsLLMModel checks whether this model is a embedding model
func (model Model) IsEmbeddingModel() bool {
	for _, t := range strings.Split(model.Spec.Types, ",") {
		if strings.ToLower(t) == "embedding" {
			return true
		}
	}
	return false
}

// IsRerankingModel checks whether this model is a reranking model
func (model Model) IsRerankingModel() bool {
	for _, t := range strings.Split(model.Spec.Types, ",") {
		if strings.ToLower(t) == "reranking" {
			return true
		}
	}
	return false
}

// FullPath with bucket and object path
func (model Model) FullPath() string {
	return fmt.Sprintf("%s/%s", model.Namespace, model.ObjectPath())
}

// ObjectPath is the path where model stored at in a bucket
func (model Model) ObjectPath() string {
	return fmt.Sprintf("model/%s/", model.Name)
}

func (model Model) ReadyCondition() Condition {
	currCon := model.Status.GetCondition(TypeReady)
	// return current condition if condition not changed
	if currCon.Status == corev1.ConditionTrue && currCon.Reason == ReasonAvailable {
		return currCon
	}
	return Condition{
		Type:               TypeReady,
		Status:             corev1.ConditionTrue,
		Reason:             ReasonAvailable,
		Message:            "Check Success",
		LastTransitionTime: metav1.Now(),
		LastSuccessfulTime: metav1.Now(),
	}
}

func (model Model) ErrorCondition(msg string) Condition {
	currCon := model.Status.GetCondition(TypeReady)
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
