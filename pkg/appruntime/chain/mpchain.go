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
	As an expert document summarizer, please provide a concise summary of the following content based on your expertise. Don't worry about the length of the summary:

	Content: {{.context}}

	Please note that your response should be in the same language as the content (English or Chinese).
		`
	DefaultPromptTemplatForReduce = `
	After segmenting the document and generating sub-summaries for each section, it is now time to create a comprehensive summary. Below are the sub-summaries, each based on a specific part of the complete document:

	{{.context}}

	Please generate a cohesive summary that encapsulates the main points from the provided sub-summaries.
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
	v1, ok := args[base.LangchaingoLLMKeyInArg]
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
	v2, ok := args["max_number_of_conccurent"]
	if ok {
		maxNumberOfConcurrent, ok := v2.(int)
		if ok && maxNumberOfConcurrent > 0 {
			l.MapReduceDocuments.MaxNumberOfConcurrent = maxNumberOfConcurrent
		}
	}

	v3, ok := args["documents"]
	if !ok {
		// skip if no documents
		klog.V(5).Infof("skip MapReduceChain due to no documents found")
		return args, nil
	}
	documents, ok := v3.([]schema.Document)
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
	args[base.AgentOutputInArg] = fmt.Sprintf("Here is the document summary: %s \n", out)
	return args, nil
}

func (l *MapReduceChain) Ready() (bool, string) {
	return l.isReady, l.message
}
