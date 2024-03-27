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
	"fmt"

	"github.com/tmc/langchaingo/vectorstores"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appnode "github.com/kubeagi/arcadia/api/app-node"
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
	var err error
	args, l.Finish, err = GenerateKnowledgebaseRetriever(ctx, cli, knowledgebaseName, knowledgebaseNamespace, l.Instance.Spec.CommonRetrieverConfig, args)
	return args, err
}

func (l *KnowledgeBaseRetriever) Ready() (isReady bool, msg string) {
	isReady, msg = l.Instance.Status.IsReadyOrGetReadyMessage()
	if !isReady {
		return isReady, msg
	}
	var knowledgebaseName, knowledgebaseNamespace string
	for _, n := range l.BaseNode.GetPrevNode() {
		if n.Kind() == "knowledgebase" {
			knowledgebaseName = n.RefName()
			knowledgebaseNamespace = n.RefNamespace()
			break
		}
	}
	if knowledgebaseName == "" || knowledgebaseNamespace == "" {
		return false, "the knowledgebaseretiever's prev node should have one knowledgebase"
	}
	return true, ""
}

func (l *KnowledgeBaseRetriever) Cleanup() {
	if l.Finish != nil {
		l.Finish()
	}
}

func GenerateKnowledgebaseRetriever(ctx context.Context, cli client.Client, knowledgebaseName, knowledgebaseNamespace string, retrieverConfig apiretriever.CommonRetrieverConfig, args map[string]any) (outArg map[string]any, finish func(), err error) {
	knowledgebase := &v1alpha1.KnowledgeBase{}
	isConversationKnowledgebase := appnode.IsPlaceholderConversationKnowledgebase(knowledgebaseName)
	if isConversationKnowledgebase {
		v, ok := args[base.ConversationIDInArg]
		if ok {
			conversationID, ok := v.(string)
			if ok && conversationID != "" {
				knowledgebaseName = conversationID
			}
		}
	}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: knowledgebaseNamespace, Name: knowledgebaseName}, knowledgebase); err != nil {
		if isConversationKnowledgebase && apierrors.IsNotFound(err) { // When there is a conversationID, look for the corresponding conversation knowledgebase. This knowledgebase may not exist. This is not a error
			// TODO We can search for whether there should be a conversation knowledgebase from the pg
			return args, nil, nil
		}
		return nil, nil, fmt.Errorf("can't find the knowledgebase in cluster: %w", err)
	}

	embedderReq := knowledgebase.Spec.Embedder
	vectorStoreReq := knowledgebase.Spec.VectorStore
	if embedderReq == nil || vectorStoreReq == nil {
		return nil, nil, fmt.Errorf("knowledgebase %s: embedder or vectorstore or filegroups is not setting", knowledgebaseName)
	}

	embedder := &v1alpha1.Embedder{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: embedderReq.GetNamespace(knowledgebaseNamespace), Name: embedderReq.Name}, embedder); err != nil {
		return nil, nil, fmt.Errorf("can't find the embedder in cluster: %w", err)
	}
	em, err := langchainwrap.GetLangchainEmbedder(ctx, embedder, cli, "")
	if err != nil {
		return nil, nil, fmt.Errorf("can't convert to langchain embedder: %w", err)
	}
	vectorStore := &v1alpha1.VectorStore{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: vectorStoreReq.GetNamespace(knowledgebaseNamespace), Name: vectorStoreReq.Name}, vectorStore); err != nil {
		return nil, nil, fmt.Errorf("can't find the vectorstore in cluster: %w", err)
	}
	var s vectorstores.VectorStore
	s, finish, err = pkgvectorstore.NewVectorStore(ctx, vectorStore, em, knowledgebase.VectorStoreCollectionName(), cli)
	if err != nil {
		return nil, finish, err
	}
	logger := klog.FromContext(ctx)
	logger.V(3).Info(fmt.Sprintf("retriever created[scorethreshold: %f][num: %d]", pointer.Float32Deref(retrieverConfig.ScoreThreshold, 0.0), retrieverConfig.NumDocuments))
	var retriever vectorstores.Retriever
	if retrieverConfig.ScoreThreshold != nil {
		retriever = vectorstores.ToRetriever(s, retrieverConfig.NumDocuments, vectorstores.WithScoreThreshold(*retrieverConfig.ScoreThreshold))
	} else {
		retriever = vectorstores.ToRetriever(s, retrieverConfig.NumDocuments)
	}
	retriever.CallbacksHandler = log.KLogHandler{LogLevel: 3}

	query, err := base.GetInputQuestionFromArg(args)
	if err != nil {
		return nil, finish, err
	}
	docs, err := retriever.GetRelevantDocuments(ctx, query)
	if err != nil {
		return nil, finish, fmt.Errorf("can't get relevant documents: %w", err)
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
	args = AddReferencesToArgs(args, refs)
	args = base.AddKnowledgebaseRetrieverToArg(args, &Fakeretriever{Docs: docs, Name: "KnowledgebaseRetriever"})
	return args, finish, nil
}
