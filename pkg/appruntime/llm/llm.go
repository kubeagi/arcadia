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

package llm

import (
	"context"
	"fmt"

	langchainllms "github.com/tmc/langchaingo/llms"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/appruntime/base"
	"github.com/kubeagi/arcadia/pkg/langchainwrap"
)

type LLM struct {
	base.BaseNode
	langchainllms.LLM
	Instance *v1alpha1.LLM
}

func NewLLM(baseNode base.BaseNode) *LLM {
	return &LLM{
		BaseNode: baseNode,
	}
}

func (z *LLM) Init(ctx context.Context, cli client.Client, _ map[string]any) error {
	ns := base.GetAppNamespace(ctx)
	instance := &v1alpha1.LLM{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: z.Ref.GetNamespace(ns), Name: z.Ref.Name}, instance); err != nil {
		return fmt.Errorf("can't find the llm in cluster: %w", err)
	}
	llm, err := langchainwrap.GetLangchainLLM(ctx, instance, cli, "")
	if err != nil {
		return fmt.Errorf("can't convert to langchain llm: %w", err)
	}
	z.LLM = llm
	z.Instance = instance
	return nil
}

func (z *LLM) Run(ctx context.Context, _ client.Client, args map[string]any) (map[string]any, error) {
	args["llm"] = z
	logger := klog.FromContext(ctx)
	ns := base.GetAppNamespace(ctx)
	logger.Info("use llm", "name", z.Ref.Name, "namespace", z.Ref.GetNamespace(ns))
	return args, nil
}

func (z *LLM) Ready() (isReady bool, msg string) {
	return z.Instance.Status.IsReadyOrGetReadyMessage()
}
