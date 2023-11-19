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
	"sync"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubeagi/arcadia/api/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/scheduler"
	"github.com/kubeagi/arcadia/pkg/utils"
)

// VersionedDatasetReconciler reconciles a VersionedDataset object
type VersionedDatasetReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	cache sync.Map
}

//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=versioneddatasets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=versioneddatasets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=versioneddatasets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VersionedDataset object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *VersionedDatasetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var err error

	instance := &v1alpha1.VersionedDataset{}
	if err = r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		klog.Errorf("reconcile: failed to get versionDataset with req: %v", req.NamespacedName)
		return reconcile.Result{}, err
	}
	updatedObj, err := r.preUpdate(ctx, instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	if updatedObj {
		return reconcile.Result{Requeue: true}, r.Client.Update(ctx, instance)
	}

	key := fmt.Sprintf("%s/%s", instance.Namespace, instance.Name)
	v, ok := r.cache.Load(key)
	if ok {
		v.(*scheduler.Scheduler).Stop()
	}
	s, err := scheduler.NewScheduler(ctx, r.Client, instance)
	if err != nil {
		return reconcile.Result{}, err
	}
	r.cache.Store(key, s)

	klog.V(4).Infof("[Debug] start to sync files for %s/%s", instance.Namespace, instance.Name)
	go func() {
		_ = s.Start()
	}()

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VersionedDatasetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.VersionedDataset{}).
		Complete(r)
}

func (r *VersionedDatasetReconciler) preUpdate(ctx context.Context, instance *v1alpha1.VersionedDataset) (bool, error) {
	var err error
	update := false
	if instance.Labels == nil {
		instance.Labels = make(map[string]string)
	}
	if v, ok := instance.Labels[v1alpha1.LabelVersionedDatasetVersion]; !ok || v != instance.Spec.Version {
		instance.Labels[v1alpha1.LabelVersionedDatasetVersion] = instance.Spec.Version
		update = true
	}
	if v, ok := instance.Labels[v1alpha1.LabelVersionedDatasetVersionOwner]; !ok || v != instance.Spec.Dataset.Name {
		instance.Labels[v1alpha1.LabelVersionedDatasetVersionOwner] = instance.Spec.Dataset.Name
		update = true
	}

	if !utils.ContainString(instance.Finalizers, v1alpha1.Finalizer) {
		update = true
		instance.Finalizers = utils.AddString(instance.Finalizers, v1alpha1.Finalizer)
	}

	dataset := &v1alpha1.Dataset{}
	namespace := instance.Namespace
	if instance.Spec.Dataset.Namespace != nil {
		namespace = *instance.Spec.Dataset.Namespace
	}
	if err = r.Client.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      instance.Spec.Dataset.Name}, dataset); err != nil {
		klog.Errorf("preUpdate: failed to get dataset %s/%s, error %s", namespace, instance.Spec.Dataset.Name, err)
		return false, err
	}

	index := 0
	for index = range instance.OwnerReferences {
		if instance.OwnerReferences[index].UID == dataset.UID {
			break
		}
	}
	if index == len(instance.OwnerReferences) {
		if err = controllerutil.SetControllerReference(dataset, instance, r.Scheme); err != nil {
			klog.Errorf("preUpdate: failed to set versionDataset %s/%s's ownerReference", instance.Namespace, instance.Name)
			return false, err
		}

		update = true
	}

	return update, err
}
