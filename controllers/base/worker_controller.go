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
	"strings"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/config"
	arcadiaworker "github.com/kubeagi/arcadia/pkg/worker"
)

// WorkerReconciler reconciles a Worker object
type WorkerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=workers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=workers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=workers/finalizers,verbs=update
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=datasources,verbs=get;list;watch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=datasources/status,verbs=get
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=embedders;llms,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=embedders/status;llms/status,verbs=get;update;patch

//+kubebuilder:rbac:groups="",resources=configmaps;secrets,verbs=get;list
//+kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=deployments/status,verbs=get;watch
//+kubebuilder:rbac:groups="",resources=services;pods;persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods/status;services/status;persistentvolumeclaims/status,verbs=get;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Worker object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *WorkerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(5).Info("Start Worker Reconcile")
	worker := &arcadiav1alpha1.Worker{}
	if err := r.Get(ctx, req.NamespacedName, worker); err != nil {
		// There's no need to requeue if the resource no longer exists.
		// Otherwise, we'll be requeued implicitly because we return an error.
		log.V(1).Info("Failed to get Worker")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Add a finalizer.Then, we can define some operations which should
	// occur before the Worker to be deleted.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers
	if newAdded := controllerutil.AddFinalizer(worker, arcadiav1alpha1.Finalizer); newAdded {
		log.V(5).Info("Try to add Finalizer for Worker")
		if err = r.Update(ctx, worker); err != nil {
			log.V(1).Info("Failed to update Worker to add finalizer")
			return ctrl.Result{}, err
		}
		log.V(5).Info("Adding Finalizer for Worker done")
		return ctrl.Result{Requeue: true}, nil
	}

	// Check if the Worker instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if worker.GetDeletionTimestamp() != nil && controllerutil.ContainsFinalizer(worker, arcadiav1alpha1.Finalizer) {
		log.V(5).Info("Performing Finalizer Operations for Worker before delete CR")
		// TODO perform the finalizer operations here, for example: remove vectorstore data?
		log.V(5).Info("Removing Finalizer for Worker after successfully performing the operations")
		controllerutil.RemoveFinalizer(worker, arcadiav1alpha1.Finalizer)
		if err = r.Update(ctx, worker); err != nil {
			log.V(1).Info("Failed to remove finalizer for Worker")
			return ctrl.Result{}, err
		}
		log.V(1).Info("Remove Worker done")
		return ctrl.Result{}, nil
	}

	// initialize labels
	requeue, err := r.initialize(ctx, log, worker)
	if err != nil {
		log.V(1).Info("Failed to update labels")
		return ctrl.Result{}, err
	}
	if requeue {
		return ctrl.Result{Requeue: true}, nil
	}

	// core rereconcile for worker
	reconciledWorker, err := r.reconcile(ctx, log, worker)
	if err != nil {
		log.Error(err, "Failed to reconcile worker")
		r.setCondition(worker, worker.ErrorCondition(err.Error()))
	}

	// update status
	updateStatusErr := r.patchStatus(ctx, reconciledWorker)
	if updateStatusErr != nil {
		log.Error(updateStatusErr, "Failed to patch worker status")
		return ctrl.Result{Requeue: true}, updateStatusErr
	}

	return ctrl.Result{}, nil
}

func (r *WorkerReconciler) initialize(ctx context.Context, _ logr.Logger, instance *arcadiav1alpha1.Worker) (bool, error) {
	instanceDeepCopy := instance.DeepCopy()

	var update bool

	// Initialize Labels
	if instanceDeepCopy.Labels == nil {
		instanceDeepCopy.Labels = make(map[string]string)
	}

	// For worker type
	currentType := string(instanceDeepCopy.Type())
	if v := instanceDeepCopy.Labels[arcadiav1alpha1.LabelWorkerType]; v != currentType {
		instanceDeepCopy.Labels[arcadiav1alpha1.LabelWorkerType] = currentType
		update = true
	}

	if instance.Spec.Model != nil {
		ns := instance.Namespace
		if instance.Spec.Model.Namespace != nil {
			ns = *instance.Spec.Model.Namespace
		}
		m := arcadiav1alpha1.Model{}
		if err := r.Client.Get(ctx, types.NamespacedName{Namespace: ns, Name: instance.Spec.Model.Name}, &m); err != nil {
			return true, err
		}
		if types, ok := instanceDeepCopy.Labels[arcadiav1alpha1.WorkerModelTypesLabel]; !ok || strings.ReplaceAll(types, "_", ",") != m.Spec.Types {
			// label do not accept `,`,so replace it with `_`
			instanceDeepCopy.Labels[arcadiav1alpha1.WorkerModelTypesLabel] = strings.ReplaceAll(m.Spec.Types, ",", "_")
			update = true
		}
	} else {
		if _, ok := instanceDeepCopy.Labels[arcadiav1alpha1.WorkerModelTypesLabel]; ok {
			delete(instanceDeepCopy.Labels, arcadiav1alpha1.WorkerModelTypesLabel)
			update = true
		}
	}

	if update {
		return true, r.Client.Update(ctx, instanceDeepCopy)
	}

	return false, nil
}

func (r *WorkerReconciler) reconcile(ctx context.Context, logger logr.Logger, worker *arcadiav1alpha1.Worker) (*arcadiav1alpha1.Worker, error) {
	logger.V(5).Info("GetSystemDatasource which hosts the worker's model files")

	m := arcadiav1alpha1.Model{}
	ns := worker.Namespace
	if worker.Spec.Model.Namespace != nil && *worker.Spec.Model.Namespace != "" {
		ns = *worker.Spec.Model.Namespace
	}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: worker.Spec.Model.Name, Namespace: ns}, &m); err != nil {
		return worker, errors.Wrap(err, "failed to get model")
	}

	var (
		datasource = &arcadiav1alpha1.Datasource{}
		err        error
	)
	if m.Spec.Source != nil {
		if err = r.Client.Get(ctx, types.NamespacedName{Namespace: ns, Name: m.Spec.Source.Name}, datasource); err != nil {
			return worker, errors.Wrap(err, "model config datasource, but get it failed.")
		}
	} else {
		datasource, err = config.GetSystemDatasource(ctx)
		if err != nil {
			return worker, errors.Wrap(err, "Failed to get system datasource")
		}
	}

	// Only PodWorker(hosts this worker via a single pod) supported now
	w, err := arcadiaworker.NewPodWorker(ctx, r.Client, r.Scheme, worker, datasource)
	if err != nil {
		return worker, errors.Wrap(err, "Failed to new a pod worker")
	}

	logger.V(5).Info("BeforeStart worker")
	if err := w.BeforeStart(ctx); err != nil {
		return w.Worker(), errors.Wrap(err, "Failed to do BeforeStart")
	}

	logger.V(5).Info("Start worker")
	if err := w.Start(ctx); err != nil {
		return w.Worker(), errors.Wrap(err, "Failed to do Start")
	}

	logger.V(5).Info("AfterStart worker")
	err = w.AfterStart(ctx)
	if err != nil {
		return w.Worker(), errors.Wrap(err, "Failed to do AfterStart")
	}

	return w.Worker(), nil
}

func (r *WorkerReconciler) setCondition(worker *arcadiav1alpha1.Worker, condition ...arcadiav1alpha1.Condition) *arcadiav1alpha1.Worker {
	worker.Status.SetConditions(condition...)
	return worker
}

func (r *WorkerReconciler) patchStatus(ctx context.Context, worker *arcadiav1alpha1.Worker) error {
	latest := &arcadiav1alpha1.Worker{}
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(worker), latest); err != nil {
		return err
	}
	if reflect.DeepEqual(worker.Status, latest.Status) {
		return nil
	}

	patch := client.MergeFrom(latest.DeepCopy())
	latest.Status = worker.Status
	return r.Client.Status().Patch(ctx, latest, patch, client.FieldOwner("worker-controller"))
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arcadiav1alpha1.Worker{}, builder.WithPredicates(predicate.Funcs{
			UpdateFunc: func(ue event.UpdateEvent) bool {
				oldWorker := ue.ObjectOld.(*arcadiav1alpha1.Worker)
				newWorker := ue.ObjectNew.(*arcadiav1alpha1.Worker)

				return !reflect.DeepEqual(oldWorker.Spec, newWorker.Spec) || newWorker.DeletionTimestamp != nil
			},
		})).
		Watches(&source.Kind{Type: &corev1.Pod{}}, handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
			pod := o.(*corev1.Pod)
			if pod.Labels != nil && pod.Labels[arcadiav1alpha1.WorkerPodLabel] != "" {
				return []ctrl.Request{
					reconcile.Request{
						NamespacedName: types.NamespacedName{
							Namespace: pod.Namespace,
							Name:      pod.Labels[arcadiav1alpha1.WorkerPodLabel],
						},
					},
				}
			}
			return []ctrl.Request{}
		})).
		Complete(r)
}
