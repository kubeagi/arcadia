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

package knowledgebase

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/appruntime/base"
)

type Knowledgebase struct {
	base.BaseNode
	Instance *v1alpha1.KnowledgeBase
}

func NewKnowledgebase(baseNode base.BaseNode) *Knowledgebase {
	return &Knowledgebase{
		BaseNode: baseNode,
	}
}

func (k *Knowledgebase) Init(ctx context.Context, cli dynamic.Interface, _ map[string]any) error {
	ns := base.GetAppNamespace(ctx)
	instance := &v1alpha1.KnowledgeBase{}
	obj, err := cli.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "knowledgebases"}).
		Namespace(k.Ref.GetNamespace(ns)).Get(ctx, k.Ref.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("can't find the knowledgebase in cluster: %w", err)
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), instance)
	if err != nil {
		return fmt.Errorf("can't convert the knowledgebase in cluster: %w", err)
	}
	k.Instance = instance
	return nil
}

func (k *Knowledgebase) Run(ctx context.Context, cli dynamic.Interface, args map[string]any) (map[string]any, error) {
	return args, nil
}
