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
	langchaingoschema "github.com/tmc/langchaingo/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/app-node/chain/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/appruntime/base"
)

type LLMChain struct {
	chains.LLMChain
	base.BaseNode
	Instance *v1alpha1.LLMChain
}

func NewLLMChain(baseNode base.BaseNode) *LLMChain {
	return &LLMChain{
		LLMChain: chains.LLMChain{},
		BaseNode: baseNode,
	}
}

func (l *LLMChain) Init(ctx context.Context, cli client.Client, _ map[string]any) error {
	instance := &v1alpha1.LLMChain{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: l.RefNamespace(), Name: l.Ref.Name}, instance); err != nil {
		return fmt.Errorf("can't find the chain in cluster: %w", err)
	}
	l.Instance = instance
	return nil
}

func (l *LLMChain) Run(ctx context.Context, _ client.Client, args map[string]any) (outArgs map[string]any, err error) {
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
	// _history is optional
	// if set ,only ChatMessageHistory allowed
	var history langchaingoschema.ChatMessageHistory
	if v3, ok := args["_history"]; ok && v3 != nil {
		history, ok = v3.(langchaingoschema.ChatMessageHistory)
		if !ok {
			return args, errors.New("history not memory.ChatMessageHistory")
		}
	}
	instance := l.Instance
	options := GetChainOptions(instance.Spec.CommonChainConfig)
	// Add the answer to the context if it's not empty
	if args["_answer"] != nil {
		klog.Infoln("get answer from upstream:", args["_answer"])
		args["context"] = fmt.Sprintf("%s\n%s", args["context"], args["_answer"])
	}
	args = runTools(ctx, args, instance.Spec.Tools)
	chain := chains.NewLLMChain(llm, prompt)
	if history != nil {
		chain.Memory = getMemory(llm, instance.Spec.Memory, history, "", "")
	}
	l.LLMChain = *chain

	var out string
	needStream := false
	needStream, ok = args["_need_stream"].(bool)
	if ok && needStream {
		options = append(options, chains.WithStreamingFunc(stream(args)))
		out, err = chains.Predict(ctx, l.LLMChain, args, options...)
	} else {
		if len(options) > 0 {
			out, err = chains.Predict(ctx, l.LLMChain, args, options...)
		} else {
			out, err = chains.Predict(ctx, l.LLMChain, args)
		}
	}
	out, err = handleNoErrNoOut(ctx, needStream, out, err, l.LLMChain, args, options)
	klog.FromContext(ctx).V(5).Info("use llmchain, blocking out:" + out)
	if err == nil {
		args["_answer"] = out
		return args, nil
	}
	return args, fmt.Errorf("llmchain run error: %w", err)
}

func (l *LLMChain) Ready() (isReady bool, msg string) {
	return l.Instance.Status.IsReadyOrGetReadyMessage()
}
