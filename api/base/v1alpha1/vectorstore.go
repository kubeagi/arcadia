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
)

const (
	LabelVectorStoreType = Group + "/vectorstore-type"
)

type VectorStoreType string

const (
	VectorStoreTypeChroma  VectorStoreType = "chroma"
	VectorStoreTypeUnknown VectorStoreType = "unknown"
)

func (vs VectorStoreSpec) Type() VectorStoreType {
	if vs.Enpoint == nil {
		return VectorStoreTypeUnknown
	}

	if vs.Chroma != nil {
		return VectorStoreTypeChroma
	}

	return VectorStoreTypeUnknown
}

func (vs *VectorStore) InitCondition() Condition {
	return Condition{
		Type:               TypeReady,
		Status:             corev1.ConditionUnknown,
		LastTransitionTime: metav1.Now(),
		LastSuccessfulTime: metav1.Now(),
		Reason:             "Init",
		Message:            "Reconciliation in progress",
	}
}

func (vs *VectorStore) PendingCondition(msg string) Condition {
	return Condition{
		Type:               TypeReady,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		LastSuccessfulTime: metav1.Now(),
		Reason:             "Pending",
		Message:            msg,
	}
}

func (vs *VectorStore) ErrorCondition(msg string) Condition {
	return Condition{
		Type:               TypeReady,
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		LastSuccessfulTime: metav1.Now(),
		Reason:             "Error",
		Message:            msg,
	}
}

func (vs *VectorStore) ReadyCondition() Condition {
	return Condition{
		Type:               TypeReady,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		LastSuccessfulTime: metav1.Now(),
		Message:            "Success",
	}
}
