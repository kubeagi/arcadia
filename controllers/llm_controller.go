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
	"reflect"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/llms"
	"github.com/kubeagi/arcadia/pkg/llms/openai"
	"github.com/kubeagi/arcadia/pkg/llms/zhipuai"
)

// LLMReconciler reconciles a LLM object
type LLMReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=llms,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=llms/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=llms/finalizers,verbs=update
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=workers,verbs=get;list;watch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=workers/status,verbs=get
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the LLM object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *LLMReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling LLM resource")

	// Fetch the LLM instance
	instance := &arcadiav1alpha1.LLM{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		// There's no need to requeue if the resource no longer exists.
		// Otherwise, we'll be requeued implicitly because we return an error.
		logger.V(1).Info("Failed to get LLM")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Add a finalizer.Then, we can define some operations which should
	// occur before the LLM to be deleted.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers
	if newAdded := controllerutil.AddFinalizer(instance, arcadiav1alpha1.Finalizer); newAdded {
		logger.Info("Try to add Finalizer for LLM")
		if err := r.Update(ctx, instance); err != nil {
			logger.Error(err, "Failed to update LLM to add finalizer, will try again later")
			return ctrl.Result{}, err
		}
		logger.Info("Adding Finalizer for LLM done")
		return ctrl.Result{Requeue: true}, nil
	}

	// Check if the LLM instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if instance.GetDeletionTimestamp() != nil && controllerutil.ContainsFinalizer(instance, arcadiav1alpha1.Finalizer) {
		logger.Info("Performing Finalizer Operations for LLM before delete CR")
		// TODO perform the finalizer operations here, for example: remove data?
		logger.Info("Removing Finalizer for LLM after successfully performing the operations")
		controllerutil.RemoveFinalizer(instance, arcadiav1alpha1.Finalizer)
		if err := r.Update(ctx, instance); err != nil {
			logger.Error(err, "Failed to remove finalizer for LLM")
			return ctrl.Result{}, err
		}
		logger.Info("Remove LLM done")
		return ctrl.Result{}, nil
	}

	err := r.CheckLLM(ctx, logger, instance)
	if err != nil {
		logger.Error(err, "Failed to check LLM")
		// Update conditioned status
		return ctrl.Result{RequeueAfter: waitMedium}, err
	}

	return ctrl.Result{RequeueAfter: waitLonger}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LLMReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arcadiav1alpha1.LLM{}, builder.WithPredicates(LLMPredicates{})).
		Complete(r)
}

// CheckLLM updates new LLM instance.
func (r *LLMReconciler) CheckLLM(ctx context.Context, logger logr.Logger, instance *arcadiav1alpha1.LLM) error {
	logger.Info("Checking LLM instance")

	switch instance.Spec.Provider.GetType() {
	case arcadiav1alpha1.ProviderType3rdParty:
		return r.check3rdPartyLLM(ctx, logger, instance)
	case arcadiav1alpha1.ProviderTypeWorker:
		return r.checkWorkerLLM(ctx, logger, instance)
	}

	return nil
}

func (r *LLMReconciler) check3rdPartyLLM(ctx context.Context, logger logr.Logger, instance *arcadiav1alpha1.LLM) error {
	logger.Info("Checking 3rd party LLM resource")

	var err error
	var msg string

	// Check Auth availability
	apiKey, err := instance.AuthAPIKey(ctx, r.Client, nil)
	if err != nil {
		return r.UpdateStatus(ctx, instance, nil, err)
	}

	switch instance.Spec.Type {
	case llms.ZhiPuAI:
		embedClient := zhipuai.NewZhiPuAI(apiKey)
		res, err := embedClient.Validate()
		if err != nil {
			return r.UpdateStatus(ctx, instance, nil, err)
		}
		msg = res.String()
	case llms.OpenAI:
		embedClient := openai.NewOpenAI(apiKey, instance.Spec.Enpoint.URL)
		res, err := embedClient.Validate()
		if err != nil {
			return r.UpdateStatus(ctx, instance, nil, err)
		}
		msg = res.String()

	default:
		return r.UpdateStatus(ctx, instance, nil, fmt.Errorf("unsupported service type: %s", instance.Spec.Type))
	}

	return r.UpdateStatus(ctx, instance, msg, err)
}

func (r *LLMReconciler) checkWorkerLLM(ctx context.Context, logger logr.Logger, instance *arcadiav1alpha1.LLM) error {
	logger.Info("Checking Worker's LLM resource")

	var err error
	var msg = "Worker is Ready"

	worker := &arcadiav1alpha1.Worker{}
	err = r.Client.Get(ctx, types.NamespacedName{Namespace: instance.Namespace, Name: instance.Spec.Worker.Name}, worker)
	if err != nil {
		return r.UpdateStatus(ctx, instance, "", err)
	}
	if !worker.Status.IsReady() {
		return r.UpdateStatus(ctx, instance, nil, errors.New("worker is not ready"))
	}

	return r.UpdateStatus(ctx, instance, msg, err)
}

func (r *LLMReconciler) UpdateStatus(ctx context.Context, instance *arcadiav1alpha1.LLM, t interface{}, err error) error {
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

type LLMPredicates struct {
	predicate.Funcs
}

func (llm LLMPredicates) Create(ce event.CreateEvent) bool {
	prompt := ce.Object.(*arcadiav1alpha1.LLM)
	return len(prompt.Status.ConditionedStatus.Conditions) == 0
}

func (llm LLMPredicates) Update(ue event.UpdateEvent) bool {
	oldLLM := ue.ObjectOld.(*arcadiav1alpha1.LLM)
	newLLM := ue.ObjectNew.(*arcadiav1alpha1.LLM)

	return !reflect.DeepEqual(oldLLM.Spec, newLLM.Spec)
}
