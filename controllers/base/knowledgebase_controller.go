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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/minio/minio-go/v7"
	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/config"
	"github.com/kubeagi/arcadia/pkg/datasource"
	pkgdocumentloaders "github.com/kubeagi/arcadia/pkg/documentloaders"
	"github.com/kubeagi/arcadia/pkg/langchainwrap"
	"github.com/kubeagi/arcadia/pkg/utils"
	"github.com/kubeagi/arcadia/pkg/vectorstore"
)

const (
	EmbedderIndexKey    = "metadata.embedder"
	VectorStoreIndexKey = "metadata.vectorstore"

	waitLonger  = time.Hour
	waitSmaller = time.Second * 3
	waitMedium  = time.Minute

	retryForFailed = "for-failed"
)

var (
	errNoSource            = fmt.Errorf("no source")
	errDataSourceNotReady  = fmt.Errorf("datasource is not ready")
	errEmbedderNotReady    = fmt.Errorf("embedder is not ready")
	errVectorStoreNotReady = fmt.Errorf("vectorstore is not ready")
	errFileSkipped         = fmt.Errorf("file is skipped")
)

// KnowledgeBaseReconciler reconciles a KnowledgeBase object
type KnowledgeBaseReconciler struct {
	client.Client
	Scheme                *runtime.Scheme
	HasHandledSuccessPath map[string]bool
	readyMu               sync.Mutex
	ReadyMap              map[string]bool
}

//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=knowledgebases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=knowledgebases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=knowledgebases/finalizers,verbs=update
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=embedders,verbs=get;list;watch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=embedders/status,verbs=get
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=versioneddataset,verbs=get;list;watch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=versioneddataset/status,verbs=get
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=vectorstores,verbs=get;list;watch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=vectorstores/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *KnowledgeBaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(5).Info("Start KnowledgeBase Reconcile")
	kb := &arcadiav1alpha1.KnowledgeBase{}
	if err := r.Get(ctx, req.NamespacedName, kb); err != nil {
		// There's no need to requeue if the resource no longer exists.
		// Otherwise, we'll be requeued implicitly because we return an error.
		log.V(1).Info("Failed to get KnowledgeBase")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log = log.WithValues("Generation", kb.GetGeneration(), "ObservedGeneration", kb.Status.ObservedGeneration, "creator", kb.Spec.Creator)
	log.V(5).Info("Get KnowledgeBase instance")

	// Check if the KnowledgeBase instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if kb.GetDeletionTimestamp() != nil && controllerutil.ContainsFinalizer(kb, arcadiav1alpha1.Finalizer) {
		log.Info("Performing Finalizer Operations for KnowledgeBase before delete CR")
		r.reconcileDelete(ctx, log, kb)
		log.Info("Removing Finalizer for KnowledgeBase after successfully performing the operations")
		controllerutil.RemoveFinalizer(kb, arcadiav1alpha1.Finalizer)
		if err = r.Update(ctx, kb); err != nil {
			log.Error(err, "Failed to remove finalizer for KnowledgeBase")
			return ctrl.Result{}, err
		}
		log.Info("Remove KnowledgeBase done")
		return ctrl.Result{}, nil
	}

	// Add a finalizer.Then, we can define some operations which should
	// occur before the KnowledgeBase to be deleted.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers
	if newAdded := controllerutil.AddFinalizer(kb, arcadiav1alpha1.Finalizer); newAdded {
		log.Info("Try to add Finalizer for KnowledgeBase")
		if err = r.Update(ctx, kb); err != nil {
			log.Error(err, "Failed to update KnowledgeBase to add finalizer, will try again later")
			return ctrl.Result{}, err
		}
		log.Info("Adding Finalizer for KnowledgeBase done")
		return ctrl.Result{}, nil
	}

	// The previous version of the knowledge base used the paths field to store files.
	// After the change, the files field is used.
	// without migration, which will lead to incorrect data display and the inability to vectorize the data normally.
	if migrated := r.migratePaths2Files(ctx, kb); migrated {
		log.Info("start to migrate files")
		return reconcile.Result{}, r.Client.Update(ctx, kb)
	}
	if len(kb.Status.Conditions) == 0 {
		log.Info("start to set Pending Condition")
		kb = r.setCondition(log, kb, kb.PendingCondition(""))
		return reconcile.Result{}, r.patchStatus(ctx, log, kb)
	}

	if kb.Status.ObservedGeneration != kb.Generation {
		kb.Status.ObservedGeneration = kb.Generation
		log.Info("start to set InitCondition")
		kb = r.setCondition(log, kb, kb.InitCondition())
		return reconcile.Result{}, r.patchStatus(ctx, log, kb)
	}

	if v := kb.Annotations[arcadiav1alpha1.UpdateSourceFileAnnotationKey]; v != "" {
		log.Info("Manual update")
		kbNew := kb.DeepCopy()
		if v != retryForFailed && len(kb.Status.FileGroupDetail) != 0 {
			log.Info("set FileGroupDetail to nil to redo embedder...")
			kbNew.Status.FileGroupDetail = nil
			kbNew = r.setCondition(log, kbNew, kbNew.InitCondition())
			return reconcile.Result{}, r.patchStatus(ctx, log, kbNew)
		}
		if v == retryForFailed {
			found := false
			for out, fg := range kbNew.Status.FileGroupDetail {
				for in, f := range fg.FileDetails {
					if f.Phase == arcadiav1alpha1.FileProcessPhaseFailed {
						found = true
						kbNew.Status.FileGroupDetail[out].FileDetails[in].Phase = arcadiav1alpha1.FileProcessPhaseProcessing
						kbNew.Status.FileGroupDetail[out].FileDetails[in].LastUpdateTime = metav1.Now()
						kbNew.Status.FileGroupDetail[out].FileDetails[in].ErrMessage = ""
					}
				}
			}
			if found {
				log.Info("there are files that failed to be processed and are ready to try again.")
				kbNew = r.setCondition(log, kbNew, kbNew.InitCondition())
				return reconcile.Result{}, r.patchStatus(ctx, log, kbNew)
			}
		}
		delete(kbNew.Annotations, arcadiav1alpha1.UpdateSourceFileAnnotationKey)
		return reconcile.Result{}, r.Patch(ctx, kbNew, client.MergeFrom(kb))
	}

	dp := kb.DeepCopy()
	if r.syncStatus(ctx, dp) {
		log.V(5).Info(fmt.Sprintf("status is different from spec. new status: %+v\n, old status: %+v", dp.Status.FileGroupDetail, kb.Status.FileGroupDetail))
		return reconcile.Result{}, r.Client.Status().Patch(ctx, dp, client.MergeFrom(kb))
	}

	log.Info("start to reconcile")
	return r.reconcile(ctx, log, kb)
}

func (r *KnowledgeBaseReconciler) patchStatus(ctx context.Context, log logr.Logger, kb *arcadiav1alpha1.KnowledgeBase) error {
	latest := &arcadiav1alpha1.KnowledgeBase{}
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(kb), latest); err != nil {
		return err
	}
	if reflect.DeepEqual(kb.Status, latest.Status) {
		log.V(5).Info("status not changed, skip")
		return nil
	}
	if r.isReady(kb) && !kb.Status.IsReady() {
		log.V(5).Info("status is ready,but not get it from cluster, has cache, skip update status")
		return nil
	}
	log.V(5).Info(fmt.Sprintf("try to patch status %#v", kb.Status))
	patch := client.MergeFrom(latest.DeepCopy())
	latest.Status = kb.Status
	return r.Client.Status().Patch(ctx, latest, patch, client.FieldOwner("knowledgebase-controller"))
}

// SetupWithManager sets up the controller with the Manager.
func (r *KnowledgeBaseReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(ctx, &arcadiav1alpha1.KnowledgeBase{}, EmbedderIndexKey,
		func(o client.Object) []string {
			kb, ok := o.(*arcadiav1alpha1.KnowledgeBase)
			if !ok {
				return nil
			}
			if kb.Spec.Embedder == nil || kb.Spec.Embedder.Name == "" {
				return nil
			}
			return []string{
				fmt.Sprintf("%s/%s", kb.Spec.Embedder.GetNamespace(kb.Namespace), kb.Spec.Embedder.Name),
			}
		},
	); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(ctx, &arcadiav1alpha1.KnowledgeBase{}, VectorStoreIndexKey,
		func(o client.Object) []string {
			kb, ok := o.(*arcadiav1alpha1.KnowledgeBase)
			if !ok {
				return nil
			}
			if kb.Spec.VectorStore == nil || kb.Spec.VectorStore.Name == "" {
				return nil
			}
			return []string{
				fmt.Sprintf("%s/%s", kb.Spec.VectorStore.GetNamespace(kb.Namespace), kb.Spec.VectorStore.Name),
			}
		},
	); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&arcadiav1alpha1.KnowledgeBase{}).
		Watches(&source.Kind{Type: &arcadiav1alpha1.Embedder{}},
			handler.EnqueueRequestsFromMapFunc(func(o client.Object) (reqs []reconcile.Request) {
				var list arcadiav1alpha1.KnowledgeBaseList
				if err := r.List(ctx, &list, client.MatchingFields{EmbedderIndexKey: client.ObjectKeyFromObject(o).String()}); err != nil {
					ctrl.LoggerFrom(ctx).Error(err, "failed to list Knowlegebase for embedder changes")
					return nil
				}
				for _, i := range list.Items {
					i := i
					reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&i)})
				}
				return reqs
			})).
		Watches(&source.Kind{Type: &arcadiav1alpha1.VectorStore{}},
			handler.EnqueueRequestsFromMapFunc(func(o client.Object) (reqs []reconcile.Request) {
				var list arcadiav1alpha1.KnowledgeBaseList
				if err := r.List(ctx, &list, client.MatchingFields{VectorStoreIndexKey: client.ObjectKeyFromObject(o).String()}); err != nil {
					ctrl.LoggerFrom(ctx).Error(err, "failed to list Knowlegebase for vectorstore changes")
					return nil
				}
				for _, i := range list.Items {
					i := i
					reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&i)})
				}
				return reqs
			})).
		Complete(r)
}

func (r *KnowledgeBaseReconciler) reconcile(ctx context.Context, log logr.Logger, kb *arcadiav1alpha1.KnowledgeBase) (ctrl.Result, error) {
	// Observe generation change or manual update
	embedderReq := kb.Spec.Embedder
	vectorStoreReq := kb.Spec.VectorStore
	fileGroupsReq := kb.Spec.FileGroups
	if embedderReq == nil || vectorStoreReq == nil || len(fileGroupsReq) == 0 {
		kb = r.setCondition(log, kb, kb.PendingCondition("embedder or vectorstore or filegroups is not setting"))
		return ctrl.Result{}, r.patchStatus(ctx, log, kb)
	}

	embedder := &arcadiav1alpha1.Embedder{}
	if err := r.Get(ctx, types.NamespacedName{Name: kb.Spec.Embedder.Name, Namespace: kb.Spec.Embedder.GetNamespace(kb.GetNamespace())}, embedder); err != nil {
		log.Info("get embedder error " + err.Error())
		if apierrors.IsNotFound(err) {
			kb = r.setCondition(log, kb, kb.PendingCondition("embedder is not found"))
		} else {
			kb = r.setCondition(log, kb, kb.ErrorCondition(err.Error()))
		}
		return ctrl.Result{}, r.patchStatus(ctx, log, kb)
	}
	if !embedder.Status.IsReady() {
		log.Info(fmt.Sprintf("embedder %s is not ready", embedder.Name))
		kb = r.setCondition(log, kb, kb.ErrorCondition(errEmbedderNotReady.Error()))
		return ctrl.Result{}, r.patchStatus(ctx, log, kb)
	}

	vectorStore := &arcadiav1alpha1.VectorStore{}
	if err := r.Get(ctx, types.NamespacedName{Name: kb.Spec.VectorStore.Name, Namespace: kb.Spec.VectorStore.GetNamespace(kb.GetNamespace())}, vectorStore); err != nil {
		log.Info("get vectorstore error " + err.Error())
		if apierrors.IsNotFound(err) {
			kb = r.setCondition(log, kb, kb.PendingCondition("vectorStore is not found"))
		} else {
			kb = r.setCondition(log, kb, kb.ErrorCondition(err.Error()))
		}
		return ctrl.Result{}, r.patchStatus(ctx, log, kb)
	}
	if !vectorStore.Status.IsReady() {
		log.Info(fmt.Sprintf("vectorstore %s is not ready", vectorStore.Name))
		kb = r.setCondition(log, kb, kb.ErrorCondition(errVectorStoreNotReady.Error()))
		return ctrl.Result{}, r.patchStatus(ctx, log, kb)
	}

	if kb.Status.IsReady() || r.isReady(kb) {
		log.Info("KnowledgeBase is ready, skip reconcile")
		return ctrl.Result{}, nil
	}

	haveFailed := false
	for out, fg := range kb.Status.FileGroupDetail {
		if fg.Source == nil {
			log.Info(fmt.Sprintf("kb.Status.FileGroupDetail[%d] source is nil, skip", out))
			continue
		}
		for in, f := range fg.FileDetails {
			if f.Phase == arcadiav1alpha1.FileProcessPhaseSkipped {
				log.Info(fmt.Sprintf("source %s/%s, file %s, the current phase is skip and will not be processed.", fg.Source.Kind, fg.Source.Name, f.Path))
				continue
			}
			if f.Phase == arcadiav1alpha1.FileProcessPhasePending {
				log.V(5).Info(fmt.Sprintf("source: %s/%s file: %s, cur is Pending,change it to Processing", fg.Source.Kind, fg.Source.Name, f.Path))
				kb.Status.FileGroupDetail[out].FileDetails[in].Phase = arcadiav1alpha1.FileProcessPhaseProcessing
				kb.Status.FileGroupDetail[out].FileDetails[in].LastUpdateTime = metav1.Now()
				return ctrl.Result{}, r.patchStatus(ctx, log, kb)
			}
			if f.Phase == arcadiav1alpha1.FileProcessPhaseFailed {
				log.Info(fmt.Sprintf("source: %s/%s, file: %s, is failed skip.", fg.Source.Kind, fg.Source.Name, f.Path))
				haveFailed = true
				continue
			}
			if f.Phase == arcadiav1alpha1.FileProcessPhaseSucceeded {
				log.Info(fmt.Sprintf("source %s/%s, file %s, processing completed", fg.Source.Kind, fg.Source.Name, f.Path))
				continue
			}
			if f.Phase == arcadiav1alpha1.FileProcessPhaseProcessing {
				log.Info(fmt.Sprintf("source: %s/%s, file: %s, is Processing", fg.Source.Kind, fg.Source.Name, f.Path))
				err := r.reconcileFileGroup(ctx, log, kb, vectorStore, embedder, out, in)
				if err != nil {
					log.Error(err, "failed to handle single file", "FileName", f.Path)
				}
				return ctrl.Result{Requeue: true}, r.patchStatus(ctx, log, kb)
			}
		}
	}
	if haveFailed {
		r.setCondition(log, kb, kb.ErrorCondition("some files failed to process."))
		return ctrl.Result{RequeueAfter: waitMedium}, r.patchStatus(ctx, log, kb)
	}
	if kb.Status.Conditions[0].Status != corev1.ConditionTrue {
		kb = r.setCondition(log, kb, kb.ReadyCondition())
	}
	return ctrl.Result{}, r.patchStatus(ctx, log, kb)
}

func (r *KnowledgeBaseReconciler) setCondition(log logr.Logger, kb *arcadiav1alpha1.KnowledgeBase, condition ...arcadiav1alpha1.Condition) *arcadiav1alpha1.KnowledgeBase {
	ready := false
	for _, c := range condition {
		if c.Type == arcadiav1alpha1.TypeReady && c.Status == corev1.ConditionTrue {
			ready = true
			break
		}
	}
	if ready {
		r.ready(log, kb)
	} else {
		r.unready(log, kb)
	}
	kb.Status.SetConditions(condition...)
	return kb
}

func (r *KnowledgeBaseReconciler) reconcileFileGroup(
	ctx context.Context,
	log logr.Logger,
	kb *arcadiav1alpha1.KnowledgeBase,
	vectorStore *arcadiav1alpha1.VectorStore,
	embedder *arcadiav1alpha1.Embedder,
	groupIndex, fileIndex int,
) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to reconcile FileGroup: %w", err)
		}
	}()
	group := kb.Status.FileGroupDetail[groupIndex]
	fileDetail := kb.Status.FileGroupDetail[groupIndex].FileDetails[fileIndex]

	ns := kb.Namespace
	if group.Source.Namespace != nil {
		ns = *group.Source.Namespace
	}

	lowerKind := strings.ToLower(group.Source.Kind)
	var ds datasource.Datasource
	info := &arcadiav1alpha1.OSS{Bucket: ns}
	var vsBasePath string
	switch lowerKind {
	case "versioneddataset":
		versionedDataset := &arcadiav1alpha1.VersionedDataset{}
		if err = r.Get(ctx, types.NamespacedName{Name: group.Source.Name, Namespace: ns}, versionedDataset); err != nil {
			if apierrors.IsNotFound(err) {
				return errNoSource
			}
			return err
		}
		if versionedDataset.Spec.Dataset == nil {
			return fmt.Errorf("versionedDataset.Spec.Dataset is nil")
		}
		if !versionedDataset.Status.IsReady() {
			return errDataSourceNotReady
		}
		system, err := config.GetSystemDatasource(ctx)
		if err != nil {
			return err
		}
		endpoint := system.Spec.Endpoint.DeepCopy()
		if endpoint != nil && endpoint.AuthSecret != nil {
			endpoint.AuthSecret.WithNameSpace(system.Namespace)
		}
		ds, err = datasource.NewLocal(ctx, r.Client, endpoint)
		if err != nil {
			return err
		}
		// basepath for this versioneddataset
		vsBasePath = filepath.Join("dataset", versionedDataset.Spec.Dataset.Name, versionedDataset.Spec.Version)
		info.Object = filepath.Join(vsBasePath, fileDetail.Path)

	case "datasource", "":
		dsObj := &arcadiav1alpha1.Datasource{}
		if err = r.Get(ctx, types.NamespacedName{Name: group.Source.Name, Namespace: ns}, dsObj); err != nil {
			if apierrors.IsNotFound(err) {
				return errNoSource
			}
			return err
		}
		if !dsObj.Status.IsReady() {
			return errDataSourceNotReady
		}
		// set endpoint's auth secret namespace to current datasource if not set
		endpoint := dsObj.Spec.Endpoint.DeepCopy()
		if endpoint != nil && endpoint.AuthSecret != nil {
			endpoint.AuthSecret.WithNameSpace(dsObj.Namespace)
		}
		ds, err = datasource.NewOSS(ctx, r.Client, endpoint)
		if err != nil {
			return err
		}
		// for none-conversation knowledgebase, bucket is the same as datasource.
		if kb.Spec.Type != arcadiav1alpha1.KnowledgeBaseTypeConversation {
			info.Bucket = dsObj.Spec.OSS.Bucket
		}

		info.Object = fileDetail.Path
	default:
		return fmt.Errorf("source type %s not supported yet", group.Source.Kind)
	}

	info.VersionID = fileDetail.Version

	stat, err := ds.StatFile(ctx, info)
	log.V(5).Info(fmt.Sprintf("raw StatFile:%#v", stat), "path", fileDetail.Path)
	if err != nil {
		log.Error(err, fmt.Sprintf("stat file failed. source: %s/%s/%s, file: %s, error: %s",
			group.Source.Kind, ns, group.Source.Name, fileDetail.Path, err))
		kb.Status.FileGroupDetail[groupIndex].FileDetails[fileIndex].UpdateErr(err, arcadiav1alpha1.FileProcessPhaseFailed)
		return err
	}

	objectStat, ok := stat.(minio.ObjectInfo)
	log.V(5).Info(fmt.Sprintf("minio StatFile:%#v", objectStat), "path", fileDetail.Path)
	if !ok {
		err = fmt.Errorf("failed to convert stat to minio.ObjectInfo:%s", fileDetail.Path)
		kb.Status.FileGroupDetail[groupIndex].FileDetails[fileIndex].UpdateErr(err, arcadiav1alpha1.FileProcessPhaseFailed)
		return err
	}
	kb.Status.FileGroupDetail[groupIndex].FileDetails[fileIndex].Version = fileDetail.Version
	if objectStat.ETag == fileDetail.Checksum {
		kb.Status.FileGroupDetail[groupIndex].FileDetails[fileIndex].Phase = arcadiav1alpha1.FileProcessPhaseSucceeded
		return nil
	}
	kb.Status.FileGroupDetail[groupIndex].FileDetails[fileIndex].Checksum = objectStat.ETag

	tags, err := ds.GetTags(ctx, info)
	if err != nil {
		fileDetail.UpdateErr(err, arcadiav1alpha1.FileProcessPhaseFailed)
		kb.Status.FileGroupDetail[groupIndex].FileDetails[fileIndex].UpdateErr(err, arcadiav1alpha1.FileProcessPhaseFailed)
		return err
	}

	kb.Status.FileGroupDetail[groupIndex].FileDetails[fileIndex].Size = utils.BytesToSizedStr(objectStat.Size)
	// File Type in string
	kb.Status.FileGroupDetail[groupIndex].FileDetails[fileIndex].Type = tags[arcadiav1alpha1.ObjectCountTag]
	// File data count in string
	kb.Status.FileGroupDetail[groupIndex].FileDetails[fileIndex].Count = tags[arcadiav1alpha1.ObjectCountTag]

	file, err := ds.ReadFile(ctx, info)
	if err != nil {
		kb.Status.FileGroupDetail[groupIndex].FileDetails[fileIndex].UpdateErr(err, arcadiav1alpha1.FileProcessPhaseFailed)
		return err
	}
	defer file.Close()
	startTime := time.Now()
	if err = r.handleFile(ctx, log, file, info.Object, tags, kb, vectorStore, embedder); err != nil {
		if errors.Is(err, errFileSkipped) {
			kb.Status.FileGroupDetail[groupIndex].FileDetails[fileIndex].UpdateErr(err, arcadiav1alpha1.FileProcessPhaseSkipped)
		} else {
			kb.Status.FileGroupDetail[groupIndex].FileDetails[fileIndex].UpdateErr(err, arcadiav1alpha1.FileProcessPhaseFailed)
		}
		return err
	}
	cost := int64(time.Since(startTime).Milliseconds())

	kb.Status.FileGroupDetail[groupIndex].FileDetails[fileIndex].TimeCost = cost
	log.Info("handle FileGroup succeeded", "timecost(milliseconds)", cost)
	kb.Status.FileGroupDetail[groupIndex].FileDetails[fileIndex].UpdateErr(err, arcadiav1alpha1.FileProcessPhaseSucceeded)
	return nil
}

func (r *KnowledgeBaseReconciler) handleFile(ctx context.Context, log logr.Logger, file io.ReadCloser, fileName string, tags map[string]string, kb *arcadiav1alpha1.KnowledgeBase, store *arcadiav1alpha1.VectorStore, embedder *arcadiav1alpha1.Embedder) (err error) {
	log = log.WithValues("fileName", fileName, "tags", tags)
	if !embedder.Status.IsReady() {
		return errEmbedderNotReady
	}
	if !store.Status.IsReady() {
		return errVectorStoreNotReady
	}
	embeddingOptions := kb.EmbeddingOptions()
	em, err := langchainwrap.GetLangchainEmbedder(ctx, embedder, r.Client, "", embeddings.WithBatchSize(embeddingOptions.BatchSize))
	if err != nil {
		return err
	}
	data, err := io.ReadAll(file) // TODO Load large files in pieces to save memory
	// TODO Line or single line byte exceeds embedder limit
	if err != nil {
		return err
	}
	dataReader := bytes.NewReader(data)
	var documents []schema.Document
	var loader documentloaders.Loader
	switch filepath.Ext(fileName) {
	case ".txt":
		loader = documentloaders.NewText(dataReader)
	case ".csv":
		v, ok := tags[arcadiav1alpha1.ObjectTypeTag]
		if ok && v == arcadiav1alpha1.ObjectTypeQA {
			// for qa csv,we skip the text splitter
			loader = pkgdocumentloaders.NewQACSV(dataReader, fileName)
		} else {
			loader = documentloaders.NewCSV(dataReader)
		}
	case ".html", ".htm":
		loader = documentloaders.NewHTML(dataReader)
	case ".pdf":
		loader = pkgdocumentloaders.NewPDF(dataReader, fileName)
	// TODO: support .mp3,.wav
	default:
		loader = documentloaders.NewText(dataReader)
	}

	// initialize text splitter
	// var split textsplitter.TextSplitter
	split := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(embeddingOptions.ChunkSize),
		textsplitter.WithChunkOverlap(pointer.IntDeref(embeddingOptions.ChunkOverlap, arcadiav1alpha1.DefaultChunkOverlap)),
	)
	// switch {
	// case "token":
	//	split = textsplitter.NewTokenSplitter(
	//		textsplitter.WithChunkSize(chunkSize),
	//		textsplitter.WithChunkOverlap(chunkOverlap),
	//	)
	// case "markdown":
	//	split = textsplitter.NewMarkdownTextSplitter(
	//		textsplitter.WithChunkSize(chunkSize),
	//		textsplitter.WithChunkOverlap(chunkOverlap),
	//	)
	//default:
	//	split = textsplitter.NewRecursiveCharacter(
	//		textsplitter.WithChunkSize(chunkSize),
	//		textsplitter.WithChunkOverlap(chunkOverlap),
	//	)
	//}

	documents, err = loader.LoadAndSplit(ctx, split)
	if err != nil {
		return err
	}

	return vectorstore.AddDocuments(ctx, log, store, em, kb.VectorStoreCollectionName(), r.Client, documents)
}

func (r *KnowledgeBaseReconciler) reconcileDelete(ctx context.Context, log logr.Logger, kb *arcadiav1alpha1.KnowledgeBase) {
	// r.cleanupHasHandledSuccessPath(kb)
	// r.unready(log, kb)
	vectorStore := &arcadiav1alpha1.VectorStore{}
	if err := r.Get(ctx, types.NamespacedName{Name: kb.Spec.VectorStore.Name, Namespace: kb.Spec.VectorStore.GetNamespace(kb.GetNamespace())}, vectorStore); err != nil {
		log.Error(err, "reconcile delete: get vector store error, may leave garbage data")
		return
	}
	// Sometimes the deletion action can jam the reconciler goroutine, the deletion is a best effort and we don't want it to block the current goroutine
	go func() {
		log.V(3).Info("remove vector store collection start")
		_ = vectorstore.RemoveCollection(ctx, log, vectorStore, kb.VectorStoreCollectionName(), r.Client)
		log.V(3).Info("remove vector store collection done")
	}()
}

func (r *KnowledgeBaseReconciler) ready(log logr.Logger, kb *arcadiav1alpha1.KnowledgeBase) {
	r.readyMu.Lock()
	defer r.readyMu.Unlock()
	log.V(5).Info("ready")
	r.ReadyMap[string(kb.GetUID())] = true
}

func (r *KnowledgeBaseReconciler) unready(log logr.Logger, kb *arcadiav1alpha1.KnowledgeBase) {
	r.readyMu.Lock()
	defer r.readyMu.Unlock()
	log.V(5).Info("unready")
	delete(r.ReadyMap, string(kb.GetUID()))
}

func (r *KnowledgeBaseReconciler) isReady(kb *arcadiav1alpha1.KnowledgeBase) bool {
	v, ok := r.ReadyMap[string(kb.GetUID())]
	return ok && v
}

// migratePaths2Files The paths field will be deprecated in version 0.3,
// the function is compatible with old version data in the cluster.
func (r *KnowledgeBaseReconciler) migratePaths2Files(ctx context.Context, kb *arcadiav1alpha1.KnowledgeBase) bool {
	migrated := false
	logger, _ := logr.FromContext(ctx)
	for idx, fg := range kb.Spec.FileGroups {
		if len(fg.Paths) == 0 { // nolint
			logger.Info(fmt.Sprintf("source: %v don't have any file", fg))
			continue
		}
		if kb.Spec.FileGroups[idx].Files == nil {
			kb.Spec.FileGroups[idx].Files = make([]arcadiav1alpha1.FileWithVersion, 0)
		}
		exists := make(map[string]struct{})
		for _, f := range kb.Spec.FileGroups[idx].Files {
			exists[f.Path] = struct{}{}
		}
		for _, p := range fg.Paths { // nolint
			if _, ok := exists[p]; ok {
				continue
			}
			kb.Spec.FileGroups[idx].Files = append(kb.Spec.FileGroups[idx].Files, arcadiav1alpha1.FileWithVersion{Path: p})
			migrated = true
		}
		kb.Spec.FileGroups[idx].Paths = nil // nolint
	}
	return migrated
}

func isFileDetailDiff(a, b arcadiav1alpha1.FileDetails) bool {
	return a.Path != b.Path ||
		a.Type != b.Type ||
		a.Count != b.Count ||
		a.Size != b.Size ||
		a.Checksum != b.Checksum ||
		a.TimeCost != b.TimeCost ||
		a.Phase != b.Phase ||
		a.ErrMessage != b.ErrMessage ||
		a.Version != b.Version
}

func (r *KnowledgeBaseReconciler) syncStatus(ctx context.Context, kb *arcadiav1alpha1.KnowledgeBase) bool {
	log, _ := logr.FromContext(ctx)
	newStatus := make([]arcadiav1alpha1.FileGroupDetail, 0)
	specSource := make(map[string]map[string][2]int)
	now := metav1.Now()
	for _, fg := range kb.Spec.FileGroups {
		if fg.Source == nil {
			continue
		}

		ns := kb.Namespace
		if fg.Source.Namespace != nil {
			ns = *fg.Source.Namespace
		}
		key := fmt.Sprintf("%s/%s/%s", fg.Source.Kind, ns, fg.Source.Name)
		if _, ok := specSource[key]; !ok {
			specSource[key] = make(map[string][2]int)
			newStatus = append(newStatus, arcadiav1alpha1.FileGroupDetail{Source: fg.Source.DeepCopy(), FileDetails: make([]arcadiav1alpha1.FileDetails, 0)})
		}

		index := len(newStatus) - 1

		for i, f := range fg.Files {
			newStatus[index].FileDetails = append(newStatus[index].FileDetails, arcadiav1alpha1.FileDetails{
				Path:           f.Path,
				Version:        f.Version,
				Phase:          arcadiav1alpha1.FileProcessPhasePending,
				LastUpdateTime: now,
			})
			specSource[key][f.Path] = [2]int{index, i}
		}
	}

	for _, fgd := range kb.Status.FileGroupDetail {
		ns := kb.Namespace
		if fgd.Source.Namespace != nil {
			ns = *fgd.Source.Namespace
		}
		key := fmt.Sprintf("%s/%s/%s", fgd.Source.Kind, ns, fgd.Source.Name)
		fileDetails, ok := specSource[key]
		if !ok {
			continue
		}
		for _, f := range fgd.FileDetails {
			v, ok := fileDetails[f.Path]
			if !ok {
				continue
			}
			vv := newStatus[v[0]].FileDetails[v[1]].Version
			newStatus[v[0]].FileDetails[v[1]] = f
			if newStatus[v[0]].FileDetails[v[1]].Version != vv {
				newStatus[v[0]].FileDetails[v[1]].Version = vv
				newStatus[v[0]].FileDetails[v[1]].Phase = arcadiav1alpha1.FileProcessPhaseProcessing
			}
		}
	}
	log.V(5).Info(fmt.Sprintf("new Status: %+v\n", newStatus))
	log.V(5).Info(fmt.Sprintf("old status: %+v\n", kb.Status.FileGroupDetail))
	if len(newStatus) != len(kb.Status.FileGroupDetail) {
		kb.Status.FileGroupDetail = newStatus
		return true
	}

	for i := 0; i < len(newStatus); i++ {
		if len(newStatus[i].FileDetails) != len(kb.Status.FileGroupDetail[i].FileDetails) {
			log.V(5).Info(fmt.Sprintf("len diff %+v | %+v\n", newStatus[i].FileDetails,
				kb.Status.FileGroupDetail[i].FileDetails))
			kb.Status.FileGroupDetail = newStatus
			return true
		}
		for j := range newStatus[i].FileDetails {
			if isFileDetailDiff(newStatus[i].FileDetails[j], kb.Status.FileGroupDetail[i].FileDetails[j]) {
				log.V(5).Info(fmt.Sprintf("fileDetail diff %+v | %+v\n",
					newStatus[i].FileDetails[j], kb.Status.FileGroupDetail[i].FileDetails[j]))
				kb.Status.FileGroupDetail = newStatus
				return true
			}
		}
	}
	return false
}
