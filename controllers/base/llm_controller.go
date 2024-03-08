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
	"strings"

	"github.com/go-logr/logr"
	langchainllms "github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/googleai"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

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
	if instance.Labels == nil {
		instance.Labels = make(map[string]string)
	}
	providerType := instance.Spec.Provider.GetType()
	if _type, ok := instance.Labels[arcadiav1alpha1.ProviderLabel]; !ok || _type != string(providerType) {
		instance.Labels[arcadiav1alpha1.ProviderLabel] = string(providerType)
		err := r.Client.Update(ctx, instance)
		if err != nil {
			logger.Error(err, "failed to update llm labels", "providerType", providerType)
		}
		return ctrl.Result{Requeue: true}, err
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
		For(&arcadiav1alpha1.LLM{}, builder.WithPredicates(predicate.Funcs{
			UpdateFunc: func(ue event.UpdateEvent) bool {
				// Avoid to handle the event that it's not spec update or delete
				oldLLM := ue.ObjectOld.(*arcadiav1alpha1.LLM)
				newLLM := ue.ObjectNew.(*arcadiav1alpha1.LLM)
				return !reflect.DeepEqual(oldLLM.Spec, newLLM.Spec) || newLLM.DeletionTimestamp != nil
			},
		})).
		Watches(&source.Kind{Type: &arcadiav1alpha1.Worker{}},
			handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
				worker := o.(*arcadiav1alpha1.Worker)
				model := worker.Spec.Model.DeepCopy()
				if model.Namespace == nil {
					model.Namespace = &worker.Namespace
				}
				m := &arcadiav1alpha1.Model{}
				if err := r.Client.Get(context.TODO(), types.NamespacedName{Namespace: *model.Namespace, Name: model.Name}, m); err != nil {
					return []ctrl.Request{}
				}
				if m.IsLLMModel() {
					return []ctrl.Request{
						reconcile.Request{
							NamespacedName: client.ObjectKeyFromObject(o),
						},
					}
				}
				return []ctrl.Request{}
			})).
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
	apiKey, err := instance.AuthAPIKey(ctx, r.Client)
	if err != nil {
		return r.UpdateStatus(ctx, instance, nil, err)
	}

	models := instance.Get3rdPartyModels()
	if len(models) == 0 {
		return r.UpdateStatus(ctx, instance, nil, errors.New("no models provided by this embedder"))
	}

	switch instance.Spec.Type {
	case llms.ZhiPuAI:
		llmClient := zhipuai.NewZhiPuAI(apiKey)
		res, err := llmClient.Validate(ctx)
		if err != nil {
			return r.UpdateStatus(ctx, instance, nil, err)
		}
		msg = res.String()
	case llms.OpenAI:
		llmClient, err := openai.NewOpenAI(apiKey, instance.Spec.Endpoint.URL)
		if err != nil {
			return r.UpdateStatus(ctx, instance, nil, err)
		}
		// validate against models
		for _, model := range models {
			res, err := llmClient.Validate(ctx, langchainllms.WithModel(model))
			if err != nil {
				return r.UpdateStatus(ctx, instance, nil, err)
			}
			msg = strings.Join([]string{msg, res.String()}, "\n")
		}
	case llms.Gemini:
		llmClient, err := googleai.New(ctx, googleai.WithAPIKey(apiKey))
		if err != nil {
			return r.UpdateStatus(ctx, instance, nil, err)
		}
		// validate against models
		for _, model := range models {
			res, err := llmClient.Call(ctx, "Hello", langchainllms.WithModel(model))
			if err != nil {
				return r.UpdateStatus(ctx, instance, nil, err)
			}
			msg = strings.Join([]string{msg, res}, "\n")
		}
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
		if worker.Status.IsOffline() {
			return r.UpdateStatus(ctx, instance, nil, errors.New("worker is offline"))
		}
		return r.UpdateStatus(ctx, instance, nil, errors.New("worker is not ready"))
	}

	return r.UpdateStatus(ctx, instance, msg, err)
}

func (r *LLMReconciler) UpdateStatus(ctx context.Context, instance *arcadiav1alpha1.LLM, t interface{}, err error) error {
	instanceCopy := instance.DeepCopy()
	var newCondition arcadiav1alpha1.Condition
	if err != nil {
		// set condition to False
		newCondition = instance.ErrorCondition(err.Error())
	} else {
		msg, ok := t.(string)
		if !ok {
			msg = _StatusNilResponse
		}
		// set condition to True
		newCondition = instance.ReadyCondition(msg)
	}
	instanceCopy.Status.SetConditions(newCondition)
	return errors.Join(err, r.Client.Status().Update(ctx, instanceCopy))
}
