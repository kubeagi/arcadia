/*
Copyright 2024 KubeAGI.

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

	langchainretrievers "github.com/tmc/langchaingo/retrievers"
	"github.com/tmc/langchaingo/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiretriever "github.com/kubeagi/arcadia/api/app-node/retriever/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/appruntime/base"
)

type MergerRetriever struct {
	base.BaseNode
	Instance *apiretriever.MergerRetriever
}

func NewMergerRetriever(baseNode base.BaseNode) *MergerRetriever {
	return &MergerRetriever{
		BaseNode: baseNode,
	}
}

func (l *MergerRetriever) Init(ctx context.Context, cli client.Client, _ map[string]any) error {
	instance := &apiretriever.MergerRetriever{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: l.RefNamespace(), Name: l.BaseNode.Ref.Name}, instance); err != nil {
		return fmt.Errorf("can't find the merger retriever in cluster: %w", err)
	}
	l.Instance = instance
	return nil
}

func (l *MergerRetriever) Run(ctx context.Context, _ client.Client, args map[string]any) (map[string]any, error) {
	retrievers, err := base.GetRetrieversFromArg(args)
	if err != nil {
		if errors.Is(err, base.ErrNoRetrievers) {
			return args, nil
		}
		return args, err
	}
	r := langchainretrievers.NewMergerRetriever(retrievers)
	query, err := base.GetInputQuestionFromArg(args)
	if err != nil {
		return args, err
	}
	docs, err := r.GetRelevantDocuments(ctx, query)
	if err != nil {
		return args, err
	}
	docs, refs := ConvertDocuments(ctx, docs, "mergerretriever")
	// note: the references in args will be replaced, not append
	args[base.RuntimeRetrieverReferencesKeyInArg] = refs
	args[base.LangchaingoRetrieversKeyInArg] = []schema.Retriever{&Fakeretriever{Docs: docs, Name: "mergerRetriever"}}
	return args, nil
}

func (l *MergerRetriever) Ready() (isReady bool, msg string) {
	return l.Instance.Status.IsReadyOrGetReadyMessage()
}
