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
)

var (
	errNoSource            = fmt.Errorf("no source")
	errDataSourceNotReady  = fmt.Errorf("datasource is not ready")
	errEmbedderNotReady    = fmt.Errorf("embedder is not ready")
	errVectorStoreNotReady = fmt.Errorf("vector store is not ready")
	errFileSkipped         = fmt.Errorf("file is skipped")
)

// KnowledgeBaseReconciler reconciles a KnowledgeBase object
type KnowledgeBaseReconciler struct {
	client.Client
	Scheme                *runtime.Scheme
	mu                    sync.Mutex
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

	kb, result, err = r.reconcile(ctx, log, kb)

	// Update status after reconciliation.
	if updateStatusErr := r.patchStatus(ctx, log, kb); updateStatusErr != nil {
		log.Error(updateStatusErr, "unable to update status after reconciliation")
		return ctrl.Result{Requeue: true}, updateStatusErr
	}
	log.V(5).Info("Reconcile done")

	return result, err
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

func (r *KnowledgeBaseReconciler) reconcile(ctx context.Context, log logr.Logger, kb *arcadiav1alpha1.KnowledgeBase) (*arcadiav1alpha1.KnowledgeBase, ctrl.Result, error) {
	// Observe generation change or manual update
	if kb.Status.ObservedGeneration != kb.Generation || kb.Annotations[arcadiav1alpha1.UpdateSourceFileAnnotationKey] != "" {
		r.cleanupHasHandledSuccessPath(kb)
		if kb.Status.ObservedGeneration != kb.Generation {
			log.Info("Generation changed")
			kb.Status.ObservedGeneration = kb.Generation
		}
		kb = r.setCondition(log, kb, kb.InitCondition())
		if updateStatusErr := r.patchStatus(ctx, log, kb); updateStatusErr != nil {
			log.Error(updateStatusErr, "unable to update status after generation update")
			return kb, ctrl.Result{Requeue: true}, updateStatusErr
		}
		if kb.Annotations[arcadiav1alpha1.UpdateSourceFileAnnotationKey] != "" {
			log.Info("Manual update")
			kbNew := kb.DeepCopy()
			delete(kbNew.Annotations, arcadiav1alpha1.UpdateSourceFileAnnotationKey)
			err := r.Patch(ctx, kbNew, client.MergeFrom(kb))
			if err != nil {
				return kb, ctrl.Result{Requeue: true}, err
			}
		}
		return kb, ctrl.Result{}, nil
	}

	embedderReq := kb.Spec.Embedder
	vectorStoreReq := kb.Spec.VectorStore
	fileGroupsReq := kb.Spec.FileGroups
	if embedderReq == nil || vectorStoreReq == nil || len(fileGroupsReq) == 0 {
		kb = r.setCondition(log, kb, kb.PendingCondition("embedder or vectorstore or filegroups is not setting"))
		return kb, ctrl.Result{}, nil
	}

	embedder := &arcadiav1alpha1.Embedder{}
	if err := r.Get(ctx, types.NamespacedName{Name: kb.Spec.Embedder.Name, Namespace: kb.Spec.Embedder.GetNamespace(kb.GetNamespace())}, embedder); err != nil {
		if apierrors.IsNotFound(err) {
			kb = r.setCondition(log, kb, kb.PendingCondition("embedder is not found"))
			return kb, ctrl.Result{RequeueAfter: waitLonger}, nil
		}
		kb = r.setCondition(log, kb, kb.ErrorCondition(err.Error()))
		return kb, ctrl.Result{}, err
	}
	if !embedder.Status.IsReady() {
		kb = r.setCondition(log, kb, kb.ErrorCondition(errEmbedderNotReady.Error()))
		return kb, ctrl.Result{RequeueAfter: waitMedium}, nil
	}

	vectorStore := &arcadiav1alpha1.VectorStore{}
	if err := r.Get(ctx, types.NamespacedName{Name: kb.Spec.VectorStore.Name, Namespace: kb.Spec.VectorStore.GetNamespace(kb.GetNamespace())}, vectorStore); err != nil {
		if apierrors.IsNotFound(err) {
			kb = r.setCondition(log, kb, kb.PendingCondition("vectorStore is not found"))
			return kb, ctrl.Result{RequeueAfter: waitLonger}, nil
		}
		kb = r.setCondition(log, kb, kb.ErrorCondition(err.Error()))
		return kb, ctrl.Result{}, err
	}
	if !vectorStore.Status.IsReady() {
		kb = r.setCondition(log, kb, kb.ErrorCondition(errVectorStoreNotReady.Error()))
		return kb, ctrl.Result{RequeueAfter: waitMedium}, nil
	}

	if kb.Status.IsReady() || r.isReady(kb) {
		log.Info("KnowledgeBase is ready, skip reconcile")
		return kb, ctrl.Result{}, nil
	}

	errs := make([]error, 0)
	for _, fileGroup := range kb.Spec.FileGroups {
		if err := r.reconcileFileGroup(ctx, log, kb, vectorStore, embedder, fileGroup); err != nil {
			log.Error(err, "Failed to reconcile FileGroup", "fileGroup", fileGroup)
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		kb = r.setCondition(log, kb, kb.ErrorCondition(err.Error()))
		return kb, ctrl.Result{RequeueAfter: waitMedium}, nil
	} else {
		for _, fileGroupDetail := range kb.Status.FileGroupDetail {
			for _, fileDetail := range fileGroupDetail.FileDetails {
				if fileDetail.Phase == arcadiav1alpha1.FileProcessPhaseFailed && fileDetail.ErrMessage != "" {
					kb = r.setCondition(log, kb, kb.ErrorCondition(fileDetail.ErrMessage))
					return kb, ctrl.Result{RequeueAfter: waitMedium}, nil
				}
			}
		}
		kb = r.setCondition(log, kb, kb.ReadyCondition())
	}
	return kb, ctrl.Result{}, nil
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

func (r *KnowledgeBaseReconciler) reconcileFileGroup(ctx context.Context, log logr.Logger, kb *arcadiav1alpha1.KnowledgeBase, vectorStore *arcadiav1alpha1.VectorStore, embedder *arcadiav1alpha1.Embedder, group arcadiav1alpha1.FileGroup) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to reconcile FileGroup: %w", err)
		}
	}()

	if group.Source == nil {
		return errNoSource
	}

	ns := kb.Namespace
	if group.Source.Namespace != nil {
		ns = *group.Source.Namespace
	}

	var ds datasource.Datasource
	info := &arcadiav1alpha1.OSS{Bucket: ns}
	var vsBasePath string
	switch strings.ToLower(group.Source.Kind) {
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
		system, err := config.GetSystemDatasource(ctx, r.Client)
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
	case "datasource":
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
		ds, err = datasource.NewOSS(ctx, r.Client, &dsObj.Spec.Endpoint)
		if err != nil {
			return err
		}
		// for none-conversation knowledgebase, bucket is the same as datasource.
		if kb.Spec.Type != arcadiav1alpha1.KnowledgeBaseTypeConversation {
			info.Bucket = dsObj.Spec.OSS.Bucket
		}
	}

	if len(kb.Status.FileGroupDetail) == 0 {
		// brand new knowledgebase, init status.
		kb.Status.FileGroupDetail = make([]arcadiav1alpha1.FileGroupDetail, 1)
		kb.Status.FileGroupDetail[0].Init(group)
		log.V(5).Info("init filegroupdetail status")
	}
	var fileGroupDetail *arcadiav1alpha1.FileGroupDetail
	pathMap := make(map[string]*arcadiav1alpha1.FileDetails, 1)
	for i, detail := range kb.Status.FileGroupDetail {
		if detail.Source != nil && detail.Source.Name == group.Source.Name && detail.Source.GetNamespace(kb.GetNamespace()) == ns {
			fileGroupDetail = &kb.Status.FileGroupDetail[i]
			for i, detail := range fileGroupDetail.FileDetails {
				pathMap[detail.Path] = &fileGroupDetail.FileDetails[i]
			}
			break
		}
	}
	if fileGroupDetail == nil {
		// this group is newly added
		log.V(5).Info("new added group, init filegroupdetail status")
		fileGroupDetail = &arcadiav1alpha1.FileGroupDetail{}
		fileGroupDetail.Init(group)
		kb.Status.FileGroupDetail = append(kb.Status.FileGroupDetail, *fileGroupDetail)
		for i, detail := range fileGroupDetail.FileDetails {
			pathMap[detail.Path] = &fileGroupDetail.FileDetails[i]
		}
	}

	errs := make([]error, 0)
	for _, path := range group.Paths {
		r.mu.Lock()
		hasHandled := r.HasHandledSuccessPath[r.hasHandledPathKey(kb, group, path)]
		r.mu.Unlock()
		if hasHandled {
			continue
		}
		fileDetail, ok := pathMap[path]
		if !ok {
			// this path is newly added
			fileGroupDetail.FileDetails = append(fileGroupDetail.FileDetails, arcadiav1alpha1.FileDetails{
				Path:           path,
				Checksum:       "",
				LastUpdateTime: metav1.Now(),
				Phase:          arcadiav1alpha1.FileProcessPhasePending,
				ErrMessage:     "",
			})
			fileDetail = &fileGroupDetail.FileDetails[len(fileGroupDetail.FileDetails)-1]
		}

		switch strings.ToLower(group.Source.Kind) {
		case "versioneddataset":
			// info.Object has been
			info.Object = filepath.Join(vsBasePath, path)
		case "datasource":
			info.Object = path
		}

		stat, err := ds.StatFile(ctx, info)
		log.V(5).Info(fmt.Sprintf("raw StatFile:%#v", stat), "path", path)
		if err != nil {
			errs = append(errs, err)
			fileDetail.UpdateErr(err, arcadiav1alpha1.FileProcessPhaseFailed)
			continue
		}

		objectStat, ok := stat.(minio.ObjectInfo)
		log.V(5).Info(fmt.Sprintf("minio StatFile:%#v", objectStat), "path", path)
		if !ok {
			err = fmt.Errorf("failed to convert stat to minio.ObjectInfo:%s", path)
			errs = append(errs, err)
			fileDetail.UpdateErr(err, arcadiav1alpha1.FileProcessPhaseFailed)
			continue
		}
		if objectStat.ETag == fileDetail.Checksum {
			fileDetail.LastUpdateTime = metav1.Now()
			continue
		}
		fileDetail.Checksum = objectStat.ETag

		tags, err := ds.GetTags(ctx, info)
		if err != nil {
			errs = append(errs, err)
			fileDetail.UpdateErr(err, arcadiav1alpha1.FileProcessPhaseFailed)
			continue
		}

		// File Size in string
		fileDetail.Size = utils.BytesToSizedStr(objectStat.Size)
		// File Type in string
		fileDetail.Type = tags[arcadiav1alpha1.ObjectTypeTag]
		// File data count in string
		fileDetail.Count = tags[arcadiav1alpha1.ObjectCountTag]

		file, err := ds.ReadFile(ctx, info)
		if err != nil {
			errs = append(errs, err)
			fileDetail.UpdateErr(err, arcadiav1alpha1.FileProcessPhaseFailed)
			continue
		}
		defer file.Close()
		startTime := time.Now()
		if err = r.handleFile(ctx, log, file, info.Object, tags, kb, vectorStore, embedder); err != nil {
			if errors.Is(err, errFileSkipped) {
				fileDetail.UpdateErr(err, arcadiav1alpha1.FileProcessPhaseSkipped)
				continue
			}
			err = fmt.Errorf("failed to handle file:%s: %w", path, err)
			errs = append(errs, err)
			fileDetail.UpdateErr(err, arcadiav1alpha1.FileProcessPhaseFailed)
			continue
		}
		// time cost for file process
		fileDetail.TimeCost = int64(time.Since(startTime).Milliseconds())
		r.mu.Lock()
		r.HasHandledSuccessPath[r.hasHandledPathKey(kb, group, path)] = true
		r.mu.Unlock()
		fileDetail.UpdateErr(nil, arcadiav1alpha1.FileProcessPhaseSucceeded)
		log.Info("handle FileGroup succeeded", "timecost(milliseconds)", fileDetail.TimeCost)
	}
	return errors.Join(errs...)
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
	r.cleanupHasHandledSuccessPath(kb)
	r.unready(log, kb)
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

func (r *KnowledgeBaseReconciler) hasHandledPathKey(kb *arcadiav1alpha1.KnowledgeBase, filegroup arcadiav1alpha1.FileGroup, path string) string {
	sourceName := ""
	if filegroup.Source != nil {
		sourceName = filegroup.Source.Name
	}
	return kb.Name + "/" + kb.Namespace + "/" + sourceName + "/" + path
}

func (r *KnowledgeBaseReconciler) cleanupHasHandledSuccessPath(kb *arcadiav1alpha1.KnowledgeBase) {
	r.mu.Lock()
	for _, fg := range kb.Spec.FileGroups {
		for _, path := range fg.Paths {
			delete(r.HasHandledSuccessPath, r.hasHandledPathKey(kb, fg, path))
		}
	}
	r.mu.Unlock()
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
