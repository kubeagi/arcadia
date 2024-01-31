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
	"github.com/kubeagi/arcadia/pkg/datasource"
)

// DatasourceReconciler reconciles a Datasource object
type DatasourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=datasources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=datasources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=datasources/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Datasource object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *DatasourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(5).Info("Starting datasource reconcile")

	instance := &arcadiav1alpha1.Datasource{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		// There's no need to requeue if the resource no longer exists.
		// Otherwise, we'll be requeued implicitly because we return an error.
		logger.V(1).Info("Failed to get Datasource")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Add a finalizer.Then, we can define some operations which should
	// occur before the Datasource to be deleted.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers
	if newAdded := controllerutil.AddFinalizer(instance, arcadiav1alpha1.Finalizer); newAdded {
		logger.Info("Try to add Finalizer for Datasource")
		if err := r.Update(ctx, instance); err != nil {
			logger.Error(err, "Failed to update Datasource to add finalizer, will try again later")
			return ctrl.Result{}, err
		}
		logger.Info("Adding Finalizer for Datasource done")
		return ctrl.Result{Requeue: true}, nil
	}

	// Check if the Datasource instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if instance.GetDeletionTimestamp() != nil && controllerutil.ContainsFinalizer(instance, arcadiav1alpha1.Finalizer) {
		logger.Info("Performing Finalizer Operations for Datasource before delete CR")
		r.RemoveDatasource(logger, instance)
		logger.Info("Removing Finalizer for Datasource after successfully performing the operations")
		controllerutil.RemoveFinalizer(instance, arcadiav1alpha1.Finalizer)
		if err := r.Update(ctx, instance); err != nil {
			logger.Error(err, "Failed to remove finalizer for Datasource")
			return ctrl.Result{}, err
		}
		logger.Info("Remove Datasource done")
		return ctrl.Result{}, nil
	}

	// initialize labels
	requeue, err := r.Initialize(ctx, logger, instance)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to initialize datasource: %w", err)
	}
	if requeue {
		return reconcile.Result{Requeue: true}, nil
	}

	// check datasource
	if err := r.Checkdatasource(ctx, logger, instance); err != nil {
		// Update conditioned status
		return reconcile.Result{RequeueAfter: waitMedium}, err
	}
	return ctrl.Result{RequeueAfter: waitLonger}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatasourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arcadiav1alpha1.Datasource{}, builder.WithPredicates(predicate.Funcs{
			UpdateFunc: func(ue event.UpdateEvent) bool {
				oldDatsource := ue.ObjectOld.(*arcadiav1alpha1.Datasource)
				newDatasource := ue.ObjectNew.(*arcadiav1alpha1.Datasource)
				return !reflect.DeepEqual(oldDatsource.Spec, newDatasource.Spec) ||
					newDatasource.DeletionTimestamp != nil
			},
		})).
		Complete(r)
}

func (r *DatasourceReconciler) Initialize(ctx context.Context, logger logr.Logger, instance *arcadiav1alpha1.Datasource) (update bool, err error) {
	instanceDeepCopy := instance.DeepCopy()
	if instanceDeepCopy.Labels == nil {
		instanceDeepCopy.Labels = make(map[string]string)
	}

	currentType := string(instanceDeepCopy.Spec.Type())
	if v := instanceDeepCopy.Labels[arcadiav1alpha1.LabelDatasourceType]; v != currentType {
		instanceDeepCopy.Labels[arcadiav1alpha1.LabelDatasourceType] = currentType
		update = true
	}

	if update {
		return true, r.Client.Update(ctx, instanceDeepCopy)
	}

	return false, nil
}

// Checkdatasource to update status
func (r *DatasourceReconciler) Checkdatasource(ctx context.Context, logger logr.Logger, instance *arcadiav1alpha1.Datasource) error {
	logger.V(5).Info("check datasource")
	var err error

	endpoint := instance.Spec.Endpoint.DeepCopy()
	// set auth secret's namespace to the datasource's namespace
	if endpoint.AuthSecret != nil {
		endpoint.AuthSecret.WithNameSpace(instance.Namespace)
	}
	// create datasource
	var ds datasource.Datasource
	var info any
	switch instance.Spec.Type() {
	case arcadiav1alpha1.DatasourceTypeOSS:
		ds, err = datasource.NewOSS(ctx, r.Client, endpoint)
		if err != nil {
			return r.UpdateStatus(ctx, instance, err)
		}
		info = instance.Spec.OSS.DeepCopy()
	case arcadiav1alpha1.DatasourceTypeRDMA:
		return r.UpdateStatus(ctx, instance, nil)
	case arcadiav1alpha1.DatasourceTypePostgreSQL:
		ds, err = datasource.GetPostgreSQLPool(ctx, r.Client, instance)
		if err != nil {
			return r.UpdateStatus(ctx, instance, err)
		}
	case arcadiav1alpha1.DatasourceTypeWeb:
		info = instance.Spec.Web.DeepCopy()
		ds, err = datasource.NewWeb(ctx, endpoint.URL)
		if err != nil {
			return r.UpdateStatus(ctx, instance, err)
		}
	default:
		ds, err = datasource.NewUnknown(ctx, r.Client)
		if err != nil {
			return r.UpdateStatus(ctx, instance, err)
		}
	}

	// check datasource
	if err := ds.Stat(ctx, info); err != nil {
		return r.UpdateStatus(ctx, instance, err)
	}

	// update status
	return r.UpdateStatus(ctx, instance, nil)
}

// UpdateStatus upon error
func (r *DatasourceReconciler) UpdateStatus(ctx context.Context, instance *arcadiav1alpha1.Datasource, err error) error {
	instanceCopy := instance.DeepCopy()
	var newCondition arcadiav1alpha1.Condition
	if err != nil {
		// set condition to False
		newCondition = instance.ErrorCondition(err.Error())
	} else {
		// set condition to True
		newCondition = instance.ReadyCondition()
	}
	instanceCopy.Status.SetConditions(newCondition)
	return r.Client.Status().Update(ctx, instanceCopy)
}

func (r *DatasourceReconciler) RemoveDatasource(logger logr.Logger, instance *arcadiav1alpha1.Datasource) {
	logger.V(5).Info("remove datasource")
	switch instance.Spec.Type() {
	case arcadiav1alpha1.DatasourceTypeOSS:
	case arcadiav1alpha1.DatasourceTypeRDMA:
	case arcadiav1alpha1.DatasourceTypePostgreSQL:
		datasource.RemovePostgreSQLPool(*instance)
	default:
	}
}
