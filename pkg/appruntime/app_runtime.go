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
	"runtime/debug"
	"strings"

	langchaingoschema "github.com/tmc/langchaingo/schema"
	"k8s.io/klog/v2"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/appruntime/agent"
	"github.com/kubeagi/arcadia/pkg/appruntime/base"
	"github.com/kubeagi/arcadia/pkg/appruntime/chain"
	"github.com/kubeagi/arcadia/pkg/appruntime/documentloader"
	"github.com/kubeagi/arcadia/pkg/appruntime/knowledgebase"
	"github.com/kubeagi/arcadia/pkg/appruntime/llm"
	"github.com/kubeagi/arcadia/pkg/appruntime/prompt"
	"github.com/kubeagi/arcadia/pkg/appruntime/retriever"
)

type Input struct {
	// User Query
	Question string
	// Files from user upload
	// normally, the user will upload the files to s3 first and use the name of files here
	Files []string
	// overrideConfig
	NeedStream     bool
	History        langchaingoschema.ChatMessageHistory
	ConversationID string
}
type Output struct {
	Answer     string
	References []retriever.Reference
}

type Application struct {
	Namespace     string
	Name          string
	Spec          arcadiav1alpha1.ApplicationSpec
	Inited        bool
	Nodes         map[string]base.Node
	StartingNodes []base.Node
	EndingNode    base.Node
}

func NewAppOrGetFromCache(ctx context.Context, cli client.Client, app *arcadiav1alpha1.Application) (*Application, error) {
	if app == nil || app.Name == "" || app.Namespace == "" {
		return nil, errors.New("app has no name or namespace")
	}
	a := &Application{
		Namespace: app.GetNamespace(),
		Name:      app.Name,
		Spec:      app.Spec,
		Inited:    false,
	}
	return a, a.Init(ctx, cli)
}

// TODO: 防止无限循环，需要找一下是不是成环
func (a *Application) Init(ctx context.Context, cli client.Client) (err error) {
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
		n, err := InitNode(ctx, a.Namespace, node.Name, *node.Ref)
		if err != nil {
			return fmt.Errorf("initnode %s failed: %w", node.Name, err)
		}
		if err := n.Init(ctx, cli, map[string]any{}); err != nil { // TODO arg
			return fmt.Errorf("%s:%s || node %s init failed: %w", n.Group(), n.Kind(), n.Name(), err)
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

func (a *Application) Run(ctx context.Context, cli client.Client, respStream chan string, input Input) (output Output, err error) {
	out := map[string]any{
		base.InputQuestionKeyInArg:                 input.Question,
		"files":                                    input.Files,
		base.OutputAnswerStreamChanKeyInArg:        respStream,
		base.InputIsNeedStreamKeyInArg:             input.NeedStream,
		base.LangchaingoChatMessageHistoryKeyInArg: input.History,
		// Use an empty context before run
		"context":                "",
		base.ConversationIDInArg: input.ConversationID,
	}
	if a.Spec.DocNullReturn != "" {
		out[base.APPDocNullReturn] = a.Spec.DocNullReturn
	}
	visited := make(map[string]bool)
	waitRunningNodes := list.New()
	for _, v := range a.StartingNodes {
		waitRunningNodes.PushBack(v)
	}
	for e := waitRunningNodes.Front(); e != nil; e = e.Next() {
		e := e.Value.(base.Node)
		if !visited[e.Name()] {
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
			defer func() {
				if r := recover(); r != nil {
					klog.FromContext(ctx).Info(fmt.Sprintf("Recovered from node:%s error:%s stack:%s", e.Name(), r, string(debug.Stack())))
				}
			}()
			defer e.Cleanup()
			if out, err = e.Run(ctx, cli, out); err != nil {
				var er *base.RetrieverGetNullDocError
				if errors.As(err, &er) {
					agentReturnNothing := true
					v, ok := out[base.OutputAnswerKeyInArg]
					if ok {
						if answer, ok := v.(string); ok && len(answer) > 0 {
							agentReturnNothing = false
						}
					}
					if agentReturnNothing {
						if input.NeedStream && respStream != nil {
							go func() {
								respStream <- er.Msg
							}()
						}
						return Output{Answer: er.Msg}, nil
					}
				} else {
					return Output{}, fmt.Errorf("run node %s: %w", e.Name(), err)
				}
			}
			visited[e.Name()] = true
		}
		for _, n := range e.GetNextNode() {
			waitRunningNodes.PushBack(n)
		}
	}
	if a, ok := out[base.OutputAnswerKeyInArg]; ok {
		if answer, ok := a.(string); ok && len(answer) > 0 {
			output = Output{Answer: answer}
		}
	}
	if a, ok := out[base.RuntimeRetrieverReferencesKeyInArg]; ok {
		if references, ok := a.([]retriever.Reference); ok && len(references) > 0 {
			output.References = references
		}
	}
	if output.Answer == "" && respStream == nil {
		return Output{}, errors.New("no answer")
	}
	return output, nil
}

func InitNode(ctx context.Context, appNamespace, name string, ref arcadiav1alpha1.TypedObjectReference) (n base.Node, err error) {
	logger := klog.FromContext(ctx)
	defer func() {
		if err != nil {
			logger.Error(err, "initnode failed")
		}
	}()
	baseNode := base.NewBaseNode(appNamespace, name, ref)
	err = fmt.Errorf("unknown kind %s:%v, get group:%s kind:%s", name, ref, baseNode.Group(), baseNode.Kind())
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
		case "rerankretriever":
			logger.V(3).Info("initnode rerankretriever")
			return retriever.NewRerankRetriever(baseNode), nil
		case "multiqueryretriever":
			logger.V(3).Info("initnode multiqueryretriever")
			return retriever.NewMultiQueryRetriever(baseNode), nil
		case "mergerretriever":
			logger.V(3).Info("initnode mergerretriever")
			return retriever.NewMergerRetriever(baseNode), nil
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
		case "documentloader":
			logger.V(3).Info("initnode agent - documentloader")
			return documentloader.NewDocumentLoader(baseNode), nil
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
		return nil, fmt.Errorf("unknown group %s/%s :%v", baseNode.Group(), baseNode.Kind(), ref)
	}
}

// FindNodesHas group means ref.APIGroup files before `arcadia.kubeagi.k8s.com.cn`
func FindNodesHas(app *arcadiav1alpha1.Application, group, kind string) (has bool, namespace, name string) {
	group, kind = strings.ToLower(group), strings.ToLower(kind)
	for _, n := range app.Spec.Nodes {
		if n.Ref == nil {
			return false, "", ""
		}
		baseNode := base.NewBaseNode(app.Namespace, n.Name, *n.Ref)
		if group == baseNode.Group() && kind == baseNode.Kind() {
			return true, baseNode.RefNamespace(), baseNode.RefName()
		}
	}
	return false, "", ""
}
