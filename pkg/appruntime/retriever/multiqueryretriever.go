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

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/retrievers"
	langchainschema "github.com/tmc/langchaingo/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiretriever "github.com/kubeagi/arcadia/api/app-node/retriever/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/appruntime/base"
	"github.com/kubeagi/arcadia/pkg/appruntime/log"
)

//nolint:lll
const _defaultQueryTemplate = `You are an AI language model assistant. Your task is to generate 3 different versions of the given user question to retrieve relevant documents from a vector database. 
By generating multiple perspectives on the user question, your goal is to help the user overcome some of the limitations of distance-based similarity search. Provide these alternative questions separated by newlines. 
Original question: {{.question}}`

type MultiQueryRetriever struct {
	base.BaseNode
	Instance *apiretriever.MultiQueryRetriever
}

func NewMultiQueryRetriever(baseNode base.BaseNode) *MultiQueryRetriever {
	return &MultiQueryRetriever{
		BaseNode: baseNode,
	}
}

func (l *MultiQueryRetriever) Init(ctx context.Context, cli client.Client, _ map[string]any) error {
	instance := &apiretriever.MultiQueryRetriever{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: l.RefNamespace(), Name: l.BaseNode.Ref.Name}, instance); err != nil {
		return fmt.Errorf("can't find the rerank retriever in cluster: %w", err)
	}
	l.Instance = instance
	return nil
}

func (l *MultiQueryRetriever) Run(ctx context.Context, cli client.Client, args map[string]any) (map[string]any, error) {
	q, ok := args[base.InputQuestionKeyInArg]
	if !ok {
		return args, errors.New("no question in args")
	}
	query, ok := q.(string)
	if !ok || len(query) == 0 {
		return args, errors.New("empty question")
	}

	retrieversInArg, err := base.GetRetrieversFromArg(args)
	if err != nil {
		if errors.Is(err, base.ErrNoRetrievers) {
			return args, nil
		}
		return args, err
	}

	v2, ok := args[base.LangchaingoLLMKeyInArg]
	if !ok {
		return args, errors.New("no llm")
	}
	llm, ok := v2.(llms.Model)
	if !ok {
		return args, errors.New("llm not llms.Model")
	}
	prompt := prompts.NewPromptTemplate(_defaultQueryTemplate, []string{"question"})
	llmchain := chains.NewLLMChain(llm, prompt, chains.WithCallback(log.KLogHandler{LogLevel: 3}))
	multiqueryRetriever := retrievers.NewMultiQueryRetriever(retrieversInArg[0], llmchain, true)
	multiqueryRetriever.CallbacksHandler = log.KLogHandler{LogLevel: 3}
	docs, err := multiqueryRetriever.GetRelevantDocuments(ctx, query)
	if err != nil {
		return args, err
	}
	newDocs := make([]langchainschema.Document, 0, len(docs))
	for _, doc := range docs {
		if l.Instance.Spec.ScoreThreshold != nil && doc.Score != 0 && doc.Score < *l.Instance.Spec.ScoreThreshold {
			continue
		}
		if l.Instance.Spec.NumDocuments > 0 && len(newDocs) >= l.Instance.Spec.NumDocuments {
			continue
		}
		newDocs = append(newDocs, doc)
	}
	newDocs, newRef := ConvertDocuments(ctx, newDocs, "multiquery")
	// note: the references in args will be replaced, not append
	args[base.RuntimeRetrieverReferencesKeyInArg] = newRef
	args[base.LangchaingoRetrieversKeyInArg] = []langchainschema.Retriever{&Fakeretriever{Docs: newDocs, Name: "MultiqueryRetriever"}}
	return args, nil
}

func (l *MultiQueryRetriever) Ready() (isReady bool, msg string) {
	isReady, msg = l.Instance.Status.IsReadyOrGetReadyMessage()
	if !isReady {
		return isReady, msg
	}
	findRetriever := false
	for _, n := range l.BaseNode.GetPrevNode() {
		if n.Group() == "retriever" {
			findRetriever = true
			break
		}
	}
	if !findRetriever {
		return false, "the multiqueryretiever's prev node should have one retriever"
	}
	return true, ""
}
