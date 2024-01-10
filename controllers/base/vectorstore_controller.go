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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/vectorstore"
)

// VectorStoreReconciler reconciles a VectorStore object
type VectorStoreReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=vectorstores,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=vectorstores/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=vectorstores/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VectorStore object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *VectorStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(5).Info("Start VectorStore Reconcile")

	vs := &arcadiav1alpha1.VectorStore{}
	if err := r.Get(ctx, req.NamespacedName, vs); err != nil {
		// There's no need to requeue if the resource no longer exists.
		// Otherwise, we'll be requeued implicitly because we return an error.
		log.V(1).Info("Failed to get VectorStore")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.V(5).Info("Get VectorStore instance")

	// Add a finalizer.Then, we can define some operations which should
	// occur before the KnowledgeBase to be deleted.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers
	if newAdded := controllerutil.AddFinalizer(vs, arcadiav1alpha1.Finalizer); newAdded {
		log.Info("Try to add Finalizer for VectorStore")
		if err := r.Update(ctx, vs); err != nil {
			log.Error(err, "Failed to update VectorStore to add finalizer, will try again later")
			return ctrl.Result{}, err
		}
		log.Info("Adding Finalizer for VectorStore done")
		return ctrl.Result{}, nil
	}

	// Check if the VectorStore instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if vs.GetDeletionTimestamp() != nil && controllerutil.ContainsFinalizer(vs, arcadiav1alpha1.Finalizer) {
		log.Info("Performing Finalizer Operations for VectorStore before delete CR")
		// TODO perform the finalizer operations here, for example: remove vectorstore data?
		log.Info("Removing Finalizer for VectorStore after successfully performing the operations")
		controllerutil.RemoveFinalizer(vs, arcadiav1alpha1.Finalizer)
		if err := r.Update(ctx, vs); err != nil {
			log.Error(err, "Failed to remove finalizer for VectorStore")
			return ctrl.Result{}, err
		}
		log.Info("Remove VectorStore done")
		return ctrl.Result{}, nil
	}

	if vs.Labels == nil {
		vs.Labels = make(map[string]string)
	}

	currentType := string(vs.Spec.Type())
	if v := vs.Labels[arcadiav1alpha1.LabelVectorStoreType]; v != currentType {
		vs.Labels[arcadiav1alpha1.LabelVectorStoreType] = currentType
		return reconcile.Result{}, r.Update(ctx, vs)
	}

	if err := r.CheckVectorStore(ctx, log, vs); err != nil {
		return reconcile.Result{RequeueAfter: waitMedium}, nil
	}

	return ctrl.Result{RequeueAfter: waitLonger}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VectorStoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arcadiav1alpha1.VectorStore{},
			builder.WithPredicates(
				predicate.Or(
					FinalizersChangedPredicate{},
					predicate.GenerationChangedPredicate{},
					predicate.LabelChangedPredicate{}))).
		Complete(r)
}

func (r *VectorStoreReconciler) CheckVectorStore(ctx context.Context, log logr.Logger, vs *arcadiav1alpha1.VectorStore) (err error) {
	log.V(5).Info("check vectorstore")
	vsRaw := vs.DeepCopy()
	_, finish, err := vectorstore.NewVectorStore(ctx, vs, nil, "", r.Client, nil)
	if err != nil {
		log.Error(err, "failed to connect to vectorstore")
		r.setCondition(vs, vs.ErrorCondition(err.Error()))
	} else {
		r.setCondition(vs, vs.ReadyCondition())
		if finish != nil {
			finish()
		}
	}
	if err := r.patchStatus(ctx, vs); err != nil {
		return err
	}
	if !reflect.DeepEqual(vsRaw, vs) {
		if err := r.Patch(ctx, vs, client.MergeFrom(vsRaw)); err != nil {
			return err
		}
	}
	return err
}

func (r *VectorStoreReconciler) setCondition(vs *arcadiav1alpha1.VectorStore, condition ...arcadiav1alpha1.Condition) *arcadiav1alpha1.VectorStore {
	vs.Status.SetConditions(condition...)
	return vs
}

func (r *VectorStoreReconciler) patchStatus(ctx context.Context, vs *arcadiav1alpha1.VectorStore) error {
	latest := &arcadiav1alpha1.VectorStore{}
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(vs), latest); err != nil {
		return err
	}
	// No need to patch if status is the same
	if reflect.DeepEqual(vs.Status, latest.Status) {
		return nil
	}
	patch := client.MergeFrom(latest.DeepCopy())
	latest.Status = vs.Status
	return r.Client.Status().Patch(ctx, latest, patch, client.FieldOwner("vectorstore-controller"))
}
