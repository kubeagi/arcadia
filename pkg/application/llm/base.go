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
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/application/base"
	"github.com/kubeagi/arcadia/pkg/llms"
)

func NewLLM(ctx context.Context, baseNode base.BaseNode, ref v1alpha1.TypedObjectReference, cli dynamic.Interface) (node base.Node, err error) {
	instance := &v1alpha1.LLM{}
	obj, err := cli.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "llms"}).
		Namespace(ref.GetNamespace()).Get(ctx, ref.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), instance)
	if err != nil {
		return nil, err
	}
	switch instance.Spec.Type {
	case llms.OpenAI:
		// TODO
	case llms.ZhiPuAI:
		node = NewZhipuLLM(baseNode)
	default:
		err = errors.New("llm type is not found")
	}
	return node, err
}
