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
	"reflect"
	"strconv"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/config"
	"github.com/kubeagi/arcadia/pkg/datasource"
)

// ModelReconciler reconciles a Model object
type ModelReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=models,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=models/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=models/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Model object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *ModelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(5).Info("Starting model reconcile")

	instance := &arcadiav1alpha1.Model{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		// There's no need to requeue if the resource no longer exists.
		// Otherwise, we'll be requeued implicitly because we return an error.
		logger.V(1).Info("Failed to get Model")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Add a finalizer.Then, we can define some operations which should
	// occur before the Model to be deleted.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers
	if newAdded := controllerutil.AddFinalizer(instance, arcadiav1alpha1.Finalizer); newAdded {
		logger.Info("Try to add Finalizer for Model")
		if err := r.Update(ctx, instance); err != nil {
			logger.Error(err, "Failed to update Model to add finalizer, will try again later")
			return ctrl.Result{}, err
		}
		logger.Info("Adding Finalizer for Model done")
		return ctrl.Result{Requeue: true}, nil
	}

	// Check if the Model instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if instance.GetDeletionTimestamp() != nil && controllerutil.ContainsFinalizer(instance, arcadiav1alpha1.Finalizer) {
		logger.Info("Performing Finalizer Operations for Model before delete CR")
		// remove all model files from storage service
		if err := r.RemoveModel(ctx, logger, instance); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to remove model: %w", err)
		}
		logger.Info("Removing Finalizer for Model after successfully performing the operations")
		controllerutil.RemoveFinalizer(instance, arcadiav1alpha1.Finalizer)
		if err := r.Update(ctx, instance); err != nil {
			logger.Error(err, "Failed to remove finalizer for Model")
			return ctrl.Result{}, err
		}
		logger.Info("Remove Model done")
		return ctrl.Result{}, nil
	}

	// initialize labels
	requeue, err := r.Initialize(ctx, logger, instance)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to initialize model: %w", err)
	}
	if requeue {
		return reconcile.Result{Requeue: true}, nil
	}

	if err := r.CheckModel(ctx, logger, instance); err != nil {
		// Update conditioned status
		return reconcile.Result{RequeueAfter: waitMedium}, err
	}

	return ctrl.Result{RequeueAfter: waitLonger}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ModelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arcadiav1alpha1.Model{}, builder.WithPredicates(predicate.Funcs{
			UpdateFunc: func(ue event.UpdateEvent) bool {
				oldModel := ue.ObjectOld.(*arcadiav1alpha1.Model)
				newModel := ue.ObjectNew.(*arcadiav1alpha1.Model)
				return !reflect.DeepEqual(oldModel.Spec, newModel.Spec) || newModel.DeletionTimestamp != nil
			},
		})).
		Complete(r)
}

func (r *ModelReconciler) Initialize(ctx context.Context, logger logr.Logger, instance *arcadiav1alpha1.Model) (update bool, err error) {
	instanceDeepCopy := instance.DeepCopy()

	// Initialize Labels
	if instanceDeepCopy.Labels == nil {
		instanceDeepCopy.Labels = make(map[string]string)
	}
	// For model types
	isEmbeddingModel := strconv.FormatBool(instanceDeepCopy.IsEmbeddingModel())
	if v := instanceDeepCopy.Labels[arcadiav1alpha1.LabelModelEmbedding]; v != isEmbeddingModel {
		instanceDeepCopy.Labels[arcadiav1alpha1.LabelModelEmbedding] = isEmbeddingModel
		update = true
	}
	isLLMModel := strconv.FormatBool(instanceDeepCopy.IsLLMModel())
	if v := instanceDeepCopy.Labels[arcadiav1alpha1.LabelModelLLM]; v != isLLMModel {
		instanceDeepCopy.Labels[arcadiav1alpha1.LabelModelLLM] = isLLMModel
		update = true
	}

	// Initialize annotations
	if instanceDeepCopy.Annotations == nil {
		instanceDeepCopy.Annotations = make(map[string]string)
	}
	// For model's full storage path
	currentFullPath := instanceDeepCopy.FullPath()
	if v := instanceDeepCopy.Annotations[arcadiav1alpha1.LabelModelFullPath]; v != currentFullPath {
		instanceDeepCopy.Annotations[arcadiav1alpha1.LabelModelFullPath] = currentFullPath
		update = true
	}

	if update {
		return true, r.Client.Update(ctx, instanceDeepCopy)
	}

	return false, nil
}

// CheckModel to update status
func (r *ModelReconciler) CheckModel(ctx context.Context, logger logr.Logger, instance *arcadiav1alpha1.Model) error {
	logger.V(5).Info("check model")

	var (
		ds   datasource.Datasource
		info any
	)

	// If source is empty, it means that the data is still sourced from the internal minio and a state check is required,
	// otherwise we consider the model file for the trans-core service to be ready.
	if instance.Spec.Source == nil {
		logger.V(5).Info(fmt.Sprintf("model %s source is empty, check minio status.", instance.Name))
		system, err := config.GetSystemDatasource(ctx, r.Client, nil)
		if err != nil {
			return r.UpdateStatus(ctx, instance, err)
		}
		endpoint := system.Spec.Endpoint.DeepCopy()
		if endpoint != nil && endpoint.AuthSecret != nil {
			endpoint.AuthSecret.WithNameSpace(system.Namespace)
		}
		ds, err = datasource.NewLocal(ctx, r.Client, nil, endpoint)
		if err != nil {
			return r.UpdateStatus(ctx, instance, err)
		}
		// oss info:
		// - bucket: same as the instance namespace
		// - object: path joined with "model/{instance.name}"
		info = &arcadiav1alpha1.OSS{
			Bucket: instance.Namespace,
			Object: instance.ObjectPath(),
		}

		// check datasource against info
		if err := ds.Stat(ctx, info); err != nil {
			return r.UpdateStatus(ctx, instance, err)
		}
	}

	// update status
	return r.UpdateStatus(ctx, instance, nil)
}

// Remove model files from storage
func (r *ModelReconciler) RemoveModel(ctx context.Context, logger logr.Logger, instance *arcadiav1alpha1.Model) error {
	var ds datasource.Datasource
	var info any

	system, err := config.GetSystemDatasource(ctx, r.Client, nil)
	if err != nil {
		return r.UpdateStatus(ctx, instance, err)
	}
	endpoint := system.Spec.Endpoint.DeepCopy()
	if endpoint != nil && endpoint.AuthSecret != nil {
		endpoint.AuthSecret.WithNameSpace(system.Namespace)
	}
	ds, err = datasource.NewLocal(ctx, r.Client, nil, endpoint)
	if err != nil {
		return r.UpdateStatus(ctx, instance, err)
	}

	info = &arcadiav1alpha1.OSS{
		Bucket: instance.Namespace,
		Object: instance.ObjectPath(),
	}

	if err := ds.Stat(ctx, info); err != nil {
		return nil
	}

	return ds.Remove(ctx, info)
}

// UpdateStatus upon error
func (r *ModelReconciler) UpdateStatus(ctx context.Context, instance *arcadiav1alpha1.Model, err error) error {
	instanceCopy := instance.DeepCopy()
	var newCondition arcadiav1alpha1.Condition
	if err != nil {
		// set condition to False
		newCondition = instance.ErrorCondition(err.Error())
	} else {
		newCondition = instance.ReadyCondition()
	}
	instanceCopy.Status.SetConditions(newCondition)
	return r.Client.Status().Update(ctx, instanceCopy)
}
