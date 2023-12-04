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

	langchainembeddings "github.com/tmc/langchaingo/embeddings"
	langchaingoschema "github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/chroma"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	apiretriever "github.com/kubeagi/arcadia/api/app-node/retriever/v1alpha1"
	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/application/base"
	"github.com/kubeagi/arcadia/pkg/embeddings"
	zhipuaiembeddings "github.com/kubeagi/arcadia/pkg/embeddings/zhipuai"
	"github.com/kubeagi/arcadia/pkg/llms/zhipuai"
)

type KnowledgeBaseRetriever struct {
	langchaingoschema.Retriever
	base.BaseNode
}

func NewKnowledgeBaseRetriever(ctx context.Context, baseNode base.BaseNode, cli dynamic.Interface) (*KnowledgeBaseRetriever, error) {
	instance := &apiretriever.KnowledgeBaseRetriever{}
	obj, err := cli.Resource(schema.GroupVersionResource{Group: apiretriever.GroupVersion.Group, Version: apiretriever.GroupVersion.Version, Resource: "knowledgebaseretrievers"}).
		Namespace(baseNode.Ref.GetNamespace()).Get(ctx, baseNode.Ref.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), instance)
	if err != nil {
		return nil, err
	}
	knowledgebaseName := instance.Spec.Input.KnowledgeBaseRef.Name

	knowledgebase := &v1alpha1.KnowledgeBase{}
	obj, err = cli.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "knowledgebases"}).
		Namespace(baseNode.Ref.GetNamespace()).Get(ctx, knowledgebaseName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), knowledgebase)
	if err != nil {
		return nil, err
	}

	embedderReq := knowledgebase.Spec.Embedder
	vectorStoreReq := knowledgebase.Spec.VectorStore
	if embedderReq == nil || vectorStoreReq == nil {
		return nil, fmt.Errorf("knowledgebase %s: embedder or vectorstore or filegroups is not setting", knowledgebaseName)
	}

	embedder := &v1alpha1.Embedder{}
	obj, err = cli.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "embedders"}).
		Namespace(embedderReq.GetNamespace()).Get(ctx, embedderReq.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), embedder)
	if err != nil {
		return nil, err
	}
	var em langchainembeddings.Embedder
	switch embedder.Spec.Type { // nolint: gocritic
	case embeddings.ZhiPuAI:
		apiKey, err := embedder.AuthAPIKeyByDynamicCli(ctx, cli)
		if err != nil {
			return nil, err
		}
		em, err = zhipuaiembeddings.NewZhiPuAI(
			zhipuaiembeddings.WithClient(*zhipuai.NewZhiPuAI(apiKey)),
		)
		if err != nil {
			return nil, err
		}
	}

	vectorStore := &v1alpha1.VectorStore{}
	obj, err = cli.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "vectorstores"}).
		Namespace(vectorStoreReq.GetNamespace()).Get(ctx, vectorStoreReq.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), vectorStore)
	if err != nil {
		return nil, err
	}
	switch vectorStore.Spec.Type() { // nolint: gocritic
	case v1alpha1.VectorStoreTypeChroma:
		s, err := chroma.New(
			chroma.WithChromaURL(vectorStore.Spec.Enpoint.URL),
			chroma.WithDistanceFunction(vectorStore.Spec.Chroma.DistanceFunction),
			chroma.WithNameSpace(knowledgebase.VectorStoreCollectionName()),
			chroma.WithEmbedder(em),
		)
		if err != nil {
			return nil, err
		}
		return &KnowledgeBaseRetriever{
			vectorstores.ToRetriever(s, 5),
			baseNode,
		}, nil
	default:
		return nil, fmt.Errorf("unknown vectorstore type: %s", vectorStore.Spec.Type())
	}
}

func (l *KnowledgeBaseRetriever) Run(ctx context.Context, _ dynamic.Interface, args map[string]any) (map[string]any, error) {
	args["retriever"] = l
	return args, nil
}
