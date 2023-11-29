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
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/embeddings"
	"github.com/kubeagi/arcadia/pkg/llms/openai"
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
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=workers,verbs=get;list;watch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=workers/status,verbs=get
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list

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
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		// There's no need to requeue if the resource no longer exists.
		// Otherwise, we'll be requeued implicitly because we return an error.
		logger.V(1).Info("Failed to get Embedder")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Add a finalizer.Then, we can define some operations which should
	// occur before the Embedder to be deleted.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers
	if newAdded := controllerutil.AddFinalizer(instance, arcadiav1alpha1.Finalizer); newAdded {
		logger.Info("Try to add Finalizer for Embedder")
		if err := r.Update(ctx, instance); err != nil {
			logger.Error(err, "Failed to update Embedder to add finalizer, will try again later")
			return ctrl.Result{}, err
		}
		logger.Info("Adding Finalizer for Embedder done")
		return ctrl.Result{}, nil
	}

	// Check if the Embedder instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if instance.GetDeletionTimestamp() != nil && controllerutil.ContainsFinalizer(instance, arcadiav1alpha1.Finalizer) {
		logger.Info("Performing Finalizer Operations for Embedder before delete CR")
		// TODO perform the finalizer operations here, for example: remove data?
		logger.Info("Removing Finalizer for Embedder after successfully performing the operations")
		controllerutil.RemoveFinalizer(instance, arcadiav1alpha1.Finalizer)
		if err := r.Update(ctx, instance); err != nil {
			logger.Error(err, "Failed to remove finalizer for Embedder")
			return ctrl.Result{}, err
		}
		logger.Info("Remove Embedder done")
		return ctrl.Result{}, nil
	}

	if err := r.CheckEmbedder(ctx, logger, instance); err != nil {
		return ctrl.Result{RequeueAfter: waitMedium}, err
	}

	return ctrl.Result{RequeueAfter: waitLonger}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EmbedderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arcadiav1alpha1.Embedder{}).
		Complete(r)
}

func (r *EmbedderReconciler) CheckEmbedder(ctx context.Context, logger logr.Logger, instance *arcadiav1alpha1.Embedder) error {
	logger.Info("Checking embedding resource")

	switch instance.Spec.Provider.GetType() {
	case arcadiav1alpha1.ProviderType3rdParty:
		return r.check3rdPartyEmbedder(ctx, logger, instance)
	case arcadiav1alpha1.ProviderTypeWorker:
		return r.checkWorkerEmbedder(ctx, logger, instance)
	}

	return nil
}

func (r *EmbedderReconciler) check3rdPartyEmbedder(ctx context.Context, logger logr.Logger, instance *arcadiav1alpha1.Embedder) error {
	logger.Info("Checking 3rd party embedding resource")

	var err error
	var msg string

	// Check Auth availability
	apiKey, err := instance.AuthAPIKey(ctx, r.Client)
	if err != nil {
		return r.UpdateStatus(ctx, instance, nil, err)
	}

	switch instance.Spec.ServiceType {
	case embeddings.ZhiPuAI:
		embedClient := zhipuai.NewZhiPuAI(apiKey)
		res, err := embedClient.Validate()
		if err != nil {
			return r.UpdateStatus(ctx, instance, nil, err)
		}
		msg = res.String()
	case embeddings.OpenAI:
		embedClient := openai.NewOpenAI(apiKey, instance.Spec.Enpoint.URL)
		res, err := embedClient.Validate()
		if err != nil {
			return r.UpdateStatus(ctx, instance, nil, err)
		}
		msg = res.String()

	default:
		return r.UpdateStatus(ctx, instance, nil, fmt.Errorf("unsupported service type: %s", instance.Spec.ServiceType))
	}

	return r.UpdateStatus(ctx, instance, msg, err)
}

func (r *EmbedderReconciler) checkWorkerEmbedder(ctx context.Context, logger logr.Logger, instance *arcadiav1alpha1.Embedder) error {
	logger.Info("Checking Worker's embedding resource")

	var err error
	var msg = "Worker is Ready"

	worker := &arcadiav1alpha1.Worker{}
	err = r.Client.Get(ctx, types.NamespacedName{Namespace: instance.Namespace, Name: instance.Spec.Worker.Name}, worker)
	if err != nil {
		return r.UpdateStatus(ctx, instance, nil, err)
	}
	if !worker.Status.IsReady() {
		return r.UpdateStatus(ctx, instance, nil, errors.New("worker is not ready"))
	}

	return r.UpdateStatus(ctx, instance, msg, err)
}

func (r *EmbedderReconciler) UpdateStatus(ctx context.Context, instance *arcadiav1alpha1.Embedder, t interface{}, err error) error {
	instanceCopy := instance.DeepCopy()
	var newCondition arcadiav1alpha1.Condition
	if err != nil {
		// Set status to unavailable
		newCondition = arcadiav1alpha1.Condition{
			Type:               arcadiav1alpha1.TypeReady,
			Status:             corev1.ConditionFalse,
			Reason:             arcadiav1alpha1.ReasonUnavailable,
			Message:            err.Error(),
			LastTransitionTime: metav1.Now(),
		}
	} else {
		msg, ok := t.(string)
		if !ok {
			msg = _StatusNilResponse
		}
		// Set status to available
		newCondition = arcadiav1alpha1.Condition{
			Type:               arcadiav1alpha1.TypeReady,
			Status:             corev1.ConditionTrue,
			Reason:             arcadiav1alpha1.ReasonAvailable,
			Message:            msg,
			LastTransitionTime: metav1.Now(),
			LastSuccessfulTime: metav1.Now(),
		}
	}
	instanceCopy.Status.SetConditions(newCondition)
	return r.Client.Status().Update(ctx, instanceCopy)
}
