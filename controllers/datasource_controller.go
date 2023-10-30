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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/datasource"
	"github.com/kubeagi/arcadia/pkg/utils"
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
	logger.Info("Starting datasource reconcile")

	instance := &arcadiav1alpha1.Datasource{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			// datasourcce has been deleted.
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if instance.DeletionTimestamp != nil {
		logger.Info("Delete datasource")
		// remove the finalizer to complete the delete action
		instance.Finalizers = utils.RemoveString(instance.Finalizers, arcadiav1alpha1.Finalizer)
		err := r.Client.Update(ctx, instance)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to update datasource finializer: %w", err)
		}
		return reconcile.Result{}, nil
	}

	// initialize labels
	requeue, err := r.Initialize(ctx, logger, instance)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to initiali datasource: %w", err)
	}
	if requeue {
		return reconcile.Result{Requeue: true}, nil
	}

	// check datasource
	if err := r.Checkdatasource(ctx, logger, instance); err != nil {
		// Update conditioned status
		return reconcile.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatasourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arcadiav1alpha1.Datasource{}).
		Complete(r)
}

func (r *DatasourceReconciler) Initialize(ctx context.Context, logger logr.Logger, instance *arcadiav1alpha1.Datasource) (bool, error) {
	instanceDeepCopy := instance.DeepCopy()
	l := len(instanceDeepCopy.Finalizers)

	var update bool

	instanceDeepCopy.Finalizers = utils.AddString(instanceDeepCopy.Finalizers, arcadiav1alpha1.Finalizer)
	if l != len(instanceDeepCopy.Finalizers) {
		logger.V(1).Info("Add Finalizer for datasource", "Finalizer", arcadiav1alpha1.Finalizer)
		update = true
	}

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
	logger.Info("check datasource")
	var err error

	// create datasource
	var ds datasource.Datasource
	var info any
	switch instance.Spec.Type() {
	case arcadiav1alpha1.DatasourceTypeLocal:
		// FIXME: implement local datasource check when system datasource defined by https://github.com/kubeagi/arcadia/issues/156
		// 1. read system datasource endpoint
		// 2. check against pre-denfined rules for local datasource rules
		return r.UpdateStatus(ctx, instance, nil)
	case arcadiav1alpha1.DatasourceTypeOSS:
		endpoiont := instance.Spec.Enpoint.DeepCopy()
		// set auth secret's namespace to the datasource's namespace
		if endpoiont.AuthSecret != nil {
			endpoiont.AuthSecret.WithNameSpace(instance.Namespace)
		}
		ds, err = datasource.NewOSS(ctx, r.Client, endpoiont)
		if err != nil {
			return r.UpdateStatus(ctx, instance, err)
		}
		info = instance.Spec.OSS.DeepCopy()
	default:
		ds, err = datasource.NewUnknown(ctx, r.Client)
		if err != nil {
			return r.UpdateStatus(ctx, instance, err)
		}
	}

	// check datasource
	if err := ds.Check(ctx, info); err != nil {
		return r.UpdateStatus(ctx, instance, err)
	}

	// update status
	return r.UpdateStatus(ctx, instance, nil)
}

func (r *DatasourceReconciler) UpdateStatus(ctx context.Context, instance *arcadiav1alpha1.Datasource, err error) error {
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
		// Set status to available
		instanceCopy.Status.SetConditions(arcadiav1alpha1.Condition{
			Type:               arcadiav1alpha1.TypeReady,
			Status:             corev1.ConditionTrue,
			Reason:             arcadiav1alpha1.ReasonAvailable,
			Message:            "health check success",
			LastTransitionTime: metav1.Now(),
			LastSuccessfulTime: metav1.Now(),
		})
	}
	return r.Client.Status().Update(ctx, instanceCopy)
}
