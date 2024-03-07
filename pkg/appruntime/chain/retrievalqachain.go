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

package chain

import (
	"context"
	"errors"
	"fmt"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
	langchainschema "github.com/tmc/langchaingo/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/app-node/chain/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/appruntime/base"
	appretriever "github.com/kubeagi/arcadia/pkg/appruntime/retriever"
)

type RetrievalQAChain struct {
	chains.ConversationalRetrievalQA
	base.BaseNode
	Instance *v1alpha1.RetrievalQAChain
}

func NewRetrievalQAChain(baseNode base.BaseNode) *RetrievalQAChain {
	return &RetrievalQAChain{
		ConversationalRetrievalQA: chains.ConversationalRetrievalQA{},
		BaseNode:                  baseNode,
	}
}

func (l *RetrievalQAChain) Init(ctx context.Context, cli client.Client, _ map[string]any) error {
	instance := &v1alpha1.RetrievalQAChain{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: l.RefNamespace(), Name: l.Ref.Name}, instance); err != nil {
		return fmt.Errorf("can't find the chain in cluster: %w", err)
	}
	l.Instance = instance
	return nil
}

func (l *RetrievalQAChain) Run(ctx context.Context, _ client.Client, args map[string]any) (outArgs map[string]any, err error) {
	v1, ok := args["llm"]
	if !ok {
		return args, errors.New("no llm")
	}
	llm, ok := v1.(llms.Model)
	if !ok {
		return args, errors.New("llm not llms.Model")
	}
	v2, ok := args["prompt"]
	if !ok {
		return args, errors.New("no prompt")
	}
	prompt, ok := v2.(prompts.FormatPrompter)
	if !ok {
		return args, errors.New("prompt not prompts.FormatPrompter")
	}
	v3, ok := args["retriever"]
	if !ok {
		return args, errors.New("no retriever")
	}
	retriever, ok := v3.(langchainschema.Retriever)
	if !ok {
		return args, errors.New("retriever not schema.Retriever")
	}
	v4, ok := args["_history"]
	if !ok {
		return args, errors.New("no history")
	}
	history, ok := v4.(langchainschema.ChatMessageHistory)
	if !ok {
		return args, errors.New("history not memory.ChatMessageHistory")
	}

	instance := l.Instance
	options := GetChainOptions(instance.Spec.CommonChainConfig)

	// Check if have files as input
	v5, ok := args["documents"]
	if ok {
		docs, ok := v5.([]langchainschema.Document)
		if ok && len(docs) != 0 {
			mpChain := NewMapReduceChain(l.BaseNode, options...)
			err = mpChain.Init(ctx, nil, args)
			if err != nil {
				return args, err
			}
			_, err = mpChain.Run(ctx, nil, args)
			if err != nil {
				return args, err
			}
			// TODO:save out as a reference of following answer
		}
	}

	args = runTools(ctx, args, instance.Spec.Tools)
	llmChain := chains.NewLLMChain(llm, prompt)
	if history != nil {
		llmChain.Memory = getMemory(llm, instance.Spec.Memory, history, "", "")
	}
	var baseChain chains.Chain
	var stuffDocuments *appretriever.KnowledgeBaseStuffDocuments
	if knowledgeBaseRetriever, ok := v3.(*appretriever.KnowledgeBaseRetriever); ok {
		stuffDocuments = appretriever.NewStuffDocuments(llmChain, knowledgeBaseRetriever.DocNullReturn)
		baseChain = stuffDocuments
	} else {
		baseChain = chains.NewStuffDocuments(llmChain)
	}
	chain := chains.NewConversationalRetrievalQA(baseChain, chains.LoadCondenseQuestionGenerator(llm), retriever, getMemory(llm, instance.Spec.Memory, history, "", ""))
	l.ConversationalRetrievalQA = chain
	args["query"] = args["question"]
	var out string
	needStream := false
	needStream, ok = args["_need_stream"].(bool)
	if ok && needStream {
		options = append(options, chains.WithStreamingFunc(stream(args)))
		out, err = chains.Predict(ctx, l.ConversationalRetrievalQA, args, options...)
	} else {
		if len(options) > 0 {
			out, err = chains.Predict(ctx, l.ConversationalRetrievalQA, args, options...)
		} else {
			out, err = chains.Predict(ctx, l.ConversationalRetrievalQA, args)
		}
	}
	if stuffDocuments != nil && len(stuffDocuments.References) > 0 {
		args = appretriever.AddReferencesToArgs(args, stuffDocuments.References)
	}
	out, err = handleNoErrNoOut(ctx, needStream, out, err, l.ConversationalRetrievalQA, args, options)
	klog.FromContext(ctx).V(5).Info("use retrievalqachain, blocking out:" + out)
	if err == nil {
		args["_answer"] = out
		return args, nil
	}
	return args, fmt.Errorf("retrievalqachain run error: %w", err)
}

func (l *RetrievalQAChain) Ready() (isReady bool, msg string) {
	return l.Instance.Status.IsReadyOrGetReadyMessage()
}
