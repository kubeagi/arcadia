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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sort"

	langchainschema "github.com/tmc/langchaingo/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiretriever "github.com/kubeagi/arcadia/api/app-node/retriever/v1alpha1"
	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/appruntime/base"
)

type RerankRetriever struct {
	base.BaseNode
	Instance *apiretriever.RerankRetriever
}

func NewRerankRetriever(baseNode base.BaseNode) *RerankRetriever {
	return &RerankRetriever{
		BaseNode: baseNode,
	}
}

func (l *RerankRetriever) Init(ctx context.Context, cli client.Client, _ map[string]any) error {
	instance := &apiretriever.RerankRetriever{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: l.RefNamespace(), Name: l.BaseNode.Ref.Name}, instance); err != nil {
		return fmt.Errorf("can't find the rerank retriever in cluster: %w", err)
	}
	l.Instance = instance
	return nil
}

func (l *RerankRetriever) Run(ctx context.Context, cli client.Client, args map[string]any) (map[string]any, error) {
	refs, ok := args[base.RuntimeRetrieverReferencesKeyInArg]
	if !ok {
		return args, errors.New("no refs in args")
	}
	references, ok := refs.([]Reference)
	if !ok {
		return args, errors.New("empty references")
	}
	if len(references) == 0 {
		klog.FromContext(ctx).V(3).Info("rerank retriever get no references, skip rerank")
		args[base.LangchaingoRetrieversKeyInArg] = []langchainschema.Retriever{&Fakeretriever{Docs: nil, Name: "RerankRetriever"}}
		return args, nil
	}
	q, ok := args[base.InputQuestionKeyInArg]
	if !ok {
		return args, errors.New("no question in args")
	}
	query, ok := q.(string)
	if !ok || len(query) == 0 {
		return args, errors.New("empty question")
	}
	body := RerankRequestBody{
		Query:    query,
		Passages: make([]string, len(references)),
	}
	for i := range references {
		// first, use the question (and answer, if it has) as the passage
		if references[i].Question != "" {
			body.Passages[i] = references[i].Question
			if references[i].Answer != "" {
				body.Passages[i] += "\n" + references[i].Answer
			}
		} else {
			// second,  use the raw content as the passage
			body.Passages[i] = references[i].Content
		}
	}
	reqBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("request json marshal failed: %w", err)
	}
	URL := fmt.Sprintf("http://%s-worker.%s.svc:%d/api/v1/reranking", l.Instance.Spec.Model.Name, l.Instance.Spec.Model.GetNamespace(l.RefNamespace()), arcadiav1alpha1.DefaultWorkerPort)
	klog.FromContext(ctx).V(5).Info(fmt.Sprintf("send req to rerank, url:%s, body:%s", URL, string(reqBytes)))
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, URL, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("get resp err: %w", err)
	}
	defer response.Body.Close()

	code := response.StatusCode
	resp := make([]float32, 0)
	if err := json.NewDecoder(response.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("parse json resp get err:%w, http status code:%d", err, code)
	}
	klog.FromContext(ctx).V(5).Info(fmt.Sprintf("get resp :%#v", resp))

	for i := range references {
		references[i].RerankScore = resp[i]
	}
	sort.Slice(references, func(i, j int) bool {
		return references[i].RerankScore > references[j].RerankScore
	})
	newRef := make([]Reference, 0, len(references))
	for i := range references {
		if l.Instance.Spec.ScoreThreshold != nil && references[i].RerankScore < *l.Instance.Spec.ScoreThreshold {
			break
		}
		if l.Instance.Spec.NumDocuments > 0 && len(newRef) >= l.Instance.Spec.NumDocuments {
			break
		}
		newRef = append(newRef, references[i])
	}
	// note: the references in args will be replaced, not append
	args[base.RuntimeRetrieverReferencesKeyInArg] = newRef

	retrievers, err := base.GetRetrieversFromArg(args)
	if err != nil {
		if errors.Is(err, base.ErrNoRetrievers) {
			return args, nil
		}
		return args, err
	}
	docs, err := retrievers[0].GetRelevantDocuments(ctx, query)
	if err != nil {
		return args, fmt.Errorf("get relevant documents failed: %w", err)
	}
	newDocs := make([]langchainschema.Document, 0, len(docs))
	for i := range newRef {
		for j := range docs {
			if newRef[i].Score == docs[j].Score && reflect.DeepEqual(newRef[i].Metadata, docs[j].Metadata) {
				if v := docs[j].Metadata; len(v) == 0 {
					docs[j].Metadata = make(map[string]any, 1)
				}
				docs[j].Metadata[RerankScoreCol] = newRef[i].RerankScore
				newDocs = append(newDocs, docs[j])
			}
		}
	}
	args[base.LangchaingoRetrieversKeyInArg] = []langchainschema.Retriever{&Fakeretriever{Docs: newDocs, Name: "RerankRetriever"}}
	return args, nil
}

func (l *RerankRetriever) Ready() (isReady bool, msg string) {
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

type RerankRequestBody struct {
	Query    string   `json:"question"`
	Passages []string `json:"answers"`
}
