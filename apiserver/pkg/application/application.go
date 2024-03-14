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
	"fmt"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	apiagent "github.com/kubeagi/arcadia/api/app-node/agent/v1alpha1"
	apichain "github.com/kubeagi/arcadia/api/app-node/chain/v1alpha1"
	apidocumentloader "github.com/kubeagi/arcadia/api/app-node/documentloader/v1alpha1"
	apiprompt "github.com/kubeagi/arcadia/api/app-node/prompt/v1alpha1"
	apiretriever "github.com/kubeagi/arcadia/api/app-node/retriever/v1alpha1"
	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	"github.com/kubeagi/arcadia/apiserver/pkg/utils"
)

func addCategory(app *v1alpha1.Application, category []*string) *v1alpha1.Application {
	if len(category) == 0 {
		delete(app.Annotations, v1alpha1.AppCategoryAnnotationKey)
		return app
	}
	if app.Annotations == nil {
		app.Annotations = make(map[string]string, 1)
	}
	c := make([]string, len(category))
	for i := range category {
		c[i] = *category[i]
	}
	app.Annotations[v1alpha1.AppCategoryAnnotationKey] = strings.Join(c, ",")
	return app
}

func addDefaultValue(gApp *generated.Application, app *v1alpha1.Application) {
	if len(app.Spec.Nodes) > 0 {
		return
	}
	gApp.DocNullReturn = pointer.String("未找到您询问的内容，请详细描述您的问题")
	gApp.NumDocuments = pointer.Int(5)
	gApp.ScoreThreshold = pointer.Float64(0.3)
	gApp.Temperature = pointer.Float64(0.7)
	gApp.MaxLength = pointer.Int(2048)
	gApp.MaxTokens = pointer.Int(2048)
	gApp.ConversionWindowSize = pointer.Int(5)
}

func cr2app(prompt *apiprompt.Prompt, chainConfig *apichain.CommonChainConfig, retriever *apiretriever.KnowledgeBaseRetriever, app *v1alpha1.Application, agent *apiagent.Agent) (*generated.Application, error) {
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
		Prologue:          pointer.String(app.Spec.Prologue),
		ShowNextGuide:     pointer.Bool(app.Spec.ShowNextGuide),
		ShowRespInfo:      pointer.Bool(app.Spec.ShowRespInfo),
		ShowRetrievalInfo: pointer.Bool(app.Spec.ShowRetrievalInfo),
		DocNullReturn:     pointer.String(app.Spec.DocNullReturn),
	}
	if prompt != nil {
		gApp.UserPrompt = pointer.String(prompt.Spec.UserMessage)
	}
	if chainConfig != nil {
		gApp.Model = pointer.String(chainConfig.Model)
		gApp.Temperature = chainConfig.Temperature
		gApp.MaxLength = pointer.Int(chainConfig.MaxLength)
		gApp.MaxTokens = pointer.Int(chainConfig.MaxTokens)
		gApp.ConversionWindowSize = chainConfig.Memory.ConversionWindowSize
	}
	if agent != nil && len(agent.Spec.AllowedTools) > 0 {
		for _, v := range agent.Spec.AllowedTools {
			gApp.Tools = append(gApp.Tools, &generated.Tool{
				Name:   pointer.String(v.Name),
				Params: utils.MapStr2Any(v.Params),
			})
		}
	}
	for _, node := range app.Spec.Nodes {
		if node.Ref == nil {
			continue
		}
		switch strings.ToLower(node.Ref.Kind) {
		case "llm":
			gApp.Llm = node.Ref.Name
		case "knowledgebase":
			gApp.Knowledgebase = pointer.String(node.Ref.Name)
		}
	}
	if retriever != nil {
		gApp.ScoreThreshold = pointer.Float64(float64(pointer.Float32Deref(retriever.Spec.ScoreThreshold, 0.0)))
		gApp.NumDocuments = pointer.Int(retriever.Spec.NumDocuments)
	}
	addDefaultValue(gApp, app)
	return gApp, nil
}

func app2metadataConverter(objApp client.Object) (generated.PageNode, error) {
	app, ok := objApp.(*v1alpha1.Application)
	if !ok {
		return nil, errors.New("can't convert client.Object to Application")
	}
	return app2metadata(app)
}

func app2metadata(app *v1alpha1.Application) (*generated.ApplicationMetadata, error) {
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
		Category:          common.GetAppCategory(app),
	}, nil
}

func CreateApplication(ctx context.Context, c client.Client, input generated.CreateApplicationMetadataInput) (*generated.ApplicationMetadata, error) {
	app := &v1alpha1.Application{
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
	app = addCategory(app, input.Category)
	common.SetCreator(ctx, &app.Spec.CommonSpec)
	if err := c.Create(ctx, app); err != nil {
		return nil, err
	}
	return app2metadata(app)
}

func UpdateApplication(ctx context.Context, c client.Client, input generated.UpdateApplicationMetadataInput) (*generated.ApplicationMetadata, error) {
	app := &v1alpha1.Application{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: input.Namespace, Name: input.Name}, app); err != nil {
		return nil, err
	}
	oldApp := app.DeepCopy()
	app.Labels = utils.MapAny2Str(input.Labels)
	app.Annotations = utils.MapAny2Str(input.Annotations)
	app = addCategory(app, input.Category)
	app.Spec.DisplayName = input.DisplayName
	app.Spec.Description = pointer.StringDeref(input.Description, app.Spec.Description)
	app.Spec.Icon = input.Icon
	app.Spec.IsPublic = pointer.BoolDeref(input.IsPublic, app.Spec.IsPublic)
	if !reflect.DeepEqual(app, oldApp) {
		if err := c.Update(ctx, app); err != nil {
			return nil, err
		}
	}
	return app2metadata(app)
}

func DeleteApplication(ctx context.Context, c client.Client, input generated.DeleteCommonInput) (*string, error) {
	resources := []client.Object{
		&apiprompt.Prompt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      *input.Name,
				Namespace: input.Namespace,
			},
		},
		&apichain.LLMChain{
			ObjectMeta: metav1.ObjectMeta{
				Name:      *input.Name,
				Namespace: input.Namespace,
			},
		},
		&apichain.RetrievalQAChain{
			ObjectMeta: metav1.ObjectMeta{
				Name:      *input.Name,
				Namespace: input.Namespace,
			},
		},
		&apiretriever.KnowledgeBaseRetriever{
			ObjectMeta: metav1.ObjectMeta{
				Name:      *input.Name,
				Namespace: input.Namespace,
			},
		},
		&apiagent.Agent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      *input.Name,
				Namespace: input.Namespace,
			},
		},
		&v1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      *input.Name,
				Namespace: input.Namespace,
			},
		},
	}
	for _, resource := range resources {
		opts, err := common.DeleteAllOptions(&input)
		if err != nil {
			return nil, err
		}
		err = c.DeleteAllOf(ctx, resource, opts...)
		if err != nil && !apierrors.IsNotFound(err) {
			return nil, err
		}
	}
	return pointer.String("ok"), nil
}

func GetApplication(ctx context.Context, c client.Client, name, namespace string) (*generated.Application, error) {
	key := types.NamespacedName{Namespace: namespace, Name: name}
	// 1. get application cr, if not exist, return error
	app := &v1alpha1.Application{}
	err := c.Get(ctx, key, app)
	if err != nil {
		return nil, err
	}

	prompt := &apiprompt.Prompt{}
	if err := c.Get(ctx, key, prompt); err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	var (
		chainConfig *apichain.CommonChainConfig
		retriever   *apiretriever.KnowledgeBaseRetriever
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
		if err := c.Get(ctx, key, qachain); err != nil && !apierrors.IsNotFound(err) {
			return nil, err
		}
		if qachain.UID != "" {
			chainConfig = &qachain.Spec.CommonChainConfig
		}
		retriever = &apiretriever.KnowledgeBaseRetriever{}
		if err := c.Get(ctx, key, retriever); err != nil && !apierrors.IsNotFound(err) {
			return nil, err
		}
	} else {
		llmchain := &apichain.LLMChain{}
		if err := c.Get(ctx, key, llmchain); err != nil && !apierrors.IsNotFound(err) {
			return nil, err
		}
		if llmchain.UID != "" {
			chainConfig = &llmchain.Spec.CommonChainConfig
		}
	}
	hasAgent := false
	for _, node := range app.Spec.Nodes {
		if node.Ref != nil && node.Ref.Kind == "Agent" {
			hasAgent = true
			break
		}
	}
	var agent *apiagent.Agent
	if hasAgent {
		agent = &apiagent.Agent{}
		if err := c.Get(ctx, key, agent); err != nil && !apierrors.IsNotFound(err) {
			return nil, err
		}
	}

	return cr2app(prompt, chainConfig, retriever, app, agent)
}

func ListApplicationMeatadatas(ctx context.Context, c client.Client, input generated.ListCommonInput) (*generated.PaginatedResult, error) {
	keyword := pointer.StringDeref(input.Keyword, "")
	page := pointer.IntDeref(input.Page, 1)
	pageSize := pointer.IntDeref(input.PageSize, 10)
	res := &v1alpha1.ApplicationList{}
	opts, err := common.NewListOptions(input)
	if err != nil {
		return nil, err
	}
	if err := c.List(ctx, res, opts...); err != nil {
		return nil, err
	}
	filter := make([]common.ResourceFilter, 0)
	if keyword != "" {
		filter = append(filter, common.FilterApplicationByKeyword(keyword))
	}
	items := make([]client.Object, len(res.Items))
	for i := range res.Items {
		items[i] = &res.Items[i]
	}
	return common.ListReources(items, page, pageSize, app2metadataConverter, filter...)
}

func UpdateApplicationConfig(ctx context.Context, c client.Client, input generated.UpdateApplicationConfigInput) (*generated.Application, error) {
	if len(input.Tools) != 0 {
		key := make(map[string]bool, len(input.Tools))
		for _, tool := range input.Tools {
			if _, exist := key[tool.Name]; exist {
				return nil, fmt.Errorf("duplicated tool name: %s", tool.Name)
			}
			key[tool.Name] = true
		}
	}
	key := types.NamespacedName{Namespace: input.Namespace, Name: input.Name}
	// 1. get application cr, if not exist, return error
	app := &v1alpha1.Application{}
	err := c.Get(ctx, key, app)
	if err != nil {
		return nil, err
	}

	// 2. create or update prompt
	prompt := &apiprompt.Prompt{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
		},
		Spec: apiprompt.PromptSpec{
			CommonSpec: v1alpha1.CommonSpec{
				DisplayName: "prompt",
				Description: "prompt",
			},
		},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, c, prompt, func() error {
		var userMessage string
		if !utils.HasValue(input.UserPrompt) {
			userMessage = apiprompt.DefaultUserPrompt
		} else {
			userMessage = *input.UserPrompt
		}
		prompt.Spec.CommonPromptConfig = apiprompt.CommonPromptConfig{
			UserMessage: userMessage,
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// 3. create or update documentloader
	documentLoader := &apidocumentloader.DocumentLoader{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
		},
		Spec: apidocumentloader.DocumentLoaderSpec{
			CommonSpec: v1alpha1.CommonSpec{
				DisplayName: "documentloader",
				Description: "documentloader",
			},
			ChunkSize:    1024,
			ChunkOverlap: pointer.Int(50),
			LoaderConfig: apidocumentloader.LoaderConfig{},
		},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, c, documentLoader, func() error {
		return nil
	}); err != nil {
		return nil, err
	}

	// 3. create or update chain
	var (
		chainConfig *apichain.CommonChainConfig
		retriever   *apiretriever.KnowledgeBaseRetriever
	)
	if utils.HasValue(input.Knowledgebase) {
		qachain := &apichain.RetrievalQAChain{
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
						ConversionWindowSize: input.ConversionWindowSize,
					},
					Model:       pointer.StringDeref(input.Model, ""),
					MaxLength:   pointer.IntDeref(input.MaxLength, 0),
					MaxTokens:   pointer.IntDeref(input.MaxTokens, 0),
					Temperature: input.Temperature,
				},
			},
		}
		if _, err = controllerutil.CreateOrUpdate(ctx, c, qachain, func() error {
			qachain.Spec.Model = pointer.StringDeref(input.Model, qachain.Spec.Model)
			qachain.Spec.MaxLength = pointer.IntDeref(input.MaxLength, qachain.Spec.MaxLength)
			qachain.Spec.MaxTokens = pointer.IntDeref(input.MaxTokens, qachain.Spec.MaxTokens)
			qachain.Spec.Temperature = input.Temperature
			qachain.Spec.Memory.ConversionWindowSize = input.ConversionWindowSize
			return nil
		}); err != nil {
			return nil, err
		}
		chainConfig = &qachain.Spec.CommonChainConfig
	} else {
		llmchain := &apichain.LLMChain{
			ObjectMeta: metav1.ObjectMeta{
				Name:      input.Name,
				Namespace: input.Namespace,
			},
			Spec: apichain.LLMChainSpec{
				CommonSpec: v1alpha1.CommonSpec{
					DisplayName: "llmchain",
					Description: "llmchain",
				},
				CommonChainConfig: apichain.CommonChainConfig{
					Memory: apichain.Memory{
						ConversionWindowSize: input.ConversionWindowSize,
					},
					Model:       pointer.StringDeref(input.Model, ""),
					MaxLength:   pointer.IntDeref(input.MaxLength, 0),
					MaxTokens:   pointer.IntDeref(input.MaxTokens, 0),
					Temperature: input.Temperature,
				},
			},
		}
		if _, err = controllerutil.CreateOrUpdate(ctx, c, llmchain, func() error {
			llmchain.Spec.Model = pointer.StringDeref(input.Model, llmchain.Spec.Model)
			llmchain.Spec.MaxLength = pointer.IntDeref(input.MaxLength, llmchain.Spec.MaxLength)
			llmchain.Spec.MaxTokens = pointer.IntDeref(input.MaxTokens, llmchain.Spec.MaxTokens)
			llmchain.Spec.Temperature = input.Temperature
			llmchain.Spec.Memory.ConversionWindowSize = input.ConversionWindowSize
			return nil
		}); err != nil {
			return nil, err
		}
		chainConfig = &llmchain.Spec.CommonChainConfig
	}

	// 4. create or update retriever
	if utils.HasValue(input.Knowledgebase) {
		retriever = &apiretriever.KnowledgeBaseRetriever{
			ObjectMeta: metav1.ObjectMeta{
				Name:      input.Name,
				Namespace: input.Namespace,
			},
			Spec: apiretriever.KnowledgeBaseRetrieverSpec{
				CommonSpec: v1alpha1.CommonSpec{
					DisplayName: "retriever",
					Description: "retriever",
				},
				CommonRetrieverConfig: apiretriever.CommonRetrieverConfig{
					ScoreThreshold: pointer.Float32(float32(pointer.Float64Deref(input.ScoreThreshold, 0.0))),
					NumDocuments:   pointer.IntDeref(input.NumDocuments, 0),
				},
			},
		}
		if _, err = controllerutil.CreateOrUpdate(ctx, c, retriever, func() error {
			retriever.Spec.ScoreThreshold = pointer.Float32(float32(pointer.Float64Deref(input.ScoreThreshold, 0.0)))
			retriever.Spec.NumDocuments = pointer.IntDeref(input.NumDocuments, retriever.Spec.NumDocuments)
			return nil
		}); err != nil {
			return nil, err
		}
	}

	// 5. create or update agent for tools
	var agent *apiagent.Agent
	if len(input.Tools) != 0 {
		agent = &apiagent.Agent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      input.Name,
				Namespace: input.Namespace,
			},
			Spec: apiagent.AgentSpec{
				CommonSpec: v1alpha1.CommonSpec{
					DisplayName: "agent",
					Description: "agent",
				},
			},
		}
		if _, err = controllerutil.CreateOrUpdate(ctx, c, agent, func() error {
			agent.Spec.AgentConfig.AllowedTools = []apiagent.Tool{}
			for _, v := range input.Tools {
				agent.Spec.AllowedTools = append(agent.Spec.AllowedTools, apiagent.Tool{
					Name:   v.Name,
					Params: utils.MapAny2Str(v.Params),
				})
			}
			agent.Spec.AgentConfig.Options.Memory.ConversionWindowSize = input.ConversionWindowSize
			return nil
		}); err != nil {
			return nil, err
		}
	}

	// 6. update application
	app = &v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
		},
	}
	appCopy := app.DeepCopy()
	_ = mutateApp(appCopy, input)
	if !equality.Semantic.DeepEqual(app.Spec, appCopy.Spec) {
		if _, err = controllerutil.CreateOrUpdate(ctx, c, app, func() error {
			return mutateApp(app, input)
		}); err != nil {
			return nil, err
		}
	}

	return cr2app(prompt, chainConfig, retriever, app, agent)
}

func mutateApp(app *v1alpha1.Application, input generated.UpdateApplicationConfigInput) error {
	app.Spec.Nodes = redefineNodes(input.Knowledgebase, input.Name, input.Llm, input.Tools)
	app.Spec.Prologue = pointer.StringDeref(input.Prologue, app.Spec.Prologue)
	app.Spec.ShowRespInfo = pointer.BoolDeref(input.ShowRespInfo, app.Spec.ShowRespInfo)
	app.Spec.ShowRetrievalInfo = pointer.BoolDeref(input.ShowRetrievalInfo, app.Spec.ShowRetrievalInfo)
	app.Spec.ShowNextGuide = pointer.BoolDeref(input.ShowNextGuide, app.Spec.ShowNextGuide)
	app.Spec.DocNullReturn = pointer.StringDeref(input.DocNullReturn, app.Spec.DocNullReturn)
	return nil
}

func redefineNodes(knowledgebase *string, name string, llmName string, tools []*generated.ToolInput) (nodes []v1alpha1.Node) {
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
			NextNodeName: []string{"documentloader-node"},
		},
		{
			NodeConfig: v1alpha1.NodeConfig{
				Name:        "documentloader-node",
				DisplayName: "documentloader",
				Description: "文档加载，可选",
				Ref: &v1alpha1.TypedObjectReference{
					APIGroup: pointer.String("arcadia.kubeagi.k8s.com.cn"),
					Kind:     "DocumentLoader",
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
	if len(tools) != 0 {
		nodes[len(nodes)-1].NextNodeName = []string{"chain-node", "agent-node"}
	}
	if knowledgebase == nil {
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
					Name:        "knowledgebase-node",
					DisplayName: "知识库",
					Description: "连接知识库",
					Ref: &v1alpha1.TypedObjectReference{
						APIGroup: pointer.String("arcadia.kubeagi.k8s.com.cn"),
						Kind:     "KnowledgeBase",
						Name:     pointer.StringDeref(knowledgebase, ""),
					},
				},
				NextNodeName: []string{"retriever-node"},
			},
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
	if len(tools) != 0 {
		nodes = append(nodes, v1alpha1.Node{
			NodeConfig: v1alpha1.NodeConfig{
				Name:        "agent-node",
				DisplayName: "agent",
				Description: "agent 调用复杂工具完成任务",
				Ref: &v1alpha1.TypedObjectReference{
					APIGroup: pointer.String("arcadia.kubeagi.k8s.com.cn"),
					Kind:     "Agent",
					Name:     name,
				},
			},
			NextNodeName: []string{"chain-node"},
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
