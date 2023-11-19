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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/utils"
)

// DatasetReconciler reconciles a Dataset object
type DatasetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=datasets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=versioneddatasets,verbs=deletecollection
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=datasets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=datasets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Dataset object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *DatasetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var err error
	instance := &arcadiav1alpha1.Dataset{}
	if err = r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	if instance.DeletionTimestamp != nil {
		if err := r.Client.DeleteAllOf(ctx, &arcadiav1alpha1.VersionedDataset{}, client.InNamespace(instance.Namespace), client.MatchingLabels{
			arcadiav1alpha1.LabelVersionedDatasetVersionOwner: instance.Name,
		}); err != nil {
			return reconcile.Result{}, err
		}
		instance.Finalizers = utils.RemoveString(instance.Finalizers, arcadiav1alpha1.Finalizer)
		err = r.Client.Update(ctx, instance)
		return reconcile.Result{}, err
	}

	if instance.Labels == nil {
		instance.Labels = make(map[string]string)
	}

	update := false
	if v, ok := instance.Labels[arcadiav1alpha1.LabelDatasetContentType]; !ok || v != instance.Spec.ContentType {
		instance.Labels[arcadiav1alpha1.LabelDatasetContentType] = instance.Spec.ContentType
		update = true
	}
	if v, ok := instance.Labels[arcadiav1alpha1.LabelDatasetField]; !ok || v != instance.Spec.Field {
		instance.Labels[arcadiav1alpha1.LabelDatasetField] = instance.Spec.Field
		update = true
	}
	if !utils.ContainString(instance.Finalizers, arcadiav1alpha1.Finalizer) {
		instance.Finalizers = utils.AddString(instance.Finalizers, arcadiav1alpha1.Finalizer)
		update = true
	}
	if update {
		err = r.Client.Update(ctx, instance)
		return reconcile.Result{Requeue: true}, err
	}

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatasetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arcadiav1alpha1.Dataset{}).
		Complete(r)
}
