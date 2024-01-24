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
	LabelDatasourceType = Group + "/datasource-type"
)

type DatasourceType string

const (
	DatasourceTypeOSS        DatasourceType = "oss"
	DatasourceTypeRDMA       DatasourceType = "RDMA"
	DatasourceTypePostgreSQL DatasourceType = "postgresql"
	DatasourceTypeWeb        DatasourceType = "web"
	DatasourceTypeUnknown    DatasourceType = "unknown"
)

func (ds DatasourceSpec) Type() DatasourceType {
	switch {
	case ds.OSS != nil:
		return DatasourceTypeOSS
	case ds.RDMA != nil:
		return DatasourceTypeRDMA
	case ds.PostgreSQL != nil:
		return DatasourceTypePostgreSQL
	case ds.Web != nil:
		return DatasourceTypeWeb
	default:
		return DatasourceTypeUnknown
	}
}

func (datasource Datasource) ReadyCondition() Condition {
	currCon := datasource.Status.GetCondition(TypeReady)
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

func (datasource Datasource) ErrorCondition(msg string) Condition {
	currCon := datasource.Status.GetCondition(TypeReady)
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
