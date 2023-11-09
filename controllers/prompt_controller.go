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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/llms"
	"github.com/kubeagi/arcadia/pkg/llms/openai"
	llmszhipuai "github.com/kubeagi/arcadia/pkg/llms/zhipuai"
)

// PromptReconciler reconciles a Prompt object
type PromptReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=prompts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=prompts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=prompts/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Prompt object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *PromptReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Starting prompt reconcile")

	// Prompt engineering
	prompt := &arcadiav1alpha1.Prompt{}
	if err := r.Get(ctx, req.NamespacedName, prompt); err != nil {
		// There's no need to requeue if the resource no longer exists.
		// Otherwise, we'll be requeued implicitly because we return an error.
		logger.V(1).Info("Failed to get Prompt")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Add a finalizer.Then, we can define some operations which should
	// occur before the Prompt to be deleted.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers
	if newAdded := controllerutil.AddFinalizer(prompt, arcadiav1alpha1.Finalizer); newAdded {
		logger.Info("Try to add Finalizer for Prompt")
		if err := r.Update(ctx, prompt); err != nil {
			logger.Error(err, "Failed to update Prompt to add finalizer, will try again later")
			return ctrl.Result{}, err
		}
		logger.Info("Adding Finalizer for Prompt done")
		return ctrl.Result{Requeue: true}, nil
	}

	// Check if the Prompt instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if prompt.GetDeletionTimestamp() != nil && controllerutil.ContainsFinalizer(prompt, arcadiav1alpha1.Finalizer) {
		logger.Info("Performing Finalizer Operations for Prompt before delete CR")
		// TODO perform the finalizer operations here, for example: remove data?
		logger.Info("Removing Finalizer for Prompt after successfully performing the operations")
		controllerutil.RemoveFinalizer(prompt, arcadiav1alpha1.Finalizer)
		if err := r.Update(ctx, prompt); err != nil {
			logger.Error(err, "Failed to remove finalizer for Prompt")
			return ctrl.Result{}, err
		}
		logger.Info("Remove Prompt done")
		return ctrl.Result{}, nil
	}

	err := r.CallLLM(ctx, logger, prompt)
	if err != nil {
		logger.Error(err, "Failed to call LLM")
		// Update conditioned status
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PromptReconciler) CallLLM(ctx context.Context, logger logr.Logger, prompt *arcadiav1alpha1.Prompt) error {
	llm := &arcadiav1alpha1.LLM{}
	if err := r.Get(ctx, types.NamespacedName{Name: prompt.Spec.LLM, Namespace: prompt.Namespace}, llm); err != nil {
		return err
	}

	apiKey, err := llm.AuthAPIKey(ctx, r.Client)
	if err != nil {
		return r.UpdateStatus(ctx, prompt, nil, err)
	}

	// llm call
	var llmClient llms.LLM
	var callData []byte
	switch llm.Spec.Type {
	case llms.ZhiPuAI:
		llmClient = llmszhipuai.NewZhiPuAI(apiKey)
		callData = prompt.Spec.ZhiPuAIParams.Marshal()
	case llms.OpenAI:
		llmClient = openai.NewOpenAI(apiKey, llm.Spec.Enpoint.URL)
	default:
		llmClient = llms.NewUnknowLLM()
	}

	resp, err := llmClient.Call(callData)
	if err != nil {
		return err
	}

	return r.UpdateStatus(ctx, prompt, resp, err)
}

func (r *PromptReconciler) UpdateStatus(ctx context.Context, prompt *arcadiav1alpha1.Prompt, response llms.Response, err error) error {
	promptDeepCodpy := prompt.DeepCopy()
	newCond := arcadiav1alpha1.Condition{
		Type:               arcadiav1alpha1.TypeDone,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             arcadiav1alpha1.ReasonReconcileSuccess,
		Message:            "Finished CallLLM",
	}
	if err != nil {
		newCond.Status = corev1.ConditionFalse
		newCond.Reason = arcadiav1alpha1.ReasonReconcileError
		newCond.Message = err.Error()
	}
	promptDeepCodpy.Status.SetConditions(newCond)
	if response != nil {
		promptDeepCodpy.Status.Data = response.Bytes()
	}
	return r.Status().Update(ctx, promptDeepCodpy)
}

// SetupWithManager sets up the controller with the Manager.
func (r *PromptReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arcadiav1alpha1.Prompt{}, builder.WithPredicates(PromptPredicates{})).
		Complete(r)
}

type PromptPredicates struct {
	predicate.Funcs
}

func (p PromptPredicates) Create(ce event.CreateEvent) bool {
	prompt := ce.Object.(*arcadiav1alpha1.Prompt)
	return len(prompt.Status.ConditionedStatus.Conditions) == 0
}

func (p PromptPredicates) Update(ue event.UpdateEvent) bool {
	oldPrompt := ue.ObjectOld.(*arcadiav1alpha1.Prompt)
	newPrompt := ue.ObjectNew.(*arcadiav1alpha1.Prompt)

	return !reflect.DeepEqual(oldPrompt.Spec, newPrompt.Spec)
}
