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

package retriever

import (
	"context"
	"errors"
	"fmt"

	"github.com/tmc/langchaingo/vectorstores"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiretriever "github.com/kubeagi/arcadia/api/app-node/retriever/v1alpha1"
	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/appruntime/base"
	"github.com/kubeagi/arcadia/pkg/appruntime/log"
	"github.com/kubeagi/arcadia/pkg/langchainwrap"
	pkgvectorstore "github.com/kubeagi/arcadia/pkg/vectorstore"
)

type KnowledgeBaseRetriever struct {
	base.BaseNode
	Instance *apiretriever.KnowledgeBaseRetriever
	Finish   func()
}

func NewKnowledgeBaseRetriever(baseNode base.BaseNode) *KnowledgeBaseRetriever {
	return &KnowledgeBaseRetriever{
		BaseNode: baseNode,
	}
}

func (l *KnowledgeBaseRetriever) Init(ctx context.Context, cli client.Client, _ map[string]any) error {
	instance := &apiretriever.KnowledgeBaseRetriever{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: l.RefNamespace(), Name: l.BaseNode.Ref.Name}, instance); err != nil {
		return fmt.Errorf("can't find the retriever in cluster: %w", err)
	}
	l.Instance = instance
	return nil
}

func (l *KnowledgeBaseRetriever) Run(ctx context.Context, cli client.Client, args map[string]any) (map[string]any, error) {
	instance := l.Instance

	var knowledgebaseName, knowledgebaseNamespace string
	for _, n := range l.BaseNode.GetPrevNode() {
		if n.Kind() == "knowledgebase" {
			knowledgebaseName = n.RefName()
			knowledgebaseNamespace = n.RefNamespace()
			break
		}
	}
	if knowledgebaseName == "" || knowledgebaseNamespace == "" {
		return nil, fmt.Errorf("knowledgebase is not setting")
	}

	knowledgebase := &v1alpha1.KnowledgeBase{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: knowledgebaseNamespace, Name: knowledgebaseName}, knowledgebase); err != nil {
		return nil, fmt.Errorf("can't find the knowledgebase in cluster: %w", err)
	}

	embedderReq := knowledgebase.Spec.Embedder
	vectorStoreReq := knowledgebase.Spec.VectorStore
	if embedderReq == nil || vectorStoreReq == nil {
		return nil, fmt.Errorf("knowledgebase %s: embedder or vectorstore or filegroups is not setting", knowledgebaseName)
	}

	embedder := &v1alpha1.Embedder{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: embedderReq.GetNamespace(knowledgebaseNamespace), Name: embedderReq.Name}, embedder); err != nil {
		return nil, fmt.Errorf("can't find the embedder in cluster: %w", err)
	}
	em, err := langchainwrap.GetLangchainEmbedder(ctx, embedder, cli, "")
	if err != nil {
		return nil, fmt.Errorf("can't convert to langchain embedder: %w", err)
	}
	vectorStore := &v1alpha1.VectorStore{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: vectorStoreReq.GetNamespace(knowledgebaseNamespace), Name: vectorStoreReq.Name}, vectorStore); err != nil {
		return nil, fmt.Errorf("can't find the vectorstore in cluster: %w", err)
	}
	var s vectorstores.VectorStore
	s, l.Finish, err = pkgvectorstore.NewVectorStore(ctx, vectorStore, em, knowledgebase.VectorStoreCollectionName(), cli)
	if err != nil {
		return nil, err
	}
	logger := klog.FromContext(ctx)
	logger.V(3).Info(fmt.Sprintf("retriever created[scorethreshold: %f][num: %d]", pointer.Float32Deref(instance.Spec.ScoreThreshold, 0.0), instance.Spec.NumDocuments))
	var retriever vectorstores.Retriever
	if instance.Spec.ScoreThreshold != nil {
		retriever = vectorstores.ToRetriever(s, instance.Spec.NumDocuments, vectorstores.WithScoreThreshold(*instance.Spec.ScoreThreshold))
	} else {
		retriever = vectorstores.ToRetriever(s, instance.Spec.NumDocuments)
	}
	retriever.CallbacksHandler = log.KLogHandler{LogLevel: 3}

	question, ok := args["question"]
	if !ok {
		return nil, errors.New("no question in args")
	}
	query, ok := question.(string)
	if !ok {
		return nil, errors.New("question not string")
	}
	docs, err := retriever.GetRelevantDocuments(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("can't get relevant documents: %w", err)
	}
	if len(docs) == 0 {
		v, exist := args[base.APPDocNullReturn]
		if exist {
			docNullReturn, ok := v.(string)
			if ok && len(docNullReturn) > 0 {
				return nil, &base.RetrieverGetNullDocError{Msg: docNullReturn}
			}
		}
	}
	// pgvector get score means vector distance, similarity = 1 - vector distance
	// chroma get score means similarity
	// we want similarity finally.
	if vectorStore.Spec.Type() == v1alpha1.VectorStoreTypePGVector {
		for i := range docs {
			docs[i].Score = 1 - docs[i].Score
		}
	}
	docs, refs := ConvertDocuments(ctx, docs, "knowledgebase")
	args[base.LangchaingoRetrieverKeyInArg] = &Fakeretriever{Docs: docs}
	AddReferencesToArgs(args, refs)
	return args, nil
}

func (l *KnowledgeBaseRetriever) Ready() (isReady bool, msg string) {
	return l.Instance.Status.IsReadyOrGetReadyMessage()
}

func (l *KnowledgeBaseRetriever) Cleanup() {
	if l.Finish != nil {
		l.Finish()
	}
}
