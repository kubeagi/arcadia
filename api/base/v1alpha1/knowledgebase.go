package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// UpdateSourceFileAnnotationKey is the key of the update source file annotation
	UpdateSourceFileAnnotationKey = Group + "/update-source-file-time"
)

func (kb *KnowledgeBase) VectorStoreCollectionName() string {
	return kb.Namespace + "_" + kb.Name
}

func (kb *KnowledgeBase) InitCondition() Condition {
	return Condition{
		Type:               TypeReady,
		Status:             corev1.ConditionUnknown,
		Reason:             "Init",
		Message:            "KnowledgeBase - embedding in progress",
		LastTransitionTime: metav1.Now(),
		LastSuccessfulTime: metav1.Now(),
	}
}

func (kb *KnowledgeBase) PendingCondition(msg string) Condition {
	return Condition{
		Type:               TypeReady,
		Status:             corev1.ConditionFalse,
		Reason:             "Pending",
		Message:            msg,
		LastTransitionTime: metav1.Now(),
		LastSuccessfulTime: metav1.Now(),
	}
}

func (kb *KnowledgeBase) ErrorCondition(msg string) Condition {
	return Condition{
		Type:               TypeReady,
		Status:             corev1.ConditionFalse,
		Reason:             "Error",
		Message:            msg,
		LastTransitionTime: metav1.Now(),
		LastSuccessfulTime: metav1.Now(),
	}
}

func (kb *KnowledgeBase) ReadyCondition() Condition {
	return Condition{
		Type:               TypeReady,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		LastSuccessfulTime: metav1.Now(),
		Message:            "Success",
	}
}

func (f *FileDetails) UpdateErr(err error, phase FileProcessPhase) {
	f.LastUpdateTime = metav1.Now()
	f.Phase = phase
	if err != nil {
		f.ErrMessage = err.Error()
	} else {
		f.ErrMessage = ""
	}
}

func (f *FileGroupDetail) Init(group FileGroup) {
	f.Source = group.Source.DeepCopy()
	f.FileDetails = make([]FileDetails, len(group.Paths))
	for i := range group.Paths {
		f.FileDetails[i].Path = group.Paths[i]
		f.FileDetails[i].Phase = FileProcessPhasePending
		f.FileDetails[i].LastUpdateTime = metav1.Now()
	}
}
