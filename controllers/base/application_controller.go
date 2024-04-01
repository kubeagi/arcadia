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
	"container/list"
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	agentv1alpha1 "github.com/kubeagi/arcadia/api/app-node/agent/v1alpha1"
	chainv1alpha1 "github.com/kubeagi/arcadia/api/app-node/chain/v1alpha1"
	documentloaderv1alpha1 "github.com/kubeagi/arcadia/api/app-node/documentloader/v1alpha1"
	promptv1alpha1 "github.com/kubeagi/arcadia/api/app-node/prompt/v1alpha1"
	retrieveralpha1 "github.com/kubeagi/arcadia/api/app-node/retriever/v1alpha1"
	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/appruntime"
	"github.com/kubeagi/arcadia/pkg/appruntime/base"
)

const (
	APIChainIndexKey               = "metadata.apichain"
	LLMChainIndexKey               = "metadata.llmchain"
	RetrievalQAChainIndexKey       = "metadata.retrievalqachain"
	KnowledgebaseIndexKey          = "metadata.knowledgebase"
	LLMIndexKey                    = "metadata.llm"
	PromptIndexKey                 = "metadata.prompt"
	KnowledgebaseRetrieverIndexKey = "metadata.knowledgebaseretriever"
	RerankRetrieverIndexKey        = "metadata.rerankretriever"
	MultiQueryRetrieverIndexKey    = "metadata.multiqueryretriever"
	AgentIndexKey                  = "metadata.agent"
	DocumentLoaderIndexKey         = "metadata.documentloader"
)

// ApplicationReconciler reconciles an Application object
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=applications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=applications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=applications/finalizers,verbs=update
//+kubebuilder:rbac:groups=chain.arcadia.kubeagi.k8s.com.cn,resources=apichains,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=chain.arcadia.kubeagi.k8s.com.cn,resources=apichains/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=chain.arcadia.kubeagi.k8s.com.cn,resources=apichains/finalizers,verbs=update
//+kubebuilder:rbac:groups=chain.arcadia.kubeagi.k8s.com.cn,resources=llmchains,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=chain.arcadia.kubeagi.k8s.com.cn,resources=llmchains/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=chain.arcadia.kubeagi.k8s.com.cn,resources=llmchains/finalizers,verbs=update
//+kubebuilder:rbac:groups=chain.arcadia.kubeagi.k8s.com.cn,resources=retrievalqachains,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=chain.arcadia.kubeagi.k8s.com.cn,resources=retrievalqachains/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=chain.arcadia.kubeagi.k8s.com.cn,resources=retrievalqachains/finalizers,verbs=update
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=knowledgebases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=knowledgebases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=knowledgebases/finalizers,verbs=update
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=llms,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=llms/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=llms/finalizers,verbs=update
//+kubebuilder:rbac:groups=prompt.arcadia.kubeagi.k8s.com.cn,resources=prompts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=prompt.arcadia.kubeagi.k8s.com.cn,resources=prompts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=prompt.arcadia.kubeagi.k8s.com.cn,resources=prompts/finalizers,verbs=update
//+kubebuilder:rbac:groups=retriever.arcadia.kubeagi.k8s.com.cn,resources=knowledgebaseretrievers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=retriever.arcadia.kubeagi.k8s.com.cn,resources=knowledgebaseretrievers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=retriever.arcadia.kubeagi.k8s.com.cn,resources=knowledgebaseretrievers/finalizers,verbs=update
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=agents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=agents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=agents/finalizers,verbs=update
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=documentloaders,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=documentloaders/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=documentloaders/finalizers,verbs=update

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
		return ctrl.Result{Requeue: true}, nil
	}

	if MigrateAppCategory(app) {
		return ctrl.Result{}, r.Client.Update(ctx, app)
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

	log.V(5).Info("Application Reconcile Done")
	return result, err
}

// MigrateAppCategory Migrate the application categories information in annotations to labels to support selection through labels.
//
// Deprecated: The function will be removed in 0.3.
func MigrateAppCategory(app *arcadiav1alpha1.Application) bool {
	if v, ok := app.Annotations[arcadiav1alpha1.AppCategoryLabelKey]; ok && len(validation.IsValidLabelValue(v)) == 0 {
		// TODO: In version 0.3, the categories in annotations will be removed.
		if app.Labels == nil {
			app.Labels = make(map[string]string)
		}
		app.Labels[arcadiav1alpha1.AppCategoryLabelKey] = v
		return true
	}
	return false
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
	log.V(5).Info("Start validate nodes...")
	defer log.V(5).Info("Validate nodes Done")
	var input, output int
	var outputNodeName string
	nodeName := make(map[string]bool, len(app.Spec.Nodes))
	for _, node := range app.Spec.Nodes {
		if _, ok := nodeName[node.Name]; ok {
			r.setCondition(app, app.Status.ErrorCondition("node name should be unique")...)
			return app, ctrl.Result{RequeueAfter: waitMedium}, nil
		}
		nodeName[node.Name] = true
		if node.Ref.Kind == arcadiav1alpha1.InputNode {
			input++
			if len(node.NextNodeName) == 0 {
				r.setCondition(app, app.Status.ErrorCondition("input node needs one or more next nodes")...)
				return app, ctrl.Result{RequeueAfter: waitMedium}, nil
			}
		}
		if node.Ref.Kind == arcadiav1alpha1.OutputNode {
			output++
			outputNodeName = node.Name
			if len(node.NextNodeName) != 0 {
				r.setCondition(app, app.Status.ErrorCondition("output node should not have next nodes")...)
				return app, ctrl.Result{RequeueAfter: waitMedium}, nil
			}
		}
	}
	if input != 1 {
		r.setCondition(app, app.Status.ErrorCondition("need one input node")...)
		return app, ctrl.Result{RequeueAfter: waitMedium}, nil
	}
	if output != 1 {
		r.setCondition(app, app.Status.ErrorCondition("need one output node")...)
		return app, ctrl.Result{RequeueAfter: waitMedium}, nil
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
					return app, ctrl.Result{RequeueAfter: waitMedium}, nil
				}
				// Only allow chain group or agent node as the ending node
				if *group != chainv1alpha1.Group && (*group != agentv1alpha1.Group && node.Ref.Kind != "agent") {
					r.setCondition(app, app.Status.ErrorCondition("ending node should be a chain or agent")...)
					return app, ctrl.Result{RequeueAfter: waitMedium}, nil
				}
			}
			toOutputNodeNext = len(node.NextNodeName)
		}
	}
	if toOutput != 1 {
		r.setCondition(app, app.Status.ErrorCondition("only one node can output")...)
		return app, ctrl.Result{RequeueAfter: waitMedium}, nil
	}
	if toOutputNodeNext != 1 {
		r.setCondition(app, app.Status.ErrorCondition("when this node points to output, it can only point to output")...)
		return app, ctrl.Result{RequeueAfter: waitMedium}, nil
	}

	log.V(5).Info("init runtimeApp")
	runtimeApp, err := appruntime.NewAppOrGetFromCache(ctx, r.Client, app)
	if err != nil {
		r.setCondition(app, app.Status.ErrorCondition(err.Error())...)
		return app, ctrl.Result{RequeueAfter: waitMedium}, nil
	}

	visited := make(map[string]bool)
	waitRunningNodes := list.New()
	for _, v := range runtimeApp.StartingNodes {
		waitRunningNodes.PushBack(v)
	}
	for e := waitRunningNodes.Front(); e != nil; e = e.Next() {
		e := e.Value.(base.Node)
		if !visited[e.Name()] {
			reWait := false
			for _, n := range e.GetPrevNode() {
				if !visited[n.Name()] {
					reWait = true
					break
				}
			}
			if reWait {
				waitRunningNodes.PushBack(e)
				continue
			}
			log.V(5).Info("runtimeApp try to check node...", "node", e.Name())
			if isReady, errMsg := e.Ready(); !isReady {
				r.setCondition(app, app.Status.ErrorCondition(fmt.Sprintf("%s:%s || node %s get failed status: %s", e.Group(), e.Kind(), e.Name(), errMsg))...)
				return app, ctrl.Result{RequeueAfter: waitMedium}, nil
			}
			log.V(5).Info("runtimeApp check node done", "node", e.Name())
			visited[e.Name()] = true
		}
		for _, n := range e.GetNextNode() {
			waitRunningNodes.PushBack(n)
		}
	}

	log.V(5).Info("runtimeApp check Done")
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
	// Set public label
	if app.Spec.IsPublic {
		if app.Labels == nil {
			app.Labels = make(map[string]string, 1)
		}
		app.Labels[arcadiav1alpha1.AppPublicLabelKey] = "true"
	} else {
		delete(app.Labels, arcadiav1alpha1.AppPublicLabelKey)
	}
	// Set recommended label
	if app.Spec.IsRecommended {
		if app.Labels == nil {
			app.Labels = make(map[string]string, 1)
		}
		app.Labels[arcadiav1alpha1.AppRecommendedLabelKey] = "true"
	} else {
		delete(app.Labels, arcadiav1alpha1.AppRecommendedLabelKey)
	}

	if !reflect.DeepEqual(app, appRaw) {
		return app, ctrl.Result{Requeue: true}, r.Patch(ctx, app, client.MergeFrom(appRaw))
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

type Dependency struct {
	IndexName   string
	GroupPrefix string
	Kind        string
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	dependencies := []Dependency{
		{APIChainIndexKey, "chain", "apichain"},
		{LLMChainIndexKey, "chain", "llmchain"},
		{RetrievalQAChainIndexKey, "chain", "retrievalqachain"},
		{KnowledgebaseIndexKey, "", "knowledgebase"},
		{LLMIndexKey, "", "llm"},
		{PromptIndexKey, "prompt", "prompt"},
		{KnowledgebaseRetrieverIndexKey, "retriever", "knowledgebaseretriever"},
		{RerankRetrieverIndexKey, "retriever", "rerankretriever"},
		{MultiQueryRetrieverIndexKey, "retriever", "multiqueryretriever"},
		{AgentIndexKey, "", "agent"},
		{DocumentLoaderIndexKey, "", "documentloader"},
	}
	for _, d := range dependencies {
		d := d
		if err := mgr.GetFieldIndexer().IndexField(ctx, &arcadiav1alpha1.Application{}, d.IndexName,
			func(o client.Object) []string {
				app, ok := o.(*arcadiav1alpha1.Application)
				if !ok {
					return nil
				}
				has, ns, name := appruntime.FindNodesHas(app, d.GroupPrefix, d.Kind)
				if !has {
					return nil
				}
				key := fmt.Sprintf("%s/%s", ns, name)
				return []string{key}
			},
		); err != nil {
			return err
		}
	}

	getEventHandler := func(indexKey string) handler.EventHandler {
		return handler.EnqueueRequestsFromMapFunc(func(o client.Object) (reqs []reconcile.Request) {
			var list arcadiav1alpha1.ApplicationList
			if err := r.List(ctx, &list, client.MatchingFields{indexKey: client.ObjectKeyFromObject(o).String()}); err != nil {
				ctrl.LoggerFrom(ctx).Error(err, "failed to list Application for"+indexKey)
				return nil
			}
			for _, i := range list.Items {
				i := i
				reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&i)})
			}
			return reqs
		})
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&arcadiav1alpha1.Application{}, builder.WithPredicates(predicate.Funcs{
			UpdateFunc: func(ue event.UpdateEvent) bool {
				// Avoid to handle the event that it's not spec update or delete
				o := ue.ObjectOld.(*arcadiav1alpha1.Application)
				n := ue.ObjectNew.(*arcadiav1alpha1.Application)
				return !reflect.DeepEqual(o.Spec, n.Spec) || n.DeletionTimestamp != nil
			},
		})).
		Watches(&source.Kind{Type: &chainv1alpha1.APIChain{}}, getEventHandler(APIChainIndexKey)).
		Watches(&source.Kind{Type: &chainv1alpha1.LLMChain{}}, getEventHandler(LLMChainIndexKey)).
		Watches(&source.Kind{Type: &chainv1alpha1.RetrievalQAChain{}}, getEventHandler(RetrievalQAChainIndexKey)).
		Watches(&source.Kind{Type: &arcadiav1alpha1.KnowledgeBase{}}, getEventHandler(KnowledgebaseIndexKey)).
		Watches(&source.Kind{Type: &arcadiav1alpha1.LLM{}}, getEventHandler(LLMIndexKey)).
		Watches(&source.Kind{Type: &promptv1alpha1.Prompt{}}, getEventHandler(PromptIndexKey)).
		Watches(&source.Kind{Type: &retrieveralpha1.KnowledgeBaseRetriever{}}, getEventHandler(KnowledgebaseRetrieverIndexKey)).
		Watches(&source.Kind{Type: &retrieveralpha1.RerankRetriever{}}, getEventHandler(RerankRetrieverIndexKey)).
		Watches(&source.Kind{Type: &retrieveralpha1.MultiQueryRetriever{}}, getEventHandler(MultiQueryRetrieverIndexKey)).
		Watches(&source.Kind{Type: &agentv1alpha1.Agent{}}, getEventHandler(AgentIndexKey)).
		Watches(&source.Kind{Type: &documentloaderv1alpha1.DocumentLoader{}}, getEventHandler(DocumentLoaderIndexKey)).
		Complete(r)
}

func (r *ApplicationReconciler) setCondition(app *arcadiav1alpha1.Application, condition ...arcadiav1alpha1.Condition) *arcadiav1alpha1.Application {
	app.Status.SetConditions(condition...)
	return app
}
