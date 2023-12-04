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

	"github.com/go-logr/logr"
	"github.com/minio/minio-go/v7"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/config"
	"github.com/kubeagi/arcadia/pkg/datasource"
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
	logger := log.FromContext(ctx)

	logger.V(5).Info("Start VersionedDataset Reconcile")

	var err error

	instance := &v1alpha1.VersionedDataset{}
	if err = r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		logger.V(1).Info("Failed to get VersionedDataset")
		return reconcile.Result{}, err
	}
	if instance.DeletionTimestamp != nil {
		logger.Info("Remove bucket files for versioneddatset")
		if err = r.removeBucketFiles(ctx, logger, instance); err != nil {
			return reconcile.Result{}, err
		}
		instance.Finalizers = nil
		logger.Info("Remove versioneddatset done")
		return reconcile.Result{}, r.Client.Update(ctx, instance)
	}

	if instance.DeletionTimestamp == nil {
		updatedObj, err := r.preUpdate(ctx, logger, instance)
		if err != nil {
			// Skip if it's NotFound error
			if errors.IsNotFound(err) {
				logger.V(1).Info(" Failed to get VersionedDataset")
				return reconcile.Result{}, nil
			}
			return reconcile.Result{}, err
		}

		if updatedObj {
			return reconcile.Result{}, r.Client.Update(ctx, instance)
		}
	}

	deepCopy := instance.DeepCopy()
	update, deleteFilestatus, err := r.checkStatus(ctx, logger, deepCopy)
	if err != nil {
		return reconcile.Result{}, err
	}

	if update || len(deleteFilestatus) > 0 {
		if len(deleteFilestatus) > 0 {
			logger.V(1).Info("Need to delete files", "Files", deleteFilestatus)
			s, err := scheduler.NewScheduler(ctx, r.Client, instance, deleteFilestatus, true)
			if err != nil {
				return reconcile.Result{}, err
			}
			logger.V(1).Info("Start to delete group files", "Number", len(deleteFilestatus))
			if err = s.Start(); err != nil {
				logger.Error(err, "failed to delete files, need retry")
				return reconcile.Result{RequeueAfter: waitMedium}, nil
			}
		}
		if instance.DeletionTimestamp == nil {
			err := r.Client.Status().Patch(ctx, deepCopy, client.MergeFrom(instance))
			if err != nil {
				logger.Error(err, "Failed to patch status")
			}
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	logger.V(1).Info("Start to add new files")

	key := fmt.Sprintf("%s/%s", instance.Namespace, instance.Name)
	v, ok := r.cache.Load(key)
	if ok {
		v.(*scheduler.Scheduler).Stop()
	}
	s, err := scheduler.NewScheduler(ctx, r.Client, instance, nil, false)
	if err != nil {
		logger.Error(err, "Faled to generate scheduler")
		return reconcile.Result{}, err
	}
	r.cache.Store(key, s)

	logger.V(1).Info("Start to sync files")
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

func (r *VersionedDatasetReconciler) preUpdate(ctx context.Context, logger logr.Logger, instance *v1alpha1.VersionedDataset) (bool, error) {
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
		logger.Error(err, "Failed to preUpdate the dataset", "Dataset Namespace", namespace, "Dataset Name", instance.Spec.Dataset.Name)
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
			logger.Error(err, "Failed to preUpdate the versionDataset ownerReference")
			return false, err
		}

		update = true
	}

	return update, err
}

func (r *VersionedDatasetReconciler) checkStatus(ctx context.Context, logger logr.Logger, instance *v1alpha1.VersionedDataset) (bool, []v1alpha1.FileStatus, error) {
	// TODO: Currently, we think there is only one default minio environment,
	// so we get the minio client directly through the configuration.
	systemDatasource, err := config.GetSystemDatasource(ctx, r.Client)
	if err != nil {
		logger.Error(err, "Failed to get system datasource")
		return false, nil, err
	}
	endpoint := systemDatasource.Spec.Enpoint.DeepCopy()
	if endpoint.AuthSecret != nil && endpoint.AuthSecret.Namespace == nil {
		endpoint.AuthSecret.WithNameSpace(systemDatasource.Namespace)
	}
	oss, err := datasource.NewOSS(ctx, r.Client, endpoint)
	if err != nil {
		logger.Error(err, "Failed to generate new minio client")
		return false, nil, err
	}

	update, deleteFileStatus := v1alpha1.CopyedFileGroup2Status(oss.Client, instance)
	return update, deleteFileStatus, nil
}

func (r *VersionedDatasetReconciler) removeBucketFiles(ctx context.Context, logger logr.Logger, instance *v1alpha1.VersionedDataset) error {
	systemDatasource, err := config.GetSystemDatasource(ctx, r.Client)
	if err != nil {
		logger.Error(err, "Failed to get system datasource")
		return err
	}
	endpoint := systemDatasource.Spec.Enpoint.DeepCopy()
	if endpoint.AuthSecret != nil && endpoint.AuthSecret.Namespace == nil {
		endpoint.AuthSecret.WithNameSpace(systemDatasource.Namespace)
	}
	oss, err := datasource.NewOSS(ctx, r.Client, endpoint)
	if err != nil {
		logger.Error(err, "Failed to generate new minio client")
		return err
	}

	for ei := range oss.Client.RemoveObjects(ctx, instance.Namespace, oss.Client.ListObjects(ctx, instance.Namespace, minio.ListObjectsOptions{
		Prefix:    fmt.Sprintf("dataset/%s/%s/", instance.Spec.Dataset.Name, instance.Spec.Version),
		Recursive: true,
	}), minio.RemoveObjectsOptions{}) {
		err = ei.Err
		logger.Error(err, "failed to remove object", "Object", ei.ObjectName)
	}
	return err
}
