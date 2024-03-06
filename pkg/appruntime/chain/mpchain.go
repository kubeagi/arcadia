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
	l.chainCallOptions = chainCallOptions

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

	return nil
}

func (l *MapReduceChain) Run(ctx context.Context, cli client.Client, args map[string]any) (outArgs map[string]any, err error) {
	v1, ok := args["documents"]
	if !ok {
		// skip if no documents
		klog.V(5).Infof("skip MapReduceChain due to no documents found")
		return args, nil
	}
	documents, ok := v1.([]schema.Document)
	if !ok {
		// skip if no documents
		klog.V(5).Infof("skip MapReduceChain due to no documents found")
		return args, nil
	}

	needStream := false
	needStream, ok = args["_need_stream"].(bool)
	if ok && needStream {
		l.chainCallOptions = append(l.chainCallOptions, chains.WithStreamingFunc(stream(args)))
	}

	// run MapReduceDocuments
	out, err := chains.Run(ctx, l.MapReduceDocuments, documents, l.chainCallOptions...)
	if err != nil {
		return args, fmt.Errorf("failed to run MapReduceChain due to %s", err.Error())
	}
	args["_answer"] = out
	return args, nil
}

func (l *MapReduceChain) Ready() (bool, string) {
	return l.isReady, l.message
}
