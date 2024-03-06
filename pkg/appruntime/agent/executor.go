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

package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/llms"
	langchaingoschema "github.com/tmc/langchaingo/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/app-node/agent/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/appruntime/base"
	"github.com/kubeagi/arcadia/pkg/appruntime/chain"
	"github.com/kubeagi/arcadia/pkg/appruntime/log"
	"github.com/kubeagi/arcadia/pkg/appruntime/tools"
)

type Executor struct {
	base.BaseNode
}

func NewExecutor(baseNode base.BaseNode) *Executor {
	return &Executor{
		baseNode,
	}
}

func (p *Executor) Run(ctx context.Context, cli client.Client, args map[string]any) (map[string]any, error) {
	v1, ok := args[base.LangchaingoLLMKeyInArg]
	if !ok {
		return args, errors.New("no llm")
	}
	llm, ok := v1.(llms.Model)
	if !ok {
		return args, errors.New("llm not llms.Model")
	}
	instance := &v1alpha1.Agent{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: p.RefNamespace(), Name: p.Ref.Name}, instance); err != nil {
		return args, fmt.Errorf("can't find the agent in cluster: %w", err)
	}
	allowedTools := tools.InitTools(ctx, instance.Spec.AllowedTools)

	var history langchaingoschema.ChatMessageHistory
	if v3, ok := args[base.LangchaingoChatMessageHistoryKeyInArg]; ok && v3 != nil {
		history, ok = v3.(langchaingoschema.ChatMessageHistory)
		if !ok {
			return args, errors.New("history not memory.ChatMessageHistory")
		}
	}
	// Initialize executor using langchaingo
	executorOptions := func(o *agents.CreationOptions) {
		agents.WithCallbacksHandler(log.KLogHandler{LogLevel: 3})(o)
		agents.WithMaxIterations(instance.Spec.Options.MaxIterations)(o)
		// Only show tool action in the streaming output if configured
		if instance.Spec.Options.ShowToolAction {
			if needStream, ok := args[base.InputIsNeedStreamKeyInArg].(bool); ok && needStream {
				streamHandler := StreamHandler{callbacks.SimpleHandler{}, args}
				agents.WithCallbacksHandler(streamHandler)(o)
			}
		}
		agents.WithMemory(chain.GetMemory(llm, instance.Spec.AgentConfig.Options.Memory, history, "", ""))(o)
	}
	executor, err := agents.Initialize(llm, allowedTools, agents.ZeroShotReactDescription, executorOptions)
	if err != nil {
		return args, fmt.Errorf("failed to initialize executor: %w", err)
	}
	input := make(map[string]any)
	input["input"] = fmt.Sprintf("%s, %s", instance.Spec.Prompt, args["question"])
	response, err := executor.Call(ctx, input)
	if err != nil {
		klog.FromContext(ctx).Error(err, "error when call agent")
		// return args, fmt.Errorf("error when call agent: %w", err)
	}
	klog.FromContext(ctx).V(5).Info("use agent, blocking out:", response["output"])
	args[base.AgentOutputInArg] = response["output"]
	return args, nil
}
