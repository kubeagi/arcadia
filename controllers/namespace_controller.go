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
	"github.com/minio/minio-go/v7"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubeagi/arcadia/pkg/config"
	"github.com/kubeagi/arcadia/pkg/datasource"
	"github.com/kubeagi/arcadia/pkg/streamlit"
	"github.com/kubeagi/arcadia/pkg/utils"
)

const (
	BucketNotEmpty = "The bucket you tried to delete is not empty"
	BucketNotExist = "The specified bucket does not exist"

	// this is the name of a configmap under the same namespace as operator. the key of the data field is the name of each namespace not to be handled.
	SkipNamespaceConfigMap = "skip-namespaces"
)

type NamespaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups="networking.k8s.io",resources=ingresses,verbs=get;list;watch;create;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Dataset object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(5).Info("Starting namespace reconcile")

	instance := &v1.Namespace{}
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			err = r.removeBucket(ctx, req.Name)
			return reconcile.Result{RequeueAfter: waitSmaller}, err
		}
		return reconcile.Result{}, err
	}

	// Check if the Namespace instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if instance.GetDeletionTimestamp() != nil {
		logger.Info("Namespace is marked to be deleted, skip it")
		return ctrl.Result{}, nil
	}

	// 1. Handle the streamlit install/uninstall, we'll run a streamlit pod for data app for each namespace
	var err error
	if instance.ObjectMeta.Annotations[streamlit.StreamlitInstalledAnnotation] == "true" {
		// check if the streamlit-server exists
		err = r.InstallStreamlitTool(ctx, logger, instance)
	} else {
		// check if the streamlit-server exists
		err = r.UninstallStreamlitTool(ctx, logger, instance)
	}

	if err != nil {
		return ctrl.Result{RequeueAfter: waitSmaller}, err
	}

	// 2. Reconcile for MinIO bucket, we will create a separate bucket for each namespace
	skip, err := r.checkSkippedNamespace(ctx, instance.Name)
	if err != nil {
		return reconcile.Result{RequeueAfter: waitMedium}, err
	}
	if skip {
		klog.Infof("namespace %s is in the filter list and will not be created, delete the corresponding bucket.", instance.Name)
		return reconcile.Result{}, nil
	}

	// TODO: check whether we need to synchronize for every event?
	err = r.syncBucket(ctx, instance.Name)
	if err != nil {
		return ctrl.Result{RequeueAfter: waitMedium}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Namespace{}).
		Complete(r)
}

func (r *NamespaceReconciler) ossClient(ctx context.Context) (*datasource.OSS, error) {
	systemDatasource, err := config.GetSystemDatasource(ctx, r.Client)
	if err != nil {
		klog.Errorf("get system datasource error %s", err)
		return nil, err
	}
	endpoint := systemDatasource.Spec.Enpoint.DeepCopy()
	if endpoint.AuthSecret != nil && endpoint.AuthSecret.Namespace == nil {
		endpoint.AuthSecret.WithNameSpace(systemDatasource.Namespace)
	}
	oss, err := datasource.NewOSS(ctx, r.Client, endpoint)
	if err != nil {
		klog.Errorf("generate new minio client error %s", err)
		return nil, err
	}
	return oss, nil
}

func (r *NamespaceReconciler) syncBucket(ctx context.Context, bucketName string) error {
	oss, err := r.ossClient(ctx)
	if err != nil {
		err = fmt.Errorf("sync bucket: failed to get oss client error %w", err)
		klog.Error(err)
		return err
	}
	// TODO: namespace might be quite short, but Minio bucket name cannot be shorter than 3 characters
	// maybe we can add suffix later to fix this
	exists, err := oss.Client.BucketExists(ctx, bucketName)
	if err != nil {
		err = fmt.Errorf("check if the bucket exists and an error occurs, error: %w", err)
		klog.Error(err)
		return err
	}
	if !exists {
		klog.Infof("bucket %s does not exist, ready to create bucket", bucketName)
		if err = oss.Client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
			err = fmt.Errorf("and error osccured creating the bucket, error %w", err)
			klog.Error(err)
			return err
		}
	}
	return nil
}

func (r *NamespaceReconciler) removeBucket(ctx context.Context, bucketName string) error {
	oss, err := r.ossClient(ctx)
	if err != nil {
		err = fmt.Errorf("remove bucket: failed to get oss client error %w", err)
		klog.Error(err)
		return err
	}
	err = oss.Client.RemoveBucket(ctx, bucketName)
	if err == nil || err.Error() == BucketNotExist {
		return nil
	}
	return err
}

func (r *NamespaceReconciler) checkSkippedNamespace(ctx context.Context, namespace string) (bool, error) {
	cm := v1.ConfigMap{}
	controllerNamespace := utils.GetCurrentNamespace()
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: controllerNamespace, Name: SkipNamespaceConfigMap}, &cm); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	_, ok := cm.Data[namespace]
	return ok, nil
}

// InstallStreamlitTool to install the required resource of streamlit
func (r *NamespaceReconciler) InstallStreamlitTool(ctx context.Context, logger logr.Logger, instance *v1.Namespace) error {
	logger.Info("Installing streamlit tool...")
	stDeployer := streamlit.NewStreamlitDeployer(ctx, r.Client, instance)

	return stDeployer.Install()
}

// UninstallStreamlitTool to remove resources of streamlit
func (r *NamespaceReconciler) UninstallStreamlitTool(ctx context.Context, logger logr.Logger, instance *v1.Namespace) error {
	stDeployer := streamlit.NewStreamlitDeployer(ctx, r.Client, instance)

	return stDeployer.Uninstall()
}
