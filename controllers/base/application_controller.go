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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	agentv1alpha1 "github.com/kubeagi/arcadia/api/app-node/agent/v1alpha1"
	chainv1alpha1 "github.com/kubeagi/arcadia/api/app-node/chain/v1alpha1"
	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
)

// ApplicationReconciler reconciles an Application object
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=applications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=applications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=applications/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(5).Info("Start Application Reconcile")
	app := &arcadiav1alpha1.Application{}
	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		// There's no need to requeue if the resource no longer exists.
		// Otherwise, we'll be requeued implicitly because we return an error.
		log.V(1).Info("Failed to get Application")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log = log.WithValues("Generation", app.GetGeneration(), "ObservedGeneration", app.Status.ObservedGeneration, "creator", app.Spec.Creator)
	log.V(5).Info("Get Application instance")

	// Add a finalizer.Then, we can define some operations which should
	// occur before the Application to be deleted.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers
	if newAdded := controllerutil.AddFinalizer(app, arcadiav1alpha1.Finalizer); newAdded {
		log.Info("Try to add Finalizer for Application")
		if err := r.Update(ctx, app); err != nil {
			log.Error(err, "Failed to update Application to add finalizer, will try again later")
			return ctrl.Result{}, err
		}
		log.Info("Adding Finalizer for Application done")
		return ctrl.Result{}, nil
	}

	// Check if the Application instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if app.GetDeletionTimestamp() != nil && controllerutil.ContainsFinalizer(app, arcadiav1alpha1.Finalizer) {
		log.Info("Performing Finalizer Operations for Application before delete CR")
		// TODO perform the finalizer operations here, for example: remove vectorstore data?
		log.Info("Removing Finalizer for Application after successfully performing the operations")
		controllerutil.RemoveFinalizer(app, arcadiav1alpha1.Finalizer)
		if err := r.Update(ctx, app); err != nil {
			log.Error(err, "Failed to remove finalizer for Application")
			return ctrl.Result{}, err
		}
		log.Info("Remove Application done")
		return ctrl.Result{}, nil
	}

	app, result, err := r.reconcile(ctx, log, app)

	// Update status after reconciliation.
	if updateStatusErr := r.patchStatus(ctx, app); updateStatusErr != nil {
		log.Error(updateStatusErr, "unable to update status after reconciliation")
		return ctrl.Result{Requeue: true}, updateStatusErr
	}

	return result, err
}

// validate nodes:
// todo remove to webhook
// 1. input node must have next node
// 2. output node must not have next node
// 3. input node must only have one
// 4. input node must only have one
// 5. only one node connected to output, and this node type should be chain or agent
// 6. when this node points to output, it can only point to output
// 7. should not have cycle TODO
// 8. nodeName should be unique
func (r *ApplicationReconciler) validateNodes(ctx context.Context, log logr.Logger, app *arcadiav1alpha1.Application) (*arcadiav1alpha1.Application, ctrl.Result, error) {
	var input, output int
	var outputNodeName string
	nodeName := make(map[string]bool, len(app.Spec.Nodes))
	for _, node := range app.Spec.Nodes {
		if _, ok := nodeName[node.Name]; ok {
			r.setCondition(app, app.Status.ErrorCondition("node name should be unique")...)
			return app, ctrl.Result{}, nil
		}
		nodeName[node.Name] = true
		if node.Ref.Kind == arcadiav1alpha1.InputNode {
			input++
			if len(node.NextNodeName) == 0 {
				r.setCondition(app, app.Status.ErrorCondition("input node needs one or more next nodes")...)
				return app, ctrl.Result{}, nil
			}
		}
		if node.Ref.Kind == arcadiav1alpha1.OutputNode {
			output++
			outputNodeName = node.Name
			if len(node.NextNodeName) != 0 {
				r.setCondition(app, app.Status.ErrorCondition("output node should not have next nodes")...)
				return app, ctrl.Result{}, nil
			}
		}
	}
	if input != 1 {
		r.setCondition(app, app.Status.ErrorCondition("need one input node")...)
		return app, ctrl.Result{}, nil
	}
	if output != 1 {
		r.setCondition(app, app.Status.ErrorCondition("need one output node")...)
		return app, ctrl.Result{}, nil
	}

	var toOutput int
	var toOutputNodeNext int
	for _, node := range app.Spec.Nodes {
		for _, n := range node.NextNodeName {
			if n == outputNodeName {
				toOutput++
				group := node.Ref.APIGroup
				if group == nil {
					r.setCondition(app, app.Status.ErrorCondition("node should have ref.group setting")...)
					return app, ctrl.Result{}, nil
				}
				// Only allow chain group or agent node as the ending node
				if *group != chainv1alpha1.Group && (*group != agentv1alpha1.Group && node.Ref.Kind != "agent") {
					r.setCondition(app, app.Status.ErrorCondition("ending node should be chain or agent")...)
					return app, ctrl.Result{}, nil
				}
			}
			toOutputNodeNext = len(node.NextNodeName)
		}
	}
	if toOutput != 1 {
		r.setCondition(app, app.Status.ErrorCondition("only one node can output")...)
		return app, ctrl.Result{}, nil
	}
	if toOutputNodeNext != 1 {
		r.setCondition(app, app.Status.ErrorCondition("when this node points to output, it can only point to output")...)
		return app, ctrl.Result{}, nil
	}

	r.setCondition(app, app.Status.ReadyCondition()...)
	return app, ctrl.Result{}, nil
}

func (r *ApplicationReconciler) reconcile(ctx context.Context, log logr.Logger, app *arcadiav1alpha1.Application) (*arcadiav1alpha1.Application, ctrl.Result, error) {
	// Observe generation change
	if app.Status.ObservedGeneration != app.Generation {
		app.Status.ObservedGeneration = app.Generation
		r.setCondition(app, app.Status.WaitingCompleteCondition()...)
		if updateStatusErr := r.patchStatus(ctx, app); updateStatusErr != nil {
			log.Error(updateStatusErr, "unable to update status after generation update")
			return app, ctrl.Result{Requeue: true}, updateStatusErr
		}
	}
	appRaw := app.DeepCopy()
	if app.Spec.IsPublic {
		if app.Labels == nil {
			app.Labels = make(map[string]string, 1)
		}
		app.Labels[arcadiav1alpha1.AppPublicLabelKey] = ""
	} else {
		delete(app.Labels, arcadiav1alpha1.AppPublicLabelKey)
	}
	if !reflect.DeepEqual(app, appRaw) {
		return app, ctrl.Result{Requeue: true}, r.Patch(ctx, app, client.MergeFrom(appRaw))
	}
	if app.Status.IsReady() {
		return app, ctrl.Result{}, nil
	}
	return r.validateNodes(ctx, log, app)
}

func (r *ApplicationReconciler) patchStatus(ctx context.Context, app *arcadiav1alpha1.Application) error {
	latest := &arcadiav1alpha1.Application{}
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(app), latest); err != nil {
		return err
	}
	if reflect.DeepEqual(app.Status, latest.Status) {
		return nil
	}
	patch := client.MergeFrom(latest.DeepCopy())
	latest.Status = app.Status
	return r.Client.Status().Patch(ctx, latest, patch, client.FieldOwner("application-controller"))
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arcadiav1alpha1.Application{}).
		Complete(r)
}

func (r *ApplicationReconciler) setCondition(app *arcadiav1alpha1.Application, condition ...arcadiav1alpha1.Condition) *arcadiav1alpha1.Application {
	app.Status.SetConditions(condition...)
	return app
}
