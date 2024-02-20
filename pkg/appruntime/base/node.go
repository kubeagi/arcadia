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

package base

import (
	"context"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
)

type Node interface {
	Name() string
	Group() string
	Kind() string
	RefName() string
	RefNamespace() string
	Init(ctx context.Context, cli client.Client, args map[string]any) error
	Run(ctx context.Context, cli client.Client, args map[string]any) (map[string]any, error)
	SetPrevNode(nodes ...Node)
	SetNextNode(nodes ...Node)
	GetPrevNode() []Node
	GetNextNode() []Node
	Cleanup()
}

func NewBaseNode(appNamespace, nodeName string, ref arcadiav1alpha1.TypedObjectReference) BaseNode {
	return BaseNode{
		appNamespace: appNamespace,
		name:         nodeName,
		Ref:          ref,
		prev:         make([]Node, 0),
		next:         make([]Node, 0),
	}
}

type BaseNode struct {
	appNamespace string
	name         string
	Ref          arcadiav1alpha1.TypedObjectReference
	prev         []Node
	next         []Node
}

func (c *BaseNode) Name() string {
	return c.name
}

func (c *BaseNode) Group() string {
	group := c.Ref.APIGroup
	if group == nil {
		return ""
	}
	before, _, _ := strings.Cut(*group, "/")
	if before == "arcadia.kubeagi.k8s.com.cn" {
		return ""
	}
	return strings.ToLower(strings.TrimSuffix(before, ".arcadia.kubeagi.k8s.com.cn"))
}

func (c *BaseNode) Kind() string {
	return strings.ToLower(c.Ref.Kind)
}

func (c *BaseNode) RefName() string {
	return c.Ref.Name
}

func (c *BaseNode) RefNamespace() string {
	return c.Ref.GetNamespace(c.appNamespace)
}
func (c *BaseNode) GetPrevNode() []Node {
	return c.prev
}

func (c *BaseNode) GetNextNode() []Node {
	return c.next
}

func (c *BaseNode) SetPrevNode(nodes ...Node) {
	c.prev = append(c.prev, nodes...)
}

func (c *BaseNode) SetNextNode(nodes ...Node) {
	c.next = append(c.next, nodes...)
}

func (c *BaseNode) Init(_ context.Context, _ client.Client, _ map[string]any) error {
	return nil
}

func (c *BaseNode) Run(_ context.Context, _ client.Client, _ map[string]any) (map[string]any, error) {
	return nil, nil
}

func (c *BaseNode) Cleanup() {
}
