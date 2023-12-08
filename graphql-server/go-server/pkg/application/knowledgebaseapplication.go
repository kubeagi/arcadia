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
	"context"
	"reflect"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/pointer"

	node "github.com/kubeagi/arcadia/api/app-node"
	apichain "github.com/kubeagi/arcadia/api/app-node/chain/v1alpha1"
	apiprompt "github.com/kubeagi/arcadia/api/app-node/prompt/v1alpha1"
	apiretriever "github.com/kubeagi/arcadia/api/app-node/retriever/v1alpha1"
	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/graphql-server/go-server/graph/generated"
	"github.com/kubeagi/arcadia/graphql-server/go-server/pkg/common"
	"github.com/kubeagi/arcadia/graphql-server/go-server/pkg/utils"
)

func cr2model(objPrompt, objChain, objRetriever, objApp *unstructured.Unstructured) (*generated.KnowledgeBaseApplication, error) {
	prompt := &apiprompt.Prompt{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(objPrompt.UnstructuredContent(), prompt); err != nil {
		return nil, err
	}

	chain := &apichain.RetrievalQAChain{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(objChain.UnstructuredContent(), chain); err != nil {
		return nil, err
	}

	retriever := &apiretriever.KnowledgeBaseRetriever{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(objRetriever.UnstructuredContent(), retriever); err != nil {
		return nil, err
	}

	app := &v1alpha1.Application{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(objApp.UnstructuredContent(), app); err != nil {
		return nil, err
	}
	condition := app.Status.GetCondition(v1alpha1.TypeReady)
	UpdateTimestamp := &condition.LastTransitionTime.Time
	status := string(condition.Status)

	return &generated.KnowledgeBaseApplication{
		Name:              app.Name,
		Namespace:         app.Namespace,
		Labels:            utils.MapStr2Any(app.Labels),
		Annotations:       utils.MapStr2Any(app.Annotations),
		Creator:           pointer.String(app.Spec.Creator),
		DisplayName:       pointer.String(app.Spec.DisplayName),
		Description:       pointer.String(app.Spec.Description),
		CreationTimestamp: &app.CreationTimestamp.Time,
		UpdateTimestamp:   UpdateTimestamp,
		Icon:              pointer.String(app.Spec.Icon),
		IsPublic:          pointer.Bool(app.Spec.IsPublic),
		KnowledgebaseName: retriever.Spec.Input.Name,
		LlmName:           chain.Spec.Input.LLM.Name,
		SystemMessage:     pointer.String(prompt.Spec.SystemMessage),
		UserMessage:       pointer.String(prompt.Spec.UserMessage),
		Status:            pointer.String(status),
	}, nil
}

func CreateKnowledgeBaseApplication(ctx context.Context, c dynamic.Interface, input generated.CreateKnowledgeBaseApplicationInput) (*generated.KnowledgeBaseApplication, error) {
	prompt := apiprompt.Prompt{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
		},
		Spec: apiprompt.PromptSpec{
			CommonPromptConfig: apiprompt.CommonPromptConfig{
				SystemMessage: pointer.StringDeref(input.SystemMessage, ""),
				UserMessage:   pointer.StringDeref(input.UserMessage, ""),
			},
			Input: apiprompt.Input{
				CommonOrInPutOrOutputRef: node.CommonOrInPutOrOutputRef{
					Kind: "Input",
					Name: "Input",
				},
			},
			Output: apiprompt.Output{
				CommonOrInPutOrOutputRef: node.CommonOrInPutOrOutputRef{
					APIGroup: pointer.String("chain.arcadia.kubeagi.k8s.com.cn"),
					Kind:     "RetrievalQAChain",
					Name:     input.Name,
				},
			},
		},
	}
	object, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&prompt)
	if err != nil {
		return nil, err
	}
	objPrompt, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Prompt")).Namespace(input.Namespace).Create(ctx, &unstructured.Unstructured{Object: object}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	chain := apichain.RetrievalQAChain{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
		},
		Spec: apichain.RetrievalQAChainSpec{
			CommonChainConfig: apichain.CommonChainConfig{
				Memory: apichain.Memory{
					MaxTokenLimit: 204800,
				},
			},
			Input: apichain.RetrievalQAChainInput{
				LLMChainInput: apichain.LLMChainInput{
					LLM: node.LLMRef{
						Kind:     "LLM",
						Name:     input.LlmName,
						APIGroup: "arcadia.kubeagi.k8s.com.cn",
					},
					Prompt: node.PromptRef{
						CommonRef: node.CommonRef{
							Kind: "Prompt",
							Name: input.Name,
						},
						APIGroup: "prompt.arcadia.kubeagi.k8s.com.cn",
					},
				},
				Retriever: node.RetrieverRef{
					CommonRef: node.CommonRef{
						Kind: "KnowledgeBaseRetriever",
						Name: input.Name,
					},
					APIGroup: "retriever.arcadia.kubeagi.k8s.com.cn",
				},
			},
			Output: apichain.Output{
				CommonOrInPutOrOutputRef: node.CommonOrInPutOrOutputRef{
					Kind: "Output",
					Name: "Output",
				},
			},
		},
	}
	object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(&chain)
	if err != nil {
		return nil, err
	}
	objChain, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Chain")).Namespace(input.Namespace).Create(ctx, &unstructured.Unstructured{Object: object}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	retriever := apiretriever.KnowledgeBaseRetriever{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
		},
		Spec: apiretriever.KnowledgeBaseRetrieverSpec{
			Input: apiretriever.Input{
				KnowledgeBaseRef: node.KnowledgeBaseRef{
					Kind:     "KnowledgeBase",
					Name:     input.KnowledgebaseName,
					APIGroup: "arcadia.kubeagi.k8s.com.cn",
				},
			},
			Output: apiretriever.Output{
				CommonOrInPutOrOutputRef: node.CommonOrInPutOrOutputRef{
					APIGroup: pointer.String("chain.arcadia.kubeagi.k8s.com.cn"),
					Kind:     "RetrievalQAChain",
					Name:     input.Name,
				},
			},
		},
	}
	object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(&retriever)
	if err != nil {
		return nil, err
	}
	objRetriever, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Retrievers")).Namespace(input.Namespace).Create(ctx, &unstructured.Unstructured{Object: object}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	app := &v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:        input.Name,
			Namespace:   input.Namespace,
			Labels:      utils.MapAny2Str(input.Labels),
			Annotations: utils.MapAny2Str(input.Annotations),
		},
		Spec: v1alpha1.ApplicationSpec{
			CommonSpec: v1alpha1.CommonSpec{
				DisplayName: pointer.StringDeref(input.DisplayName, ""),
				Description: pointer.StringPtrDerefOr(input.Description, ""),
			},
			Icon:     pointer.StringPtrDerefOr(input.Icon, ""),
			IsPublic: pointer.BoolDeref(input.IsPublic, false),
			Prologue: "Welcome to talk to the KnowledgeBase!🤖",
			Nodes: []v1alpha1.Node{
				{
					NodeConfig: v1alpha1.NodeConfig{
						Name:        "Input",
						DisplayName: "用户输入",
						Description: "用户输入节点，必须",
						Ref: &v1alpha1.TypedObjectReference{
							Kind: "Input",
							Name: "Input",
						},
					},
					NextNodeName: []string{"prompt-node"},
				},
				{
					NodeConfig: v1alpha1.NodeConfig{
						Name:        "prompt-node",
						DisplayName: "prompt",
						Description: "设定prompt，template中可以使用{{.}}来替换变量",
						Ref: &v1alpha1.TypedObjectReference{
							APIGroup: pointer.String("prompt.arcadia.kubeagi.k8s.com.cn"),
							Kind:     "Prompt",
							Name:     input.Name,
						},
					},
					NextNodeName: []string{"chain-node"},
				},
				{
					NodeConfig: v1alpha1.NodeConfig{
						Name:        "llm-node",
						DisplayName: "llm",
						Description: "设定大模型的访问信息",
						Ref: &v1alpha1.TypedObjectReference{
							APIGroup: pointer.String("arcadia.kubeagi.k8s.com.cn"),
							Kind:     "LLM",
							Name:     input.LlmName,
						},
					},
					NextNodeName: []string{"chain-node"},
				},
				{
					NodeConfig: v1alpha1.NodeConfig{
						Name:        "retriever-node",
						DisplayName: "从知识库提取信息的retriever",
						Description: "连接应用和知识库",
						Ref: &v1alpha1.TypedObjectReference{
							APIGroup: pointer.String("retriever.arcadia.kubeagi.k8s.com.cn"),
							Kind:     "KnowledgeBaseRetriever",
							Name:     input.Name,
						},
					},
					NextNodeName: []string{"chain-node"},
				},
				{
					NodeConfig: v1alpha1.NodeConfig{
						Name:        "chain-node",
						DisplayName: "RetrievalQA chain",
						Description: "chain是langchain的核心概念RetrievalQAChain用于从retriver中提取信息，供llm调用",
						Ref: &v1alpha1.TypedObjectReference{
							APIGroup: pointer.String("retriever.arcadia.kubeagi.k8s.com.cn"),
							Kind:     "KnowledgeBaseRetriever",
							Name:     input.Name,
						},
					},
					NextNodeName: []string{"Output"},
				},
				{
					NodeConfig: v1alpha1.NodeConfig{
						Name:        "Output",
						DisplayName: "最终输出",
						Description: "最终输出节点，必须",
						Ref: &v1alpha1.TypedObjectReference{
							Kind: "Output",
							Name: "Output",
						},
					},
				},
			},
		},
	}
	if app.Labels == nil {
		app.Labels = make(map[string]string)
	}
	app.Labels[v1alpha1.ApplicationTypeLabel] = v1alpha1.KnowledgeBaseApplicationType
	object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(&app)
	if err != nil {
		return nil, err
	}
	objApp, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Application")).Namespace(input.Namespace).Create(ctx, &unstructured.Unstructured{Object: object}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return cr2model(objPrompt, objChain, objRetriever, objApp)
}

func UpdateKnowledgeBaseApplication(ctx context.Context, c dynamic.Interface, input generated.UpdateKnowledgeBaseApplicationInput) (*generated.KnowledgeBaseApplication, error) {
	// 1. get old resource
	objPrompt, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Prompt")).Namespace(input.Namespace).Get(ctx, input.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	objChain, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Chain")).Namespace(input.Namespace).Get(ctx, input.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	objRetriever, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Retrievers")).Namespace(input.Namespace).Get(ctx, input.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	objApp, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Application")).Namespace(input.Namespace).Get(ctx, input.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// 2. compare and update
	prompt := &apiprompt.Prompt{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(objPrompt.UnstructuredContent(), prompt); err != nil {
		return nil, err
	}
	if (input.SystemMessage != nil && *input.SystemMessage != prompt.Spec.SystemMessage) || (input.UserMessage != nil && *input.UserMessage != prompt.Spec.UserMessage) {
		prompt.Spec.SystemMessage = pointer.StringDeref(input.SystemMessage, prompt.Spec.SystemMessage)
		prompt.Spec.UserMessage = pointer.StringDeref(input.UserMessage, prompt.Spec.UserMessage)
		object, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&prompt)
		if err != nil {
			return nil, err
		}
		objPrompt, err = c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Prompt")).Namespace(input.Namespace).Update(ctx, &unstructured.Unstructured{Object: object}, metav1.UpdateOptions{})
		if err != nil {
			return nil, err
		}
	}

	retriever := apiretriever.KnowledgeBaseRetriever{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(objRetriever.UnstructuredContent(), retriever); err != nil {
		return nil, err
	}
	if input.KnowledgebaseName != "" && input.KnowledgebaseName != retriever.Spec.Input.KnowledgeBaseRef.Name {
		retriever.Spec.Input.KnowledgeBaseRef.Name = input.KnowledgebaseName
		object, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&retriever)
		if err != nil {
			return nil, err
		}
		objRetriever, err = c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Retrievers")).Namespace(input.Namespace).Update(ctx, &unstructured.Unstructured{Object: object}, metav1.UpdateOptions{})
		if err != nil {
			return nil, err
		}
	}

	app := &v1alpha1.Application{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(objApp.UnstructuredContent(), app); err != nil {
		return nil, err
	}
	oldApp := app.DeepCopy()
	app.Labels = utils.MapAny2Str(input.Labels)
	app.Annotations = utils.MapAny2Str(input.Annotations)
	if input.DisplayName != "" {
		app.Spec.DisplayName = input.DisplayName
	}
	if input.Description != nil && *input.Description != app.Spec.Description {
		app.Spec.Description = pointer.StringDeref(input.Description, app.Spec.Description)
	}
	if input.Icon != nil && *input.Icon != app.Spec.Icon {
		app.Spec.Icon = pointer.StringDeref(input.Icon, app.Spec.Icon)
	}
	if input.IsPublic != nil && *input.IsPublic != app.Spec.IsPublic {
		app.Spec.IsPublic = pointer.BoolDeref(input.IsPublic, app.Spec.IsPublic)
	}
	oldLLMName := ""
	oldLLMIndex := 0
	for i, node := range app.Spec.Nodes {
		if node.Ref != nil && node.Ref.Kind == "LLM" {
			oldLLMName = node.Ref.Name
			oldLLMIndex = i
		}
	}
	if input.LlmName != "" && oldLLMName != input.LlmName {
		app.Spec.Nodes[oldLLMIndex].Ref.Name = input.LlmName
	}
	if !reflect.DeepEqual(app, oldApp) {
		object, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&app)
		if err != nil {
			return nil, err
		}
		objApp, err = c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Application")).Namespace(input.Namespace).Update(ctx, &unstructured.Unstructured{Object: object}, metav1.UpdateOptions{})
		if err != nil {
			return nil, err
		}
	}
	return cr2model(objPrompt, objChain, objRetriever, objApp)
}

func DeleteKnowledgeBaseApplication(ctx context.Context, c dynamic.Interface, input generated.DeleteCommonInput) (*string, error) {
	resources := []dynamic.NamespaceableResourceInterface{
		c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Prompt")),
		c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Chain")),
		c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Retrievers")),
		c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Application")),
	}
	for _, resource := range resources {
		if input.Name != nil {
			err := resource.Namespace(input.Namespace).Delete(ctx, *input.Name, metav1.DeleteOptions{})
			if err != nil {
				return nil, err
			}
		} else {
			err := resource.Namespace(input.Namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
				LabelSelector: pointer.StringDeref(input.LabelSelector, ""),
				FieldSelector: pointer.StringDeref(input.FieldSelector, ""),
			})
			if err != nil {
				return nil, err
			}
		}
	}
	return nil, nil
}

func GetKnowledgeBaseApplication(ctx context.Context, c dynamic.Interface, name, namespace string) (*generated.KnowledgeBaseApplication, error) {
	resources := []dynamic.NamespaceableResourceInterface{
		c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Prompt")),
		c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Chain")),
		c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Retrievers")),
		c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Application")),
	}
	res := make([]*unstructured.Unstructured, len(resources))
	for i, resource := range resources {
		u, err := resource.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		res[i] = u
	}
	return cr2model(res[0], res[1], res[2], res[3])
}

func ListKnowledgeBaseApplications(ctx context.Context, c dynamic.Interface, input generated.ListCommonInput) (*generated.PaginatedResult, error) {
	keyword := pointer.StringDeref(input.Keyword, "")
	labelSelector := pointer.StringDeref(input.LabelSelector, "")
	fieldSelector := pointer.StringDeref(input.FieldSelector, "")
	page := pointer.IntDeref(input.Page, 1)
	pageSize := pointer.IntDeref(input.PageSize, 10)
	if labelSelector == "" {
		labelSelector = v1alpha1.ApplicationTypeLabel + "=" + v1alpha1.KnowledgeBaseApplicationType
	} else {
		labelSelector = labelSelector + "," + v1alpha1.ApplicationTypeLabel + "=" + v1alpha1.KnowledgeBaseApplicationType
	}
	res, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Application")).Namespace(input.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(res.Items, func(i, j int) bool {
		return res.Items[i].GetCreationTimestamp().After(res.Items[j].GetCreationTimestamp().Time)
	})

	totalCount := len(res.Items)

	filterd := make([]unstructured.Unstructured, 0)
	for _, u := range res.Items {
		if keyword != "" {
			displayName, _, _ := unstructured.NestedString(u.Object, "spec", "displayName")
			if !strings.Contains(u.GetName(), keyword) && !strings.Contains(displayName, keyword) {
				continue
			}
		}
		filterd = append(filterd, u)
	}
	end := page * pageSize
	if end > totalCount {
		end = totalCount
	}
	start := (page - 1) * pageSize
	if start < totalCount {
		filterd = filterd[start:end]
	} else {
		filterd = []unstructured.Unstructured{}
	}

	result := []generated.PageNode{}
	for _, u := range filterd {
		resources := []dynamic.NamespaceableResourceInterface{
			c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Prompt")),
			c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Chain")),
			c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Retrievers")),
		}
		res := make([]*unstructured.Unstructured, len(resources))
		for i, resource := range resources {
			u, err := resource.Namespace(u.GetNamespace()).Get(ctx, u.GetName(), metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			res[i] = u
		}
		m, err := cr2model(res[0], res[1], res[2], &u)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}

	return &generated.PaginatedResult{
		TotalCount:  totalCount,
		HasNextPage: end < totalCount,
		Nodes:       result,
		Page:        &page,
		PageSize:    &pageSize,
	}, nil
}
