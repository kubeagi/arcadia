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
	langchainembeddings "github.com/tmc/langchaingo/embeddings"
	langchainopenai "github.com/tmc/langchaingo/llms/openai"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	"github.com/kubeagi/arcadia/pkg/embeddings"
	embeddingszhipuai "github.com/kubeagi/arcadia/pkg/embeddings/zhipuai"
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
	logger.V(5).Info("Reconciling embedding resource")

	instance := &arcadiav1alpha1.Embedder{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		// There's no need to requeue if the resource no longer exists.
		// Otherwise, we'll be requeued implicitly because we return an error.
		logger.Error(err, "Failed to get Embedder")
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
		return ctrl.Result{Requeue: true}, nil
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
		For(&arcadiav1alpha1.Embedder{}, builder.WithPredicates(predicate.Funcs{
			UpdateFunc: func(ue event.UpdateEvent) bool {
				// Avoid to handle the event that it's not spec update or delete
				oldEmbedder := ue.ObjectOld.(*arcadiav1alpha1.Embedder)
				newEmbedder := ue.ObjectNew.(*arcadiav1alpha1.Embedder)
				return !reflect.DeepEqual(oldEmbedder.Spec, newEmbedder.Spec) || newEmbedder.DeletionTimestamp != nil
			},
			// for other event handler, we must add the function explicitly.
			CreateFunc: func(event.CreateEvent) bool {
				return true
			},
			DeleteFunc: func(event.DeleteEvent) bool {
				return true
			},
			GenericFunc: func(event.GenericEvent) bool {
				return true
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
				if m.IsEmbeddingModel() {
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

func (r *EmbedderReconciler) CheckEmbedder(ctx context.Context, logger logr.Logger, instance *arcadiav1alpha1.Embedder) error {
	logger.V(5).Info("Checking embedding resource")

	switch instance.Spec.Provider.GetType() {
	case arcadiav1alpha1.ProviderType3rdParty:
		return r.check3rdPartyEmbedder(ctx, logger, instance)
	case arcadiav1alpha1.ProviderTypeWorker:
		return r.checkWorkerEmbedder(ctx, logger, instance)
	}

	return nil
}

func (r *EmbedderReconciler) check3rdPartyEmbedder(ctx context.Context, logger logr.Logger, instance *arcadiav1alpha1.Embedder) error {
	logger.V(5).Info("Checking 3rd party embedding resource")

	var err error
	var msg string

	// Check Auth availability
	apiKey, err := instance.AuthAPIKey(ctx, r.Client, nil)
	if err != nil {
		return r.UpdateStatus(ctx, instance, nil, err)
	}

	// embedding models provided by 3rd_party
	models := instance.Get3rdPartyModels()
	if len(models) == 0 {
		return r.UpdateStatus(ctx, instance, nil, errors.New("no models provided by this embedder"))
	}

	embedingText := "validate embedding"
	switch instance.Spec.Type {
	case embeddings.ZhiPuAI:
		embedClient, err := embeddingszhipuai.NewZhiPuAI(embeddingszhipuai.WithClient(*zhipuai.NewZhiPuAI(apiKey)))
		if err != nil {
			return r.UpdateStatus(ctx, instance, nil, err)
		}
		_, err = embedClient.EmbedQuery(ctx, embedingText)
		if err != nil {
			return r.UpdateStatus(ctx, instance, nil, err)
		}
		msg = "Success"
	case embeddings.OpenAI:
		// validate all embedding models
		for _, model := range models {
			llm, err := langchainopenai.New(
				langchainopenai.WithBaseURL(instance.Spec.Endpoint.URL),
				langchainopenai.WithToken(apiKey),
				langchainopenai.WithModel(model),
			)
			if err != nil {
				return r.UpdateStatus(ctx, instance, nil, err)
			}
			embedClient, err := langchainembeddings.NewEmbedder(llm)
			if err != nil {
				return r.UpdateStatus(ctx, instance, nil, err)
			}
			_, err = embedClient.EmbedQuery(ctx, embedingText)
			if err != nil {
				return r.UpdateStatus(ctx, instance, nil, err)
			}
			msg = "Success"
		}
	default:
		return r.UpdateStatus(ctx, instance, nil, fmt.Errorf("unsupported service type: %s", instance.Spec.Type))
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
		if worker.Status.IsOffline() {
			return r.UpdateStatus(ctx, instance, nil, errors.New("worker is offline"))
		}
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
