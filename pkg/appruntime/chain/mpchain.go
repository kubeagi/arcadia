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
		{{.context}}

		With above content, please summarize it with only half content size of it.
		`
	DefaultPromptTemplatForReduce = `"{{.context}}"`

	// For post process the map-reduced summary
	DefaultPromptTemplateForPostMapReduce = `
		Here is the map-reduced summary of a document:

		Summary: {{.summary}}

		Now please answer the following question based on the above document summary. Make sure the answer is using same language with the question:

		Question: {{.question}}

		Answer:
	`

	DefaultSummaryMaxNumberOfConcurrent = 2
	DefaultDocumentChunkSize            = 1024
	DefaultDocumentChunkOverlap         = 100
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

func NewMapReduceChain(baseNode base.BaseNode) *MapReduceChain {
	return &MapReduceChain{
		BaseNode:           baseNode,
		MapReduceDocuments: chains.MapReduceDocuments{},
	}
}

func (l *MapReduceChain) Init(ctx context.Context, cli client.Client, args map[string]any) error {
	if args == nil {
		return errors.New("no arguments provided for MapReduceChain")
	}
	// initialize the LLM
	v1, ok := args["llm"]
	if !ok {
		return errors.New("no llm")
	}
	llm, ok := v1.(llms.Model)
	if !ok {
		return errors.New("llm not llms.Model")
	}

	// only group `chain` is allowed
	if l.BaseNode.Group() != "chain" {
		return fmt.Errorf("invalid base node with group %s.must be in group chain", l.BaseNode.Group())
	}
	// initialize call options
	var chainCallOptions []chains.ChainCallOption
	switch kind := l.BaseNode.Kind(); kind {
	case "llmchain":
		llmchain := NewLLMChain(l.BaseNode)
		if err := llmchain.Init(ctx, cli, nil); err != nil {
			return err
		}
		l.isReady, l.message = llmchain.Ready()
		if !l.isReady {
			return fmt.Errorf("llmchain is not ready with %s", l.message)
		}
		chainCallOptions = GetChainOptions(llmchain.Instance.Spec.CommonChainConfig)
	case "retrievalqachain":
		retrivalQAChain := NewRetrievalQAChain(l.BaseNode)
		if err := retrivalQAChain.Init(ctx, cli, nil); err != nil {
			return err
		}
		l.isReady, l.message = retrivalQAChain.Ready()
		if !l.isReady {
			return fmt.Errorf("retrivalQAChain is not ready with %s", l.message)
		}
		chainCallOptions = GetChainOptions(retrivalQAChain.Instance.Spec.CommonChainConfig)
	default:
		return fmt.Errorf("invalid base node kind %s for MapReduceChain.not supported yet", kind)
	}
	l.chainCallOptions = append(l.chainCallOptions, chainCallOptions...)

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

	l.LLMChain = *chains.NewLLMChain(llm, prompts.NewPromptTemplate(DefaultPromptTemplateForPostMapReduce, []string{"summary", "question"}))

	return nil
}

func (l *MapReduceChain) Run(ctx context.Context, cli client.Client, args map[string]any) (outArgs map[string]any, err error) {
	v1, ok := args["documents"]
	if !ok {
		return args, errors.New("no documents")
	}
	documents, ok := v1.([]schema.Document)
	if !ok {
		return args, errors.New("llm not llms.LanguageModel")
	}
	// run MapReduceDocuments
	out, err := chains.Run(ctx, l.MapReduceDocuments, documents, l.chainCallOptions...)
	if err != nil {
		return args, fmt.Errorf("failed to run MapReduceChain due to %s", err.Error())
	}
	// set the summary with the output of MapReduceDocuments
	args["summary"] = out

	// run LLMChain
	needStream := false
	needStream, ok = args["_need_stream"].(bool)
	if ok && needStream {
		l.chainCallOptions = append(l.chainCallOptions, chains.WithStreamingFunc(stream(args)))
	}
	// call llmchain
	out, err = chains.Predict(ctx, l.LLMChain, args, l.chainCallOptions...)
	// handler out & error
	out, err = handleNoErrNoOut(ctx, needStream, out, err, l.LLMChain, args, l.chainCallOptions)
	klog.FromContext(ctx).V(5).Info("use MapReduceChain, blocking out:" + out)
	if err == nil {
		args["_answer"] = out
		return args, nil
	}
	return args, fmt.Errorf("mapreaducechain run error: %w", err)
}

func (l *MapReduceChain) Ready() (bool, string) {
	return l.isReady, l.message
}
