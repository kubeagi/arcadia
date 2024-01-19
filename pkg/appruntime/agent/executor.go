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
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"

	"github.com/kubeagi/arcadia/api/app-node/agent/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/appruntime/base"
	"github.com/kubeagi/arcadia/pkg/appruntime/tools/weather"
)

type Executor struct {
	base.BaseNode
}

func NewExecutor(baseNode base.BaseNode) *Executor {
	return &Executor{
		baseNode,
	}
}

func (p *Executor) Run(ctx context.Context, cli dynamic.Interface, args map[string]any) (map[string]any, error) {
	v1, ok := args["llm"]
	if !ok {
		return args, errors.New("no llm")
	}
	llm, ok := v1.(llms.LLM)
	if !ok {
		return args, errors.New("llm not llms.LanguageModel")
	}
	ns := base.GetAppNamespace(ctx)
	instance := &v1alpha1.Agent{}

	obj, err := cli.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "agents"}).
		Namespace(p.Ref.GetNamespace(ns)).Get(ctx, p.Ref.Name, metav1.GetOptions{})
	if err != nil {
		return args, fmt.Errorf("can't find the agent in cluster: %w", err)
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), instance)
	if err != nil {
		return args, fmt.Errorf("can't convert the agent in cluster: %w", err)
	}
	var allowedTools []tools.Tool
	// prepare tools that can be used by this agent
	for _, toolSpec := range instance.Spec.AllowedTools {
		switch toolSpec.Name {
		case weather.ToolName:
			tool, err := weather.New(&toolSpec)
			if err != nil {
				klog.Errorf("failed to create a new weather tool:", err)
				continue
			}
			allowedTools = append(allowedTools, tool)
		default:
			return nil, fmt.Errorf("no tool found with name: %s", toolSpec.Name)
		}
	}

	// Initialize executor using langchaingo
	options := agents.WithMaxIterations(instance.Spec.Options.MaxIterations)
	executor, err := agents.Initialize(llm, allowedTools, agents.ZeroShotReactDescription, options)
	if err != nil {
		return args, fmt.Errorf("failed to initialize executor: %w", err)
	}
	input := make(map[string]any)
	input["input"] = args["question"]
	response, err := executor.Call(ctx, input)
	if err != nil {
		return args, fmt.Errorf("error when call agent: %w", err)
	}
	args["_answer"] = response["output"]
	return args, nil
}
