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

package appruntime

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
	"github.com/kubeagi/arcadia/pkg/appruntime/agent"
	"github.com/kubeagi/arcadia/pkg/appruntime/base"
	"github.com/kubeagi/arcadia/pkg/appruntime/chain"
	"github.com/kubeagi/arcadia/pkg/appruntime/knowledgebase"
	"github.com/kubeagi/arcadia/pkg/appruntime/llm"
	"github.com/kubeagi/arcadia/pkg/appruntime/prompt"
	"github.com/kubeagi/arcadia/pkg/appruntime/retriever"
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
	Namespace     string
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

func NewAppOrGetFromCache(ctx context.Context, cli dynamic.Interface, app *arcadiav1alpha1.Application) (*Application, error) {
	if app == nil || app.Name == "" || app.Namespace == "" {
		return nil, errors.New("app has no name or namespace")
	}
	// make sure namespace value exists in context
	if base.GetAppNamespace(ctx) == "" {
		ctx = base.SetAppNamespace(ctx, app.Namespace)
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
		Namespace: app.GetNamespace(),
		Spec:      app.Spec,
		Inited:    false,
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
		n, err := InitNode(ctx, a.Namespace, node.Name, *node.Ref, cli)
		if err != nil {
			return fmt.Errorf("initnode %s failed: %w", node.Name, err)
		}
		if err := n.Init(ctx, cli, map[string]any{}); err != nil { // TODO arg
			return fmt.Errorf("node %s init failed: %w", node.Name, err)
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
	klog.FromContext(ctx).V(5).Info(fmt.Sprintf("init application success starting nodes: %#v\n", a.StartingNodes))
	return nil
}

func (a *Application) Run(ctx context.Context, cli dynamic.Interface, respStream chan string, input Input) (output Output, err error) {
	// make sure ns value set
	if base.GetAppNamespace(ctx) == "" {
		ctx = base.SetAppNamespace(ctx, a.Namespace)
	}

	out := map[string]any{
		"question":       input.Question,
		"_answer_stream": respStream,
		"_history":       input.History,
		"context":        "",
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
			reWait := false
			for _, n := range e.GetPrevNode() {
				if !visited[n.Name()] {
					reWait = true
					break
				}
			}
			if reWait {
				waitRunningNodes.PushBack(e)
				continue
			}
			if a.EndingNode.Name() == e.Name() && input.NeedStream {
				out["_need_stream"] = true
			}
			klog.FromContext(ctx).V(3).Info(fmt.Sprintf("try to run node:%s", e.Name()))
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

func InitNode(ctx context.Context, appNamespace, name string, ref arcadiav1alpha1.TypedObjectReference, cli dynamic.Interface) (n base.Node, err error) {
	logger := klog.FromContext(ctx)
	defer func() {
		if err != nil {
			logger.Error(err, "initnode failed")
		}
	}()
	baseNode := base.NewBaseNode(appNamespace, name, ref)
	err = fmt.Errorf("unknown kind %s:%v", name, ref)
	switch baseNode.Group() {
	case "chain":
		switch baseNode.Kind() {
		case "llmchain":
			logger.V(3).Info("initnode llmchain")
			return chain.NewLLMChain(baseNode), nil
		case "retrievalqachain":
			logger.V(3).Info("initnode retrievalqachain")
			return chain.NewRetrievalQAChain(baseNode), nil
		case "apichain":
			logger.V(3).Info("initnode llmchain")
			return chain.NewAPIChain(baseNode), nil
		default:
			return nil, err
		}
	case "retriever":
		switch baseNode.Kind() {
		case "knowledgebaseretriever":
			logger.V(3).Info("initnode knowledgebaseretriever")
			return retriever.NewKnowledgeBaseRetriever(baseNode), nil
		default:
			return nil, err
		}
	case "":
		switch baseNode.Kind() {
		case "llm":
			logger.V(3).Info("initnode llm")
			return llm.NewLLM(baseNode), nil
		case "input":
			return base.NewInput(baseNode), nil
		case "output":
			return base.NewOutput(baseNode), nil
		case "knowledgebase":
			logger.V(3).Info("initnode knowledgebase")
			return knowledgebase.NewKnowledgebase(baseNode), nil
		case "agent":
			logger.V(3).Info("initnode agent - executor")
			return agent.NewExecutor(baseNode), nil
		default:
			return nil, err
		}
	case "prompt":
		switch baseNode.Kind() {
		case "prompt":
			logger.V(3).Info("initnode prompt")
			return prompt.NewPrompt(baseNode), nil
		default:
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown group %s:%v", name, ref)
	}
}
