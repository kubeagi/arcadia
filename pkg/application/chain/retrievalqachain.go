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

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"

	"github.com/kubeagi/arcadia/pkg/application/base"
	appretriever "github.com/kubeagi/arcadia/pkg/application/retriever"
)

type RetrievalQAChain struct {
	chains.RetrievalQA
	base.BaseNode
}

func NewRetrievalQAChain(baseNode base.BaseNode) *RetrievalQAChain {
	return &RetrievalQAChain{
		chains.RetrievalQA{},
		baseNode,
	}
}

func (l *RetrievalQAChain) Run(ctx context.Context, _ dynamic.Interface, args map[string]any) (map[string]any, error) {
	v1, ok := args["llm"]
	if !ok {
		return args, errors.New("no llm")
	}
	llm, ok := v1.(llms.LanguageModel)
	if !ok {
		return args, errors.New("llm not llms.LanguageModel")
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
	retriever, ok := v3.(schema.Retriever)
	if !ok {
		return args, errors.New("retriever not schema.Retriever")
	}

	llmChain := chains.NewLLMChain(llm, prompt)
	var baseChain chains.Chain
	if _, ok := v3.(*appretriever.KnowledgeBaseRetriever); ok {
		baseChain = appretriever.NewStuffDocuments(llmChain)
	} else {
		baseChain = chains.NewStuffDocuments(llmChain)
	}
	chain := chains.NewRetrievalQA(baseChain, retriever)
	l.RetrievalQA = chain
	args["query"] = args["question"]
	var out string
	var err error
	if needStream, ok := args["need_stream"].(bool); ok && needStream {
		option := chains.WithStreamingFunc(stream(args))
		out, err = chains.Predict(ctx, l.RetrievalQA, args, option)
	} else {
		out, err = chains.Predict(ctx, l.RetrievalQA, args)
	}
	klog.Infof("out:%v, err:%s", out, err)
	if err == nil {
		args["answer"] = out
	}
	return args, err
}
