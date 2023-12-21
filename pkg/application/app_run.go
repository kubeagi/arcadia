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

package application

import (
	"container/list"
	"context"
	"errors"
	"fmt"

	langchaingoschema "github.com/tmc/langchaingo/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
	"k8s.io/utils/strings/slices"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/application/base"
	"github.com/kubeagi/arcadia/pkg/application/chain"
	"github.com/kubeagi/arcadia/pkg/application/llm"
	"github.com/kubeagi/arcadia/pkg/application/prompt"
	"github.com/kubeagi/arcadia/pkg/application/retriever"
)

type Input struct {
	Question string
	// overrideConfig
	NeedStream bool
	History    langchaingoschema.ChatMessageHistory
}
type Output struct {
	Answer     string
	References []retriever.Reference
}

type Application struct {
	Spec          arcadiav1alpha1.ApplicationSpec
	Inited        bool
	Nodes         map[string]base.Node
	StartingNodes []base.Node
	EndingNode    base.Node
}

// var cache = map[string]*Application{}

// func cacheKey(app *arcadiav1alpha1.Application) string {
//	return app.Namespace + "/" + app.Name
//}

func NewAppOrGetFromCache(ctx context.Context, app *arcadiav1alpha1.Application, cli dynamic.Interface) (*Application, error) {
	if app == nil || app.Name == "" || app.Namespace == "" {
		return nil, errors.New("app has no name or namespace")
	}
	// TODO: disable cache for now.
	// https://github.com/kubeagi/arcadia/issues/391
	// a, ok := cache[cacheKey(app)]
	// if !ok {
	//	a = &Application{
	//		Spec: app.Spec,
	//	}
	//	cache[cacheKey(app)] = a
	//	return a, a.Init(ctx, cli)
	// }
	// if reflect.DeepEqual(a.Spec, app.Spec) {
	//	return a, nil
	// }
	a := &Application{
		Spec:   app.Spec,
		Inited: false,
	}
	// a.Spec = app.Spec
	// a.Inited = false
	return a, a.Init(ctx, cli)
}

// todo 防止无限循环，需要找一下是不是成环
func (a *Application) Init(ctx context.Context, cli dynamic.Interface) (err error) {
	if a.Inited {
		return
	}
	a.Nodes = make(map[string]base.Node)

	var inputNodeName, outputNodeName string
	for _, node := range a.Spec.Nodes {
		if node.Ref != nil {
			if node.Ref.Kind == arcadiav1alpha1.OutputNode {
				outputNodeName = node.Name
			} else if node.Ref.Kind == arcadiav1alpha1.InputNode {
				inputNodeName = node.Name
			}
		}
	}

	for _, node := range a.Spec.Nodes {
		n, err := InitNode(ctx, node.Name, *node.Ref, cli)
		if err != nil {
			return fmt.Errorf("initnode %s failed: %v", node.Name, err)
		}
		if err := n.Init(ctx, cli, map[string]any{}); err != nil { // TODO arg
			return fmt.Errorf("node %s init failed: %v", node.Name, err)
		}
		a.Nodes[node.Name] = n
		if node.Name == inputNodeName {
			a.StartingNodes = append(a.StartingNodes, n)
		} else if slices.Contains(node.NextNodeName, outputNodeName) {
			a.EndingNode = n
		}
	}

	for _, node := range a.Spec.Nodes {
		current := a.Nodes[node.Name]
		for _, next := range node.NextNodeName {
			n, ok := a.Nodes[next]
			if !ok {
				return fmt.Errorf("node %s not found", next)
			}
			current.SetNextNode(n)
		}
	}

	for _, current := range a.Nodes {
		for _, next := range current.GetNextNode() {
			next.SetPrevNode(current)
		}
	}

	for _, current := range a.Nodes {
		if len(current.GetPrevNode()) == 0 && current.Name() != inputNodeName {
			a.StartingNodes = append(a.StartingNodes, current)
		}
	}
	klog.Infof("init application success ending node: %#v\n", a.EndingNode)
	return nil
}

func (a *Application) Run(ctx context.Context, cli dynamic.Interface, respStream chan string, input Input) (output Output, err error) {
	out := map[string]any{
		"question":       input.Question,
		"_answer_stream": respStream,
		"_history":       input.History,
	}
	visited := make(map[string]bool)
	waitRunningNodes := list.New()
	for _, v := range a.StartingNodes {
		waitRunningNodes.PushBack(v)
	}
	for e := waitRunningNodes.Front(); e != nil; e = e.Next() {
		e := e.Value.(base.Node)
		if !visited[e.Name()] {
			out["_need_stream"] = false
			if a.EndingNode.Name() == e.Name() && input.NeedStream {
				out["_need_stream"] = true
			}
			if out, err = e.Run(ctx, cli, out); err != nil {
				return Output{}, fmt.Errorf("run node %s: %w", e.Name(), err)
			}
			visited[e.Name()] = true
		}
		for _, n := range e.GetNextNode() {
			waitRunningNodes.PushBack(n)
		}
	}
	if a, ok := out["_answer"]; ok {
		if answer, ok := a.(string); ok && len(answer) > 0 {
			output = Output{Answer: answer}
		}
	}
	if a, ok := out["_references"]; ok {
		if references, ok := a.([]retriever.Reference); ok && len(references) > 0 {
			output.References = references
		}
	}
	if output.Answer == "" && respStream == nil {
		return Output{}, errors.New("no answer")
	}
	return output, nil
}

func InitNode(ctx context.Context, name string, ref arcadiav1alpha1.TypedObjectReference, cli dynamic.Interface) (base.Node, error) {
	baseNode := base.NewBaseNode(name, ref)
	switch baseNode.Group() {
	case "chain":
		switch baseNode.Kind() {
		case "llmchain":
			return chain.NewLLMChain(baseNode), nil
		case "retrievalqachain":
			return chain.NewRetrievalQAChain(baseNode), nil
		default:
			return nil, fmt.Errorf("%s:%v kind is not found", name, ref)
		}
	case "retriever":
		switch baseNode.Kind() {
		case "knowledgebaseretriever":
			return retriever.NewKnowledgeBaseRetriever(ctx, baseNode, cli)
		default:
			return nil, fmt.Errorf("%s:%v kind is not found", name, ref)
		}
	case "":
		switch baseNode.Kind() {
		case "llm":
			return llm.NewLLM(baseNode), nil
		case "input":
			return base.NewInput(baseNode), nil
		case "output":
			return base.NewOutput(baseNode), nil
		default:
			return nil, fmt.Errorf("%s:%v kind is not found", name, ref)
		}
	case "prompt":
		switch baseNode.Kind() {
		case "prompt":
			return prompt.NewPrompt(baseNode), nil
		default:
			return nil, fmt.Errorf("%s:%v kind is not found", name, ref)
		}
	default:
		return nil, fmt.Errorf("%s:%v group is not found", name, ref)
	}
}
