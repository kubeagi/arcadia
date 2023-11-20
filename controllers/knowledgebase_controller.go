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
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/go-logr/logr"
	"github.com/minio/minio-go/v7"
	"github.com/tmc/langchaingo/documentloaders"
	langchainembeddings "github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores/chroma"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/config"
	"github.com/kubeagi/arcadia/pkg/datasource"
	"github.com/kubeagi/arcadia/pkg/embeddings"
	zhipuaiembeddings "github.com/kubeagi/arcadia/pkg/embeddings/zhipuai"
	"github.com/kubeagi/arcadia/pkg/llms/zhipuai"
)

const (
	waitLonger  = time.Minute
	waitSmaller = time.Second * 3
	waitMedium  = time.Second * 30
)

var (
	errNoDataSource          = fmt.Errorf("no datasource")
	errDataSourceTypeUnkonwn = fmt.Errorf("unknown datasource type")
	errDataSourceNotReady    = fmt.Errorf("datasource is not ready")
	errEmbedderNotReady      = fmt.Errorf("embedder is not ready")
	errVectorStoreNotReady   = fmt.Errorf("vector store is not ready")
)

// KnowledgeBaseReconciler reconciles a KnowledgeBase object
type KnowledgeBaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=knowledgebases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=knowledgebases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=knowledgebases/finalizers,verbs=update
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=embedders,verbs=get;list;watch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=embedders/status,verbs=get
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=datasources,verbs=get;list;watch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=datasources/status,verbs=get
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=vectorstores,verbs=get;list;watch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=vectorstores/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *KnowledgeBaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(1).Info("Start KnowledgeBase Reconcile")
	kb := &arcadiav1alpha1.KnowledgeBase{}
	if err := r.Get(ctx, req.NamespacedName, kb); err != nil {
		// There's no need to requeue if the resource no longer exists.
		// Otherwise, we'll be requeued implicitly because we return an error.
		log.V(1).Info("Failed to get KnowledgeBase")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log = log.WithValues("Generation", kb.GetGeneration(), "ObservedGeneration", kb.Status.ObservedGeneration, "creator", kb.Spec.Creator)
	log.V(1).Info("Get KnowledgeBase instance")

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
		// TODO perform the finalizer operations here, for example: remove vectorstore data?
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
	if updateStatusErr := r.patchStatus(ctx, kb); updateStatusErr != nil {
		log.Error(updateStatusErr, "unable to update status after reconciliation")
		return ctrl.Result{Requeue: true}, updateStatusErr
	}

	return result, err
}

func (r *KnowledgeBaseReconciler) patchStatus(ctx context.Context, kb *arcadiav1alpha1.KnowledgeBase) error {
	latest := &arcadiav1alpha1.KnowledgeBase{}
	if err := r.Client.Get(ctx, client.ObjectKeyFromObject(kb), latest); err != nil {
		return err
	}
	patch := client.MergeFrom(latest.DeepCopy())
	latest.Status = kb.Status
	return r.Client.Status().Patch(ctx, latest, patch, client.FieldOwner("knowledgebase-controller"))
}

// SetupWithManager sets up the controller with the Manager.
func (r *KnowledgeBaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arcadiav1alpha1.KnowledgeBase{}).
		Complete(r)
}

func (r *KnowledgeBaseReconciler) reconcile(ctx context.Context, log logr.Logger, kb *arcadiav1alpha1.KnowledgeBase) (*arcadiav1alpha1.KnowledgeBase, ctrl.Result, error) {
	// Observe generation change
	if kb.Status.ObservedGeneration != kb.Generation {
		kb.Status.ObservedGeneration = kb.Generation
		r.setCondition(kb, kb.InitCondition())
		if updateStatusErr := r.patchStatus(ctx, kb); updateStatusErr != nil {
			log.Error(updateStatusErr, "unable to update status after generation update")
			return kb, ctrl.Result{Requeue: true}, updateStatusErr
		}
	}

	if kb.Status.IsReady() {
		return kb, ctrl.Result{}, nil
	}

	embedderReq := kb.Spec.Embedder
	vectorStoreReq := kb.Spec.VectorStore
	fileGroupsReq := kb.Spec.FileGroups
	if embedderReq == nil || vectorStoreReq == nil || len(fileGroupsReq) == 0 {
		r.setCondition(kb, kb.PendingCondition("emberder or vectorstore or filegroups is not setting"))
		return kb, ctrl.Result{}, nil
	}

	embedder := &arcadiav1alpha1.Embedder{}
	if err := r.Get(ctx, types.NamespacedName{Name: kb.Spec.Embedder.Name, Namespace: kb.Spec.Embedder.GetNamespace()}, embedder); err != nil {
		if errors.IsNotFound(err) {
			r.setCondition(kb, kb.PendingCondition("embedder is not found"))
			return kb, ctrl.Result{RequeueAfter: waitLonger}, nil
		}
		r.setCondition(kb, kb.ErrorCondition(err.Error()))
		return kb, ctrl.Result{}, err
	}

	vectorStore := &arcadiav1alpha1.VectorStore{}
	if err := r.Get(ctx, types.NamespacedName{Name: kb.Spec.VectorStore.Name, Namespace: kb.Spec.VectorStore.GetNamespace()}, vectorStore); err != nil {
		if errors.IsNotFound(err) {
			r.setCondition(kb, kb.PendingCondition("vectorStore is not found"))
			return kb, ctrl.Result{RequeueAfter: waitLonger}, nil
		}
		r.setCondition(kb, kb.ErrorCondition(err.Error()))
		return kb, ctrl.Result{}, err
	}

	errs := make([]error, 0)
	for _, fileGroup := range kb.Spec.FileGroups {
		if err := r.reconcileFileGroup(ctx, log, kb, vectorStore, embedder, fileGroup); err != nil {
			log.Error(err, "Failed to reconcile FileGroup", "fileGroup", fileGroup)
			errs = append(errs, err)
		}
	}
	if err := utilerrors.NewAggregate(errs); err != nil {
		r.setCondition(kb, kb.ErrorCondition(err.Error()))
		return kb, ctrl.Result{RequeueAfter: waitLonger}, nil
	} else {
		for _, fileGroupDetail := range kb.Status.FileGroupDetail {
			for _, fileDetail := range fileGroupDetail.FileDetails {
				if fileDetail.ErrMessage != "" {
					r.setCondition(kb, kb.ErrorCondition(fileDetail.ErrMessage))
					return kb, ctrl.Result{RequeueAfter: waitLonger}, nil
				}
			}
		}
		r.setCondition(kb, kb.ReadyCondition())
	}

	return kb, ctrl.Result{}, nil
}

func (r *KnowledgeBaseReconciler) setCondition(kb *arcadiav1alpha1.KnowledgeBase, condition ...arcadiav1alpha1.Condition) *arcadiav1alpha1.KnowledgeBase {
	kb.Status.SetConditions(condition...)
	return kb
}

func (r *KnowledgeBaseReconciler) reconcileFileGroup(ctx context.Context, log logr.Logger, kb *arcadiav1alpha1.KnowledgeBase, vectorStore *arcadiav1alpha1.VectorStore, embedder *arcadiav1alpha1.Embedder, group arcadiav1alpha1.FileGroup) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to reconcile FileGroup.DataSource: %s: %w", group.Source.Name, err)
		}
	}()

	if group.Source == nil {
		return errNoDataSource
	}
	dataSource := &arcadiav1alpha1.Datasource{}
	ns := group.Source.GetNamespace()
	if err = r.Get(ctx, types.NamespacedName{Name: group.Source.Name, Namespace: ns}, dataSource); err != nil {
		if errors.IsNotFound(err) {
			return errNoDataSource
		} else {
			return err
		}
	}
	if !dataSource.Status.IsReady() {
		return errDataSourceNotReady
	}
	if dataSource.Spec.Type() == arcadiav1alpha1.DatasourceTypeUnknown {
		return errDataSourceTypeUnkonwn
	}

	if len(kb.Status.FileGroupDetail) == 0 {
		kb.Status.FileGroupDetail = make([]arcadiav1alpha1.FileGroupDetail, 1)
		kb.Status.FileGroupDetail[0].Init(group)
	}
	var fileGroupDetail *arcadiav1alpha1.FileGroupDetail
	pathMap := make(map[string]*arcadiav1alpha1.FileDetails, 1)
	for i, detail := range kb.Status.FileGroupDetail {
		if detail.Source != nil && detail.Source.Name == dataSource.Name && detail.Source.GetNamespace() == dataSource.GetNamespace() {
			fileGroupDetail = &kb.Status.FileGroupDetail[i]
			for i, detail := range fileGroupDetail.FileDetails {
				pathMap[detail.Path] = &fileGroupDetail.FileDetails[i] // FIXME 这样对不？
			}
			break
		}
	}
	if fileGroupDetail == nil {
		fileGroupDetail = &arcadiav1alpha1.FileGroupDetail{}
		fileGroupDetail.Init(group)
		kb.Status.FileGroupDetail = append(kb.Status.FileGroupDetail, *fileGroupDetail)
	}

	var ds datasource.Datasource
	info := &arcadiav1alpha1.OSS{}
	switch dataSource.Spec.Type() {
	case arcadiav1alpha1.DatasourceTypeLocal:
		system, err := config.GetSystemDatasource(ctx, r.Client)
		if err != nil {
			return err
		}
		endpoint := system.Spec.Enpoint.DeepCopy()
		if endpoint != nil && endpoint.AuthSecret != nil {
			endpoint.AuthSecret.WithNameSpace(system.Namespace)
		}
		ds, err = datasource.NewLocal(ctx, r.Client, endpoint)
		if err != nil {
			return err
		}
		info = &arcadiav1alpha1.OSS{Bucket: dataSource.Namespace}
	case arcadiav1alpha1.DatasourceTypeOSS:
		endpoint := dataSource.Spec.Enpoint.DeepCopy()
		// set auth secret's namespace to the datasource's namespace
		if endpoint.AuthSecret != nil {
			endpoint.AuthSecret.WithNameSpace(dataSource.Namespace)
		}
		ds, err = datasource.NewOSS(ctx, r.Client, endpoint)
		if err != nil {
			return err
		}
		info = dataSource.Spec.OSS.DeepCopy()
	}
	errs := make([]error, 0)
	for _, path := range group.Paths {
		fileDatail, ok := pathMap[path]
		if !ok {
			fileDatail = &arcadiav1alpha1.FileDetails{
				Path:           path,
				Checksum:       "",
				LastUpdateTime: metav1.Now(),
				Phase:          arcadiav1alpha1.FileProcessPhasePending,
				ErrMessage:     "",
			}
			fileGroupDetail.FileDetails = append(fileGroupDetail.FileDetails, *fileDatail)
		}
		info.Object = path
		stat, err := ds.StatFile(ctx, info)
		log.V(0).Info(fmt.Sprintf("raw StatFile:%#v", stat), "path", path)
		if err != nil {
			errs = append(errs, err)
			fileDatail.UpdateErr(err)
			continue
		}
		switch dataSource.Spec.Type() {
		case arcadiav1alpha1.DatasourceTypeLocal, arcadiav1alpha1.DatasourceTypeOSS:
			objectStat, ok := stat.(minio.ObjectInfo)
			log.V(0).Info(fmt.Sprintf("minio StatFile:%#v", objectStat), "path", path)
			if !ok {
				err = fmt.Errorf("failed to convert stat to minio.ObjectInfo:%s", path)
				errs = append(errs, err)
				fileDatail.UpdateErr(err)
				continue
			}
			if objectStat.ETag == fileDatail.Checksum {
				fileDatail.LastUpdateTime = metav1.Now()
				continue
			}
			fileDatail.Checksum = objectStat.ETag
			tags, err := ds.GetTags(ctx, info)
			if err != nil {
				errs = append(errs, err)
				fileDatail.UpdateErr(err)
				continue
			}
			file, err := ds.ReadFile(ctx, info)
			if err != nil {
				errs = append(errs, err)
				fileDatail.UpdateErr(err)
				continue
			}
			defer file.Close()
			if err = r.handleFile(ctx, log, file, info.Object, tags, kb, vectorStore, embedder); err != nil {
				err = fmt.Errorf("failed to handle file:%s: %w", path, err)
				errs = append(errs, err)
				fileDatail.UpdateErr(err)
				continue
			}
			fileDatail.UpdateErr(nil)
		}
	}
	return utilerrors.NewAggregate(errs)
}

func (r *KnowledgeBaseReconciler) handleFile(ctx context.Context, log logr.Logger, file io.ReadCloser, fileName string, tags map[string]string, kb *arcadiav1alpha1.KnowledgeBase, store *arcadiav1alpha1.VectorStore, embedder *arcadiav1alpha1.Embedder) (err error) {
	if !embedder.Status.IsReady() {
		return errEmbedderNotReady
	}
	if !store.Status.IsReady() {
		return errVectorStoreNotReady
	}
	var em langchainembeddings.Embedder
	switch embedder.Spec.ServiceType { // nolint: gocritic
	case embeddings.ZhiPuAI:
		apiKey, err := embedder.AuthAPIKey(ctx, r.Client)
		if err != nil {
			return err
		}
		em, err = zhipuaiembeddings.NewZhiPuAI(
			zhipuaiembeddings.WithClient(*zhipuai.NewZhiPuAI(apiKey)),
		)
		if err != nil {
			return err
		}
	}
	data, err := io.ReadAll(file) // TODO Load large files in pieces to save memory
	// TODO Line or single line byte exceeds emberder limit
	if err != nil {
		return err
	}
	log.Info("file tags", "tags", tags) // TODO use tags to handle file
	dataReader := bytes.NewReader(data)
	var loader documentloaders.Loader
	switch filepath.Ext(fileName) {
	case "txt":
		loader = documentloaders.NewText(dataReader)
	case "csv":
		loader = documentloaders.NewCSV(dataReader)
	case "html", "htm":
		loader = documentloaders.NewHTML(dataReader)
	default:
		loader = documentloaders.NewText(dataReader)
	}

	// initliaze text splitter
	// var split textsplitter.TextSplitter
	split := textsplitter.NewTokenSplitter(
		textsplitter.WithChunkSize(300),
		textsplitter.WithChunkOverlap(30),
	)
	// TODO tags -> qa or fulltext
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

	documents, err := loader.LoadAndSplit(ctx, split)
	if err != nil {
		return err
	}

	switch store.Spec.Type() { // nolint: gocritic
	case arcadiav1alpha1.VectorStoreTypeChroma:
		s, err := chroma.New(
			chroma.WithChromaURL(store.Spec.Enpoint.URL),
			chroma.WithDistanceFunction(store.Spec.Chroma.DistanceFunction),
			chroma.WithNameSpace(kb.VectorStoreCollectionName()),
			chroma.WithEmbedder(em),
		)
		if err != nil {
			return err
		}
		if err = s.AddDocuments(ctx, documents); err != nil {
			return err
		}
	}
	return nil
}
