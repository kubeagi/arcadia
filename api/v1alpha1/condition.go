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
	"sort"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// A ConditionType represents a condition a resource could be in.
type ConditionType string

// Some common Condition types.
const (
	// TypeReady resources are believed to be ready to handle work.
	TypeReady ConditionType = "Ready"
	// TypeUnknown resources are unknown to the system
	TypeUnknown ConditionType = "Unknown"
	// TypeDone resources are believed to be processed
	TypeDone ConditionType = "Done"
)

// A ConditionReason represents the reason a resource is in a condition.
// Should be only one word
type ConditionReason string

// Some common Condition reasons.
const (
	ReasonAvailable        ConditionReason = "Available"
	ReasonUnavailable      ConditionReason = "Unavailable"
	ReasonCreating         ConditionReason = "Creating"
	ReasonDeleting         ConditionReason = "Deleting"
	ReasonReconcileSuccess ConditionReason = "ReconcileSuccess"
	ReasonReconcileError   ConditionReason = "ReconcileError"
	ReasonReconcilePaused  ConditionReason = "ReconcilePaused"
)

// Some Data related Condition Types
const (
	// Dataset have 3 phases: load -> process -> publish
	// TypeLoaded resources are believed to be loaded
	TypeLoaded ConditionType = "Loaded"
	// TypeProcessed resources are believed to be processed
	TypeProcessed ConditionType = "Processed"
	// TypePublished resources are believed to be published
	TypePublished ConditionType = "Published"
)

// Some Dataset related Condition reasons
const (
	// Load data
	ReasonDataLoading     ConditionReason = "DataLoading"
	ReasonDataLoadError   ConditionReason = "DataLoadError"
	ReasonDataLoadSuccess ConditionReason = "DataLoadSuccess"
	// Process data
	ReasonDataProcessing     ConditionReason = "DataProcessing"
	ReasonDataProcessError   ConditionReason = "DataProcessError"
	ReasonDataProcessSuccess ConditionReason = "DataProcessSuccess"
	// Publish dataset
	ReasonDatasetUnpublished ConditionReason = "DatasetUnpublished"
	ReasonDatasetPublished   ConditionReason = "DatasetPublished"
)

// A Condition that may apply to a resource.
type Condition struct {
	// Type of this condition. At most one of each condition type may apply to
	// a resource at any point in time.
	Type ConditionType `json:"type"`

	// Status of this condition; is it currently True, False, or Unknown
	Status corev1.ConditionStatus `json:"status"`

	// LastTransitionTime is the last time this condition transitioned from one
	// status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`

	// LastSuccessfulTime is repository Last Successful Update Time
	LastSuccessfulTime metav1.Time `json:"lastSuccessfulTime,omitempty"`

	// A Reason for this condition's last transition from one status to another.
	Reason ConditionReason `json:"reason"`

	// A Message containing details about this condition's last transition from
	// one status to another, if any.
	// +optional
	Message string `json:"message,omitempty"`
}

// Equal returns true if the condition is identical to the supplied condition
func (c Condition) Equal(other Condition) bool {
	return c.Type == other.Type &&
		c.Status == other.Status &&
		c.Reason == other.Reason &&
		c.Message == other.Message &&
		c.LastSuccessfulTime.Equal(&other.LastSuccessfulTime) &&
		c.LastTransitionTime.Equal(&other.LastTransitionTime)
}

// WithMessage returns a condition by adding the provided message to existing
// condition.
func (c Condition) WithMessage(msg string) Condition {
	c.Message = msg
	return c
}

// NOTE: Conditions are implemented as a slice rather than a map to comply
// with Kubernetes API conventions. Ideally we'd comply by using a map that
// marshaled to a JSON array, but doing so confuses the CRD schema generator.
// https://github.com/kubernetes/community/blob/9bf8cd/contributors/devel/sig-architecture/api-conventions.md#lists-of-named-subobjects-preferred-over-maps

// NOTE: Do not manipulate Conditions directly. Use the Set method.

// A ConditionedStatus reflects the observed status of a resource. Only
// one condition of each type may exist.
type ConditionedStatus struct {
	// Conditions of the resource.
	// +optional
	Conditions []Condition `json:"conditions,omitempty"`
}

// NewConditionedStatus returns a stat with the supplied conditions set.
func NewConditionedStatus(c ...Condition) *ConditionedStatus {
	s := &ConditionedStatus{}
	s.SetConditions(c...)
	return s
}

// GetCondition returns the condition for the given ConditionType if exists,
// otherwise returns nil
func (s *ConditionedStatus) GetCondition(ct ConditionType) Condition {
	for _, c := range s.Conditions {
		if c.Type == ct {
			return c
		}
	}

	return Condition{Type: ct, Status: corev1.ConditionUnknown}
}

// SetConditions sets the supplied conditions, replacing any existing conditions
// of the same type. This is a no-op if all supplied conditions are identical,
// ignoring the last transition time, to those already set.
func (s *ConditionedStatus) SetConditions(c ...Condition) {
	for _, new := range c {
		exists := false
		for i, existing := range s.Conditions {
			if existing.Type != new.Type {
				continue
			}

			if existing.Equal(new) {
				exists = true
				continue
			}

			s.Conditions[i] = new
			exists = true
		}
		if !exists {
			s.Conditions = append(s.Conditions, new)
		}
	}
}

// Equal returns true if the status is identical to the supplied status,
// ignoring the LastTransitionTimes and order of statuses.
func (s *ConditionedStatus) Equal(other *ConditionedStatus) bool {
	if s == nil || other == nil {
		return s == nil && other == nil
	}

	if len(other.Conditions) != len(s.Conditions) {
		return false
	}

	sc := make([]Condition, len(s.Conditions))
	copy(sc, s.Conditions)

	oc := make([]Condition, len(other.Conditions))
	copy(oc, other.Conditions)

	// We should not have more than one condition of each type.
	sort.Slice(sc, func(i, j int) bool { return sc[i].Type < sc[j].Type })
	sort.Slice(oc, func(i, j int) bool { return oc[i].Type < oc[j].Type })

	for i := range sc {
		if !sc[i].Equal(oc[i]) {
			return false
		}
	}

	return true
}

func (s *ConditionedStatus) IsReady() bool {
	return s.GetCondition(TypeReady).Status == corev1.ConditionTrue
}
