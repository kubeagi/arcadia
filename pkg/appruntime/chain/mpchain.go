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

package chain

import (
	"context"
	"errors"
	"fmt"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/schema"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/pkg/appruntime/base"
)

const (
	// For map-reduce
	DefaultPromptTemplateForMap = `
		Content: {{.context}}

		With above content, please summarize it with only 1/5 size of the content.Please remind that your answer must use same language(中文或English) of the content.
		`
	DefaultPromptTemplatForReduce = `
	Below is the sub-summaries that each is based on a piece of a complete document: 

		{{.context}}

	Please generate a single summary based on above sub-summaries.
	`
)

type MapReduceChain struct {
	// BaseNode for this MapReduceChain
	// Only chain is allowed
	base.BaseNode

	// isReady indicates whether this chain is ready to use
	isReady bool
	// message indicates the detailed message of ready/not ready
	message string

	// MapReduceDocuments used to generate summary
	chains.MapReduceDocuments
	// LLMChain used to
	chains.LLMChain

	// call options against llm
	chainCallOptions []chains.ChainCallOption
}

func NewMapReduceChain(baseNode base.BaseNode, chainCallOptions ...chains.ChainCallOption) *MapReduceChain {
	return &MapReduceChain{
		BaseNode:           baseNode,
		MapReduceDocuments: chains.MapReduceDocuments{},
		chainCallOptions:   chainCallOptions,
	}
}

func (l *MapReduceChain) Init(ctx context.Context, _ client.Client, _ map[string]any) error {
	return nil
}

func (l *MapReduceChain) Run(ctx context.Context, _ client.Client, args map[string]any) (outArgs map[string]any, err error) {
	// initialize the LLM
	v1, ok := args["llm"]
	if !ok {
		return args, errors.New("no llm")
	}
	llm, ok := v1.(llms.Model)
	if !ok {
		return args, errors.New("llm not llms.Model")
	}

	// initialize MapReduceDocuments
	l.MapReduceDocuments = chains.NewMapReduceDocuments(
		chains.NewLLMChain(llm, prompts.NewPromptTemplate(DefaultPromptTemplateForMap, []string{"context"})),
		chains.NewStuffDocuments(
			chains.NewLLMChain(
				llm,
				prompts.NewPromptTemplate(DefaultPromptTemplatForReduce, []string{"context"}),
			),
		),
	)
	// TODO: able to configure this MaxNumberOfConcurrent
	l.MapReduceDocuments.MaxNumberOfConcurrent = 1

	v2, ok := args["documents"]
	if !ok {
		// skip if no documents
		klog.V(5).Infof("skip MapReduceChain due to no documents found")
		return args, nil
	}
	documents, ok := v2.([]schema.Document)
	if !ok {
		// skip if no documents
		klog.V(5).Infof("skip MapReduceChain due to no documents found")
		return args, nil
	}

	// run MapReduceDocuments
	out, err := chains.Run(ctx, l.MapReduceDocuments, documents, l.chainCallOptions...)
	if err != nil {
		return args, fmt.Errorf("failed to run MapReduceChain due to %s", err.Error())
	}
	args["_answer"] = fmt.Sprintf("Here is the document summary: %s \n", out)
	return args, nil
}

func (l *MapReduceChain) Ready() (bool, string) {
	return l.isReady, l.message
}
