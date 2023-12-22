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
	"errors"
	"reflect"
	"sort"
	"strings"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/pointer"

	node "github.com/kubeagi/arcadia/api/app-node"
	apichain "github.com/kubeagi/arcadia/api/app-node/chain/v1alpha1"
	apiprompt "github.com/kubeagi/arcadia/api/app-node/prompt/v1alpha1"
	apiretriever "github.com/kubeagi/arcadia/api/app-node/retriever/v1alpha1"
	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	"github.com/kubeagi/arcadia/apiserver/pkg/utils"
)

func cr2app(prompt *apiprompt.Prompt, chainConfig *apichain.CommonChainConfig, chainInput *apichain.LLMChainInput, retriever *apiretriever.KnowledgeBaseRetriever, app *v1alpha1.Application) (*generated.Application, error) {
	if app == nil {
		return nil, errors.New("no app found")
	}
	condition := app.Status.GetCondition(v1alpha1.TypeReady)
	UpdateTimestamp := &condition.LastTransitionTime.Time
	status := common.GetObjStatus(app)

	gApp := &generated.Application{
		Metadata: &generated.ApplicationMetadata{
			Name:              app.Name,
			Namespace:         app.Namespace,
			ID:                pointer.String(string(app.UID)),
			Labels:            utils.MapStr2Any(app.Labels),
			Annotations:       utils.MapStr2Any(app.Annotations),
			DisplayName:       pointer.String(app.Spec.DisplayName),
			Description:       pointer.String(app.Spec.Description),
			Icon:              pointer.String(app.Spec.Icon),
			Creator:           pointer.String(app.Spec.Creator),
			CreationTimestamp: &app.CreationTimestamp.Time,
			UpdateTimestamp:   UpdateTimestamp,
			IsPublic:          pointer.Bool(app.Spec.IsPublic),
			Status:            pointer.String(status),
		},
		Prologue:     pointer.String(app.Spec.Prologue),
		ShowNextGUID: pointer.Bool(false),
	}
	if prompt != nil {
		gApp.UserPrompt = pointer.String(prompt.Spec.UserMessage)
	}
	if chainConfig != nil {
		gApp.Model = pointer.String(chainConfig.Model)
		gApp.Temperature = pointer.Float64(chainConfig.Temperature)
		gApp.MaxLength = pointer.Int(chainConfig.MaxLength)
		gApp.ConversionWindowSize = pointer.Int(chainConfig.Memory.ConversionWindowSize)
	}
	if chainInput != nil {
		gApp.Llm = chainInput.LLM.Name
	}
	if retriever != nil {
		gApp.Knowledgebase = pointer.String(retriever.Spec.Input.KnowledgeBaseRef.Name)
		gApp.ScoreThreshold = pointer.Float64(float64(retriever.Spec.ScoreThreshold))
		gApp.NumDocuments = pointer.Int(retriever.Spec.NumDocuments)
		gApp.DocNullReturn = pointer.String(retriever.Spec.DocNullReturn)
	}
	return gApp, nil
}

func app2metadata(objApp *unstructured.Unstructured) (*generated.ApplicationMetadata, error) {
	app := &v1alpha1.Application{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(objApp.UnstructuredContent(), app); err != nil {
		return nil, err
	}
	condition := app.Status.GetCondition(v1alpha1.TypeReady)
	UpdateTimestamp := &condition.LastTransitionTime.Time
	status := common.GetObjStatus(app)

	return &generated.ApplicationMetadata{
		Name:              app.Name,
		Namespace:         app.Namespace,
		ID:                pointer.String(string(app.UID)),
		Labels:            utils.MapStr2Any(app.Labels),
		Annotations:       utils.MapStr2Any(app.Annotations),
		Creator:           pointer.String(app.Spec.Creator),
		DisplayName:       pointer.String(app.Spec.DisplayName),
		Description:       pointer.String(app.Spec.Description),
		CreationTimestamp: &app.CreationTimestamp.Time,
		UpdateTimestamp:   UpdateTimestamp,
		Icon:              pointer.String(app.Spec.Icon),
		IsPublic:          pointer.Bool(app.Spec.IsPublic),
		Status:            pointer.String(status),
	}, nil
}

func CreateApplication(ctx context.Context, c dynamic.Interface, input generated.CreateApplicationMetadataInput) (*generated.ApplicationMetadata, error) {
	app := &v1alpha1.Application{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Application",
			APIVersion: common.ArcadiaAPIGroup,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        input.Name,
			Namespace:   input.Namespace,
			Labels:      utils.MapAny2Str(input.Labels),
			Annotations: utils.MapAny2Str(input.Annotations),
		},
		Spec: v1alpha1.ApplicationSpec{
			CommonSpec: v1alpha1.CommonSpec{
				DisplayName: input.DisplayName,
				Description: pointer.StringPtrDerefOr(input.Description, ""),
			},
			Icon:     input.Icon,
			IsPublic: pointer.BoolDeref(input.IsPublic, false),
			Prologue: "",
			Nodes:    []v1alpha1.Node{},
		},
	}
	object, err := runtime.DefaultUnstructuredConverter.ToUnstructured(app)
	if err != nil {
		return nil, err
	}
	objApp, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Application")).Namespace(input.Namespace).Create(ctx, &unstructured.Unstructured{Object: object}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return app2metadata(objApp)
}

func UpdateApplication(ctx context.Context, c dynamic.Interface, input generated.UpdateApplicationMetadataInput) (*generated.ApplicationMetadata, error) {
	objApp, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Application")).Namespace(input.Namespace).Get(ctx, input.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	app := &v1alpha1.Application{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(objApp.UnstructuredContent(), app); err != nil {
		return nil, err
	}
	oldApp := app.DeepCopy()
	app.Labels = utils.MapAny2Str(input.Labels)
	app.Annotations = utils.MapAny2Str(input.Annotations)
	app.Spec.DisplayName = input.DisplayName
	app.Spec.Description = pointer.StringDeref(input.Description, app.Spec.Description)
	app.Spec.Icon = input.Icon
	app.Spec.IsPublic = pointer.BoolDeref(input.IsPublic, app.Spec.IsPublic)
	if !reflect.DeepEqual(app, oldApp) {
		object, err := runtime.DefaultUnstructuredConverter.ToUnstructured(app)
		if err != nil {
			return nil, err
		}
		objApp, err = c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Application")).Namespace(input.Namespace).Update(ctx, &unstructured.Unstructured{Object: object}, metav1.UpdateOptions{})
		if err != nil {
			return nil, err
		}
	}
	return app2metadata(objApp)
}

func DeleteApplication(ctx context.Context, c dynamic.Interface, input generated.DeleteCommonInput) (*string, error) {
	resources := []dynamic.NamespaceableResourceInterface{
		c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Prompt")),
		c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "LLMChain")),
		c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "RetrievalQAChain")),
		c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "KnowledgeBaseRetriever")),
		c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Application")),
	}
	for _, resource := range resources {
		if input.Name != nil {
			err := resource.Namespace(input.Namespace).Delete(ctx, *input.Name, metav1.DeleteOptions{})
			if err != nil && !k8serrors.IsNotFound(err) {
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
	return pointer.String("ok"), nil
}

func GetApplication(ctx context.Context, c dynamic.Interface, name, namespace string) (*generated.Application, error) {
	// 1. get application cr, if not exist, return error
	_, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Application")).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	app := &v1alpha1.Application{}
	if err := getResource(ctx, c, common.SchemaOf(&common.ArcadiaAPIGroup, "Application"), namespace, name, app); err != nil {
		return nil, err
	}

	prompt := &apiprompt.Prompt{}
	if err := getResource(ctx, c, common.SchemaOf(&common.ArcadiaAPIGroup, "Prompt"), namespace, name, prompt); err != nil {
		return nil, err
	}
	var (
		chainConfig   *apichain.CommonChainConfig
		llmChainInput *apichain.LLMChainInput
		retriever     *apiretriever.KnowledgeBaseRetriever
	)
	hasKnowledgeBaseRetriever := false
	for _, node := range app.Spec.Nodes {
		if node.Ref != nil && node.Ref.APIGroup != nil && *node.Ref.APIGroup == apiretriever.Group {
			hasKnowledgeBaseRetriever = true
			break
		}
	}
	if hasKnowledgeBaseRetriever {
		qachain := &apichain.RetrievalQAChain{}
		if err := getResource(ctx, c, common.SchemaOf(&common.ArcadiaAPIGroup, "RetrievalQAChain"), namespace, name, qachain); err != nil {
			return nil, err
		}
		if qachain.UID != "" {
			chainConfig = &qachain.Spec.CommonChainConfig
			llmChainInput = &qachain.Spec.Input.LLMChainInput
		}
		retriever = &apiretriever.KnowledgeBaseRetriever{}
		if err := getResource(ctx, c, common.SchemaOf(&common.ArcadiaAPIGroup, "KnowledgeBaseRetriever"), namespace, name, retriever); err != nil {
			return nil, err
		}
	} else {
		llmchain := &apichain.LLMChain{}
		if err := getResource(ctx, c, common.SchemaOf(&common.ArcadiaAPIGroup, "LLMChain"), namespace, name, llmchain); err != nil {
			return nil, err
		}
		if llmchain.UID != "" {
			chainConfig = &llmchain.Spec.CommonChainConfig
			llmChainInput = &llmchain.Spec.Input
		}
	}

	return cr2app(prompt, chainConfig, llmChainInput, retriever, app)
}

func ListApplicationMeatadatas(ctx context.Context, c dynamic.Interface, input generated.ListCommonInput) (*generated.PaginatedResult, error) {
	keyword := pointer.StringDeref(input.Keyword, "")
	labelSelector := pointer.StringDeref(input.LabelSelector, "")
	fieldSelector := pointer.StringDeref(input.FieldSelector, "")
	page := pointer.IntDeref(input.Page, 1)
	pageSize := pointer.IntDeref(input.PageSize, 10)
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

	filterd := make([]generated.PageNode, 0)
	for _, u := range res.Items {
		if keyword != "" {
			displayName, _, _ := unstructured.NestedString(u.Object, "spec", "displayName")
			if !strings.Contains(u.GetName(), keyword) && !strings.Contains(displayName, keyword) {
				continue
			}
		}
		m, err := app2metadata(&u)
		if err != nil {
			return nil, err
		}
		filterd = append(filterd, m)
	}
	totalCount := len(filterd)

	end := page * pageSize
	if end > totalCount {
		end = totalCount
	}
	start := (page - 1) * pageSize
	if start < totalCount {
		filterd = filterd[start:end]
	} else {
		filterd = []generated.PageNode{}
	}

	return &generated.PaginatedResult{
		TotalCount:  totalCount,
		HasNextPage: end < totalCount,
		Nodes:       filterd,
		Page:        &page,
		PageSize:    &pageSize,
	}, nil
}

func UpdateApplicationConfig(ctx context.Context, c dynamic.Interface, input generated.UpdateApplicationConfigInput) (*generated.Application, error) {
	// 1. get application cr, if not exist, return error
	_, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Application")).Namespace(input.Namespace).Get(ctx, input.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	chainKind := "LLMChain"
	if input.Knowledgebase != nil {
		chainKind = "RetrievalQAChain"
	}

	// 2. create or update prompt
	prompt := &apiprompt.Prompt{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Prompt",
			APIVersion: apiprompt.Group + "/" + apiprompt.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
		},
		Spec: apiprompt.PromptSpec{
			CommonSpec: v1alpha1.CommonSpec{
				DisplayName: "prompt",
				Description: "prompt",
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
					Kind:     chainKind,
					Name:     input.Name,
				},
			},
		},
	}
	if err = createOrUpdateResource(ctx, c, common.SchemaOf(&common.ArcadiaAPIGroup, "Prompt"), input.Namespace, input.Name, func() {
		prompt.Spec.CommonPromptConfig = apiprompt.CommonPromptConfig{
			UserMessage: pointer.StringDeref(input.UserPrompt, "just say something."),
		}
	}, prompt); err != nil {
		return nil, err
	}

	// 3. create or update chain
	var (
		chainConfig   *apichain.CommonChainConfig
		llmchainInput *apichain.LLMChainInput
		retriever     *apiretriever.KnowledgeBaseRetriever
	)
	if input.Knowledgebase != nil {
		qachain := &apichain.RetrievalQAChain{
			TypeMeta: metav1.TypeMeta{
				Kind:       "RetrievalQAChain",
				APIVersion: apichain.Group + "/" + apichain.Version,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      input.Name,
				Namespace: input.Namespace,
			},
			Spec: apichain.RetrievalQAChainSpec{
				CommonSpec: v1alpha1.CommonSpec{
					DisplayName: "qachain",
					Description: "qachain",
				},
				CommonChainConfig: apichain.CommonChainConfig{
					Memory: apichain.Memory{
						ConversionWindowSize: pointer.IntDeref(input.ConversionWindowSize, 0),
					},
					Model:       pointer.StringDeref(input.Model, ""),
					MaxLength:   pointer.IntDeref(input.MaxLength, 0),
					Temperature: pointer.Float64Deref(input.Temperature, 0),
				},
				Input: apichain.RetrievalQAChainInput{
					LLMChainInput: apichain.LLMChainInput{
						LLM: node.LLMRef{
							Kind:     "LLM",
							Name:     input.Llm,
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
		if err = createOrUpdateResource(ctx, c, common.SchemaOf(&common.ArcadiaAPIGroup, strings.ToLower(chainKind)), input.Namespace, input.Name, func() {
			qachain.Spec.Model = pointer.StringDeref(input.Model, qachain.Spec.Model)
			qachain.Spec.MaxLength = pointer.IntDeref(input.MaxLength, qachain.Spec.MaxLength)
			qachain.Spec.Temperature = pointer.Float64Deref(input.Temperature, qachain.Spec.Temperature)
			qachain.Spec.Memory.ConversionWindowSize = pointer.IntDeref(input.ConversionWindowSize, qachain.Spec.Memory.ConversionWindowSize)
			qachain.Spec.Input.LLM.Name = input.Llm
		}, qachain); err != nil {
			return nil, err
		}
		chainConfig = &qachain.Spec.CommonChainConfig
		llmchainInput = &qachain.Spec.Input.LLMChainInput
	} else {
		llmchain := &apichain.LLMChain{
			TypeMeta: metav1.TypeMeta{
				Kind:       "LLMChain",
				APIVersion: apichain.Group + "/" + apichain.Version,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      input.Name,
				Namespace: input.Namespace,
			},
			Spec: apichain.LLMChainSpec{
				CommonSpec: v1alpha1.CommonSpec{
					DisplayName: "qachain",
					Description: "qachain",
				},
				CommonChainConfig: apichain.CommonChainConfig{
					Memory: apichain.Memory{
						ConversionWindowSize: pointer.IntDeref(input.ConversionWindowSize, 0),
					},
					Model:       pointer.StringDeref(input.Model, ""),
					MaxLength:   pointer.IntDeref(input.MaxLength, 0),
					Temperature: pointer.Float64Deref(input.Temperature, 0),
				},
				Input: apichain.LLMChainInput{
					LLM: node.LLMRef{
						Kind:     "LLM",
						Name:     input.Llm,
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
				Output: apichain.Output{
					CommonOrInPutOrOutputRef: node.CommonOrInPutOrOutputRef{
						Kind: "Output",
						Name: "Output",
					},
				},
			},
		}
		if err = createOrUpdateResource(ctx, c, common.SchemaOf(&common.ArcadiaAPIGroup, strings.ToLower(chainKind)), input.Namespace, input.Name, func() {
			llmchain.Spec.Model = pointer.StringDeref(input.Model, llmchain.Spec.Model)
			llmchain.Spec.MaxLength = pointer.IntDeref(input.MaxLength, llmchain.Spec.MaxLength)
			llmchain.Spec.Temperature = pointer.Float64Deref(input.Temperature, llmchain.Spec.Temperature)
			llmchain.Spec.Memory.ConversionWindowSize = pointer.IntDeref(input.ConversionWindowSize, llmchain.Spec.Memory.ConversionWindowSize)
			llmchain.Spec.Input.LLM.Name = input.Llm
		}, llmchain); err != nil {
			return nil, err
		}
		chainConfig = &llmchain.Spec.CommonChainConfig
		llmchainInput = &llmchain.Spec.Input
	}

	// 4. create or update retriever
	if input.Knowledgebase != nil {
		retriever = &apiretriever.KnowledgeBaseRetriever{
			TypeMeta: metav1.TypeMeta{
				Kind:       "KnowledgeBaseRetriever",
				APIVersion: apiretriever.Group + "/" + apiretriever.Version,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      input.Name,
				Namespace: input.Namespace,
			},
			Spec: apiretriever.KnowledgeBaseRetrieverSpec{
				CommonSpec: v1alpha1.CommonSpec{
					DisplayName: "retriever",
					Description: "retriever",
				},
				Input: apiretriever.Input{
					KnowledgeBaseRef: node.KnowledgeBaseRef{
						Kind:     "KnowledgeBase",
						Name:     *input.Knowledgebase,
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
				CommonRetrieverConfig: apiretriever.CommonRetrieverConfig{
					ScoreThreshold: float32(pointer.Float64Deref(input.ScoreThreshold, 0)),
					NumDocuments:   pointer.IntDeref(input.NumDocuments, 0),
					DocNullReturn:  pointer.StringDeref(input.DocNullReturn, ""),
				},
			},
		}
		if err = createOrUpdateResource(ctx, c, common.SchemaOf(&common.ArcadiaAPIGroup, "KnowledgeBaseRetriever"), input.Namespace, input.Name, func() {
			retriever.Spec.ScoreThreshold = float32(pointer.Float64Deref(input.ScoreThreshold, float64(retriever.Spec.ScoreThreshold)))
			retriever.Spec.NumDocuments = pointer.IntDeref(input.NumDocuments, retriever.Spec.NumDocuments)
			retriever.Spec.DocNullReturn = pointer.StringDeref(input.DocNullReturn, retriever.Spec.DocNullReturn)
			retriever.Spec.Input.KnowledgeBaseRef.Name = *input.Knowledgebase
		}, retriever); err != nil {
			return nil, err
		}
	}

	// 5. update application
	app := &v1alpha1.Application{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Application",
			APIVersion: common.ArcadiaAPIGroup,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
		},
	}
	if err = createOrUpdateResource(ctx, c, common.SchemaOf(&common.ArcadiaAPIGroup, "Application"), input.Namespace, input.Name, func() {
		app.Spec.Nodes = redefineNodes(input.Knowledgebase != nil, input.Name, input.Llm)
		app.Spec.Prologue = pointer.StringDeref(input.Prologue, app.Spec.Prologue)
	}, app); err != nil {
		return nil, err
	}

	return cr2app(prompt, chainConfig, llmchainInput, retriever, app)
}

func redefineNodes(hasknowledgebase bool, name, llmName string) (nodes []v1alpha1.Node) {
	nodes = []v1alpha1.Node{
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
					Name:     name,
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
					Name:     llmName,
				},
			},
			NextNodeName: []string{"chain-node"},
		},
	}
	if !hasknowledgebase {
		nodes = append(nodes, v1alpha1.Node{
			NodeConfig: v1alpha1.NodeConfig{
				Name:        "chain-node",
				DisplayName: "llm chain",
				Description: "chain是langchain的核心概念，llmChain用于连接prompt和llm",
				Ref: &v1alpha1.TypedObjectReference{
					APIGroup: pointer.String("chain.arcadia.kubeagi.k8s.com.cn"),
					Kind:     "LLMChain",
					Name:     name,
				},
			},
			NextNodeName: []string{"Output"},
		})
	} else {
		nodes = append(nodes,
			v1alpha1.Node{
				NodeConfig: v1alpha1.NodeConfig{
					Name:        "retriever-node",
					DisplayName: "从知识库提取信息的retriever",
					Description: "连接应用和知识库",
					Ref: &v1alpha1.TypedObjectReference{
						APIGroup: pointer.String("retriever.arcadia.kubeagi.k8s.com.cn"),
						Kind:     "KnowledgeBaseRetriever",
						Name:     name,
					},
				},
				NextNodeName: []string{"chain-node"},
			},
			v1alpha1.Node{
				NodeConfig: v1alpha1.NodeConfig{
					Name:        "chain-node",
					DisplayName: "RetrievalQA chain",
					Description: "chain是langchain的核心概念RetrievalQAChain用于从retriver中提取信息，供llm调用",
					Ref: &v1alpha1.TypedObjectReference{
						APIGroup: pointer.String("chain.arcadia.kubeagi.k8s.com.cn"),
						Kind:     "RetrievalQAChain",
						Name:     name,
					},
				},
				NextNodeName: []string{"Output"},
			})
	}
	nodes = append(nodes, v1alpha1.Node{
		NodeConfig: v1alpha1.NodeConfig{
			Name:        "Output",
			DisplayName: "最终输出",
			Description: "最终输出节点，必须",
			Ref: &v1alpha1.TypedObjectReference{
				Kind: "Output",
				Name: "Output",
			},
		},
	})
	return nodes
}

func createOrUpdateResource(ctx context.Context, c dynamic.Interface, resource schema.GroupVersionResource, namespace, name string, override func(), typedObj any) error {
	needUpdate := true
	resourceInterface := c.Resource(resource).Namespace(namespace)
	obj, err := resourceInterface.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
		needUpdate = false
	}
	if needUpdate {
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), typedObj); err != nil {
			return err
		}
	}
	override()
	object, err := runtime.DefaultUnstructuredConverter.ToUnstructured(typedObj)
	if err != nil {
		return err
	}
	if needUpdate {
		if obj, err = resourceInterface.Update(ctx, &unstructured.Unstructured{Object: object}, metav1.UpdateOptions{}); err != nil {
			return err
		}
	} else {
		if obj, err = resourceInterface.Create(ctx, &unstructured.Unstructured{Object: object}, metav1.CreateOptions{}); err != nil {
			return err
		}
	}
	return runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), typedObj)
}

func getResource(ctx context.Context, c dynamic.Interface, resource schema.GroupVersionResource, namespace, name string, typedObj any) error {
	resourceInterface := c.Resource(resource).Namespace(namespace)
	obj, err := resourceInterface.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), typedObj)
}
