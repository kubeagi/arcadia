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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/application/base"
	"github.com/kubeagi/arcadia/pkg/langchainwrap"
)

type LLM struct {
	base.BaseNode
	langchainllms.LanguageModel
}

func NewLLM(baseNode base.BaseNode) *LLM {
	return &LLM{
		BaseNode: baseNode,
	}
}

func (z *LLM) Init(ctx context.Context, cli dynamic.Interface, args map[string]any) error {
	ns := base.GetAppNamespace(ctx)
	instance := &v1alpha1.LLM{}
	obj, err := cli.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "llms"}).
		Namespace(z.Ref.GetNamespace(ns)).Get(ctx, z.Ref.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("can't find the llm in cluster: %w", err)
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), instance)
	if err != nil {
		return fmt.Errorf("can't convert the llm in cluster: %w", err)
	}
	llm, err := langchainwrap.GetLangchainLLM(ctx, instance, nil, cli)
	if err != nil {
		return fmt.Errorf("can't convert to langchain llm: %w", err)
	}
	z.LanguageModel = llm
	return nil
}

func (z *LLM) Run(_ context.Context, _ dynamic.Interface, args map[string]any) (map[string]any, error) {
	args["llm"] = z
	return args, nil
}
