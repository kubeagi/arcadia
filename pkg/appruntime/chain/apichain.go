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
	"net/http"

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

type APIChain struct {
	chains.APIChain
	base.BaseNode
	Instance *v1alpha1.APIChain
}

func NewAPIChain(baseNode base.BaseNode) *APIChain {
	return &APIChain{
		APIChain: chains.APIChain{},
		BaseNode: baseNode,
	}
}

func (l *APIChain) Init(ctx context.Context, cli client.Client, _ map[string]any) error {
	instance := &v1alpha1.APIChain{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: l.RefNamespace(), Name: l.Ref.Name}, instance); err != nil {
		return fmt.Errorf("can't find the chain in cluster: %w", err)
	}
	l.Instance = instance
	return nil
}

func (l *APIChain) Run(ctx context.Context, _ client.Client, args map[string]any) (map[string]any, error) {
	v1, ok := args[base.LangchaingoLLMKeyInArg]
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
	p, err := prompt.FormatPrompt(args)
	if err != nil {
		return args, fmt.Errorf("can't format prompt: %w", err)
	}
	args["input"] = p.String()
	v3, ok := args[base.LangchaingoChatMessageHistoryKeyInArg]
	if !ok {
		return args, errors.New("no history")
	}
	history, ok := v3.(langchaingoschema.ChatMessageHistory)
	if !ok {
		return args, errors.New("history not memory.ChatMessageHistory")
	}

	instance := l.Instance
	options := GetChainOptions(instance.Spec.CommonChainConfig)

	chain := chains.NewAPIChain(llm, http.DefaultClient)
	chain.RequestChain.Memory = getMemory(llm, instance.Spec.Memory, history, "", "")
	chain.AnswerChain.Memory = getMemory(llm, instance.Spec.Memory, history, "input", "")
	l.APIChain = chain
	apiDoc := instance.Spec.APIDoc
	if apiDoc == "" {
		return args, errors.New("no apidoc in apichain")
	}
	args["api_docs"] = apiDoc
	var out string
	needStream := false
	needStream, ok = args[base.InputIsNeedStreamKeyInArg].(bool)
	if ok && needStream {
		options = append(options, chains.WithStreamingFunc(stream(args)))
		out, err = chains.Predict(ctx, l.APIChain, args, options...)
	} else {
		if len(options) > 0 {
			out, err = chains.Predict(ctx, l.APIChain, args, options...)
		} else {
			out, err = chains.Predict(ctx, l.APIChain, args)
		}
	}
	out, err = handleNoErrNoOut(ctx, needStream, out, err, l.APIChain, args, options)
	klog.FromContext(ctx).V(5).Info("use apichain, blocking out:" + out)
	if err == nil {
		args[base.OutputAnserKeyInArg] = out
		return args, nil
	}
	return args, fmt.Errorf("apichain run error: %w", err)
}

func (l *APIChain) Ready() (isReady bool, msg string) {
	return l.Instance.Status.IsReadyOrGetReadyMessage()
}
