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

package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/embeddings"
	"github.com/kubeagi/arcadia/pkg/llms/zhipuai"
)

const (
	_StatusNilResponse = "No err replied but response is not string"
)

// EmbedderReconciler reconciles a Embedder object
type EmbedderReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=embedders,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=embedders/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=embedders/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Embedder object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *EmbedderReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling embedding resource")

	instance := &arcadiav1alpha1.Embedder{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	err = r.CheckEmbedder(ctx, logger, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EmbedderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arcadiav1alpha1.Embedder{}).
		Complete(r)
}

func (r *EmbedderReconciler) CheckEmbedder(ctx context.Context, logger logr.Logger, instance *arcadiav1alpha1.Embedder) error {
	logger.Info("Checking embedding resource")
	var err error
	var msg string

	// Check Auth availability
	apiKey, err := instance.AuthAPIKey(ctx, r.Client)
	if err != nil {
		return r.UpdateStatus(ctx, instance, nil, err)
	}

	switch instance.Spec.ServiceType {
	case embeddings.ZhiPuAI:
		{
			embedClient := zhipuai.NewZhiPuAI(apiKey)
			res, err := embedClient.Validate()
			if err != nil {
				return r.UpdateStatus(ctx, instance, nil, err)
			}
			msg = res.String()
		}
	default:
		return r.UpdateStatus(ctx, instance, nil, fmt.Errorf("unsupported service type: %s", instance.Spec.ServiceType))
	}

	return r.UpdateStatus(ctx, instance, msg, err)
}

func (r *EmbedderReconciler) UpdateStatus(ctx context.Context, instance *arcadiav1alpha1.Embedder, t interface{}, err error) error {
	instanceCopy := instance.DeepCopy()
	if err != nil {
		// Set status to unavailable
		instanceCopy.Status.SetConditions(arcadiav1alpha1.Condition{
			Type:               arcadiav1alpha1.TypeReady,
			Status:             corev1.ConditionFalse,
			Reason:             arcadiav1alpha1.ReasonUnavailable,
			Message:            err.Error(),
			LastTransitionTime: metav1.Now(),
		})
	} else {
		msg, ok := t.(string)
		if !ok {
			msg = _StatusNilResponse
		}
		// Set status to available
		instanceCopy.Status.SetConditions(arcadiav1alpha1.Condition{
			Type:               arcadiav1alpha1.TypeReady,
			Status:             corev1.ConditionTrue,
			Reason:             arcadiav1alpha1.ReasonAvailable,
			Message:            msg,
			LastTransitionTime: metav1.Now(),
			LastSuccessfulTime: metav1.Now(),
		})
	}
	return r.Client.Status().Update(ctx, instanceCopy)
}
