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
	"k8s.io/client-go/dynamic"

	"github.com/kubeagi/arcadia/pkg/application/base"
)

type LLMChain struct {
	chains.LLMChain
	base.BaseNode
}

func NewLLMChain(baseNode base.BaseNode) *LLMChain {
	return &LLMChain{
		chains.LLMChain{},
		baseNode,
	}
}

func (l *LLMChain) Run(ctx context.Context, _ dynamic.Interface, args map[string]any) (map[string]any, error) {
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
	chain := chains.NewLLMChain(llm, prompt)
	l.LLMChain = *chain
	var out string
	var err error
	if needStream, ok := args["need_stream"].(bool); ok && needStream {
		option := chains.WithStreamingFunc(stream(args))
		out, err = chains.Predict(ctx, l.LLMChain, args, option)
	} else {
		out, err = chains.Predict(ctx, l.LLMChain, args)
	}
	if err == nil {
		args["answer"] = out
	}
	return args, err
}
