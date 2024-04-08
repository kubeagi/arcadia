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
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/minio/minio-go/v7"
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
	pkgconf "github.com/kubeagi/arcadia/apiserver/config"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	"github.com/kubeagi/arcadia/apiserver/pkg/gpt"
	"github.com/kubeagi/arcadia/apiserver/pkg/utils"
	"github.com/kubeagi/arcadia/pkg/config"
	"github.com/kubeagi/arcadia/pkg/datasource"
	pkgutils "github.com/kubeagi/arcadia/pkg/utils"
)

func addCategory(app *v1alpha1.Application, category []*string) *v1alpha1.Application {
	if len(category) == 0 {
		app.Spec.Category = ""
		return app
	}
	c := make([]string, len(category))
	for i := range category {
		c[i] = *category[i]
	}
	app.Spec.Category = strings.Join(c, ",")
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

func cr2app(prompt *apiprompt.Prompt, chainConfig *apichain.CommonChainConfig, retriever *apiretriever.CommonRetrieverConfig, app *v1alpha1.Application, agent *apiagent.Agent, doc *apidocumentloader.DocumentLoader, enableRerank, enableMultiQuery *bool, rerankModel *string) (*generated.Application, error) {
	if app == nil {
		return nil, errors.New("no app found")
	}
	condition := app.Status.GetCondition(v1alpha1.TypeReady)
	UpdateTimestamp := &condition.LastTransitionTime.Time
	status := common.GetObjStatus(app)

	icon := common.AppIconLink(app, pkgconf.GetConfig().PlaygroundEndpointPrefix)
	gApp := &generated.Application{
		Metadata: &generated.ApplicationMetadata{
			Name:               app.Name,
			Namespace:          app.Namespace,
			ID:                 pointer.String(string(app.UID)),
			Labels:             utils.MapStr2Any(app.Labels),
			Annotations:        utils.MapStr2Any(app.Annotations),
			DisplayName:        pointer.String(app.Spec.DisplayName),
			Description:        pointer.String(app.Spec.Description),
			Icon:               &icon,
			Creator:            pointer.String(app.Spec.Creator),
			CreationTimestamp:  &app.CreationTimestamp.Time,
			UpdateTimestamp:    UpdateTimestamp,
			IsPublic:           pointer.Bool(app.Spec.IsPublic),
			IsRecommended:      pointer.Bool(app.Spec.IsRecommended),
			Status:             pointer.String(status),
			NotReadyReasonCode: pointer.String(string(gpt.GetGPTNotReadyReasonCode(app))),
		},
		Prologue:          pointer.String(app.Spec.Prologue),
		ShowNextGuide:     pointer.Bool(app.Spec.ShowNextGuide),
		ShowRespInfo:      pointer.Bool(app.Spec.ShowRespInfo),
		ShowRetrievalInfo: pointer.Bool(app.Spec.ShowRetrievalInfo),
		DocNullReturn:     pointer.String(app.Spec.DocNullReturn),
		ChatTimeout:       pointer.Float64(app.Spec.ChatTimeoutSecond),
		EnableUploadFile:  app.Spec.EnableUploadFile,
	}
	if prompt != nil {
		gApp.UserPrompt = pointer.String(prompt.Spec.UserMessage)
		gApp.SystemPrompt = pointer.String(prompt.Spec.SystemMessage)
	}
	if chainConfig != nil {
		gApp.Model = pointer.String(chainConfig.Model)
		gApp.Temperature = chainConfig.Temperature
		gApp.MaxLength = pointer.Int(chainConfig.MaxLength)
		gApp.MaxTokens = pointer.Int(chainConfig.MaxTokens)
		gApp.ConversionWindowSize = chainConfig.Memory.ConversionWindowSize
	}
	if agent != nil && agent.ResourceVersion != "" && len(agent.Spec.AllowedTools) > 0 {
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
		gApp.ScoreThreshold = pointer.Float64(float64(pointer.Float32Deref(retriever.ScoreThreshold, 0.0)))
		gApp.NumDocuments = pointer.Int(retriever.NumDocuments)
	}
	if doc != nil && doc.ResourceVersion != "" {
		gApp.BatchSize = pointer.Int(doc.Spec.BatchSize)
		gApp.ChunkSize = pointer.Int(doc.Spec.ChunkSize)
		gApp.ChunkOverlap = doc.Spec.ChunkOverlap
	}
	addDefaultValue(gApp, app)
	gApp.EnableRerank = enableRerank
	gApp.EnableMultiQuery = enableMultiQuery
	gApp.RerankModel = rerankModel
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

	icon := common.AppIconLink(app, pkgconf.GetConfig().PlaygroundEndpointPrefix)
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
		Icon:              &icon,
		IsPublic:          pointer.Bool(app.Spec.IsPublic),
		IsRecommended:     pointer.Bool(app.Spec.IsRecommended),
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
			IsPublic:      pointer.BoolDeref(input.IsPublic, false),
			IsRecommended: pointer.BoolDeref(input.IsRecommended, false),
			Prologue:      "",
			Nodes:         []v1alpha1.Node{},
		},
	}
	if len(input.Category) > 0 {
		app = addCategory(app, input.Category)
	}

	_, err := UploadIcon(ctx, c, input.Icon, input.Name, input.Namespace)
	if err != nil {
		return nil, err
	}
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
	if len(input.Labels) > 0 {
		app.Labels = utils.MapAny2Str(input.Labels)
	}
	if len(input.Annotations) > 0 {
		app.Annotations = utils.MapAny2Str(input.Annotations)
	}
	if len(input.Category) > 0 {
		app = addCategory(app, input.Category)
	}
	app.Spec.DisplayName = input.DisplayName
	app.Spec.Description = pointer.StringDeref(input.Description, app.Spec.Description)
	app.Spec.IsPublic = pointer.BoolDeref(input.IsPublic, app.Spec.IsPublic)
	app.Spec.IsRecommended = pointer.BoolDeref(input.IsRecommended, app.Spec.IsRecommended)
	_, err := UploadIcon(ctx, c, input.Icon, input.Name, input.Namespace)
	if err != nil {
		return nil, err
	}
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
		&apidocumentloader.DocumentLoader{
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
		&apiretriever.MultiQueryRetriever{
			ObjectMeta: metav1.ObjectMeta{
				Name:      *input.Name,
				Namespace: input.Namespace,
			},
		},
		&apiretriever.RerankRetriever{
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
		retriever   *apiretriever.CommonRetrieverConfig
	)
	hasKnowledgeBaseRetriever := false
	for _, node := range app.Spec.Nodes {
		if node.Ref != nil && node.Ref.APIGroup != nil && *node.Ref.APIGroup == apiretriever.Group {
			hasKnowledgeBaseRetriever = true
			break
		}
	}
	enableRerankRetriever := false
	rerankModel := ""
	enableMultiQueryRetriever := false
	if hasKnowledgeBaseRetriever {
		qachain := &apichain.RetrievalQAChain{}
		if err := c.Get(ctx, key, qachain); err != nil && !apierrors.IsNotFound(err) {
			return nil, err
		}
		if qachain.UID != "" {
			chainConfig = &qachain.Spec.CommonChainConfig
		}
		kbRetriever := &apiretriever.KnowledgeBaseRetriever{}
		if err := c.Get(ctx, key, kbRetriever); err != nil && !apierrors.IsNotFound(err) {
			return nil, err
		}
		if kbRetriever.ResourceVersion != "" {
			retriever = &kbRetriever.Spec.CommonRetrieverConfig
		}
		mulRetriever := &apiretriever.MultiQueryRetriever{}
		if err := c.Get(ctx, key, mulRetriever); err != nil && !apierrors.IsNotFound(err) {
			return nil, err
		}
		if mulRetriever.ResourceVersion != "" {
			retriever = &mulRetriever.Spec.CommonRetrieverConfig
			enableMultiQueryRetriever = true
		}
		rerankRetriever := &apiretriever.RerankRetriever{}
		if err := c.Get(ctx, key, rerankRetriever); err != nil && !apierrors.IsNotFound(err) {
			return nil, err
		}
		if rerankRetriever.ResourceVersion != "" {
			retriever = &rerankRetriever.Spec.CommonRetrieverConfig
			enableRerankRetriever = true
			if rerankRetriever.Spec.Model != nil {
				rerankModel = rerankRetriever.Spec.Model.Name
			}
		}
	} else {
		llmchain := &apichain.LLMChain{}
		if err := c.Get(ctx, key, llmchain); err != nil && !apierrors.IsNotFound(err) {
			return nil, err
		}
		if llmchain.ResourceVersion != "" {
			chainConfig = &llmchain.Spec.CommonChainConfig
		}
	}
	agent := &apiagent.Agent{}
	if err := c.Get(ctx, key, agent); err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	doc := &apidocumentloader.DocumentLoader{}
	if err := c.Get(ctx, key, doc); err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}

	return cr2app(prompt, chainConfig, retriever, app, agent, doc, pointer.Bool(enableRerankRetriever), pointer.Bool(enableMultiQueryRetriever), pointer.String(rerankModel))
}

func ListApplicationMeatadatas(ctx context.Context, c client.Client, input generated.ListCommonInput) (*generated.PaginatedResult, error) {
	keyword := pointer.StringDeref(input.Keyword, "")
	page := pointer.IntDeref(input.Page, 1)
	pageSize := pointer.IntDeref(input.PageSize, -1)
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
	// get application cr, if not exist, return error
	app := &v1alpha1.Application{}
	err := c.Get(ctx, key, app)
	if err != nil {
		return nil, err
	}

	// create or update prompt
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
		if utils.HasValue(input.SystemPrompt) {
			prompt.Spec.CommonPromptConfig.SystemMessage = *input.SystemPrompt
		} else {
			prompt.Spec.CommonPromptConfig.SystemMessage = ""
		}
		prompt.Spec.CommonPromptConfig.UserMessage = userMessage
		return nil
	}); err != nil {
		return nil, err
	}

	// create or update documentloader
	var documentLoader *apidocumentloader.DocumentLoader
	if pointer.BoolDeref(input.EnableUploadFile, false) {
		documentLoader = &apidocumentloader.DocumentLoader{
			ObjectMeta: metav1.ObjectMeta{
				Name:      input.Name,
				Namespace: input.Namespace,
			},
			Spec: apidocumentloader.DocumentLoaderSpec{
				CommonSpec: v1alpha1.CommonSpec{
					DisplayName: "documentloader",
					Description: "documentloader",
				},
				ChunkSize:    pointer.IntDeref(input.ChunkSize, 1024),
				ChunkOverlap: input.ChunkOverlap,
				BatchSize:    pointer.IntDeref(input.BatchSize, 3),
				LoaderConfig: apidocumentloader.LoaderConfig{},
			},
		}
		if _, err := controllerutil.CreateOrUpdate(ctx, c, documentLoader, func() error {
			documentLoader.Spec.ChunkSize = pointer.IntDeref(input.ChunkSize, documentLoader.Spec.ChunkSize)
			documentLoader.Spec.BatchSize = pointer.IntDeref(input.BatchSize, documentLoader.Spec.BatchSize)
			if input.ChunkOverlap == nil {
				documentLoader.Spec.ChunkOverlap = pointer.Int(50)
			} else {
				documentLoader.Spec.ChunkOverlap = input.ChunkOverlap
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}

	// create or update chain
	var (
		chainConfig *apichain.CommonChainConfig
		retriever   *apiretriever.CommonRetrieverConfig
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

	// create or update retrievers
	// knowledgebaseRetriever (must have) -> multiQueryRetriever (optional) -> rerankRetriever (optional) -> Output
	hasKnowledgebaseRetriever := utils.HasValue(input.Knowledgebase)
	hasMultiQueryRetriever := hasKnowledgebaseRetriever && pointer.BoolDeref(input.EnableMultiQuery, false)
	hasRerankRetriever := hasKnowledgebaseRetriever && pointer.BoolDeref(input.EnableRerank, false)
	rerankModel := ""
	var knowledgebaseRetriever *apiretriever.KnowledgeBaseRetriever
	if hasKnowledgebaseRetriever {
		knowledgebaseRetriever = &apiretriever.KnowledgeBaseRetriever{
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
					ScoreThreshold: pointer.Float32(float32(pointer.Float64Deref(input.ScoreThreshold, apiretriever.DefaultScoreThreshold))),
					NumDocuments:   pointer.IntDeref(input.NumDocuments, apiretriever.DefaultNumDocuments),
				},
			},
		}
		if _, err = controllerutil.CreateOrUpdate(ctx, c, knowledgebaseRetriever, func() error {
			if input.ScoreThreshold != nil {
				knowledgebaseRetriever.Spec.ScoreThreshold = pointer.Float32(float32(*input.ScoreThreshold))
			}
			knowledgebaseRetriever.Spec.NumDocuments = pointer.IntDeref(input.NumDocuments, knowledgebaseRetriever.Spec.NumDocuments)
			return nil
		}); err != nil {
			return nil, err
		}
		retriever = &knowledgebaseRetriever.Spec.CommonRetrieverConfig
	}

	if hasMultiQueryRetriever {
		multiQueryRetriever := &apiretriever.MultiQueryRetriever{
			ObjectMeta: metav1.ObjectMeta{
				Name:      input.Name,
				Namespace: input.Namespace,
			},
			Spec: apiretriever.MultiQueryRetrieverSpec{
				CommonSpec: v1alpha1.CommonSpec{
					DisplayName: "multiquery retriever",
					Description: "multiquery retriever",
				},
				CommonRetrieverConfig: apiretriever.CommonRetrieverConfig{
					ScoreThreshold: pointer.Float32(float32(pointer.Float64Deref(input.ScoreThreshold, apiretriever.DefaultScoreThreshold))),
					NumDocuments:   pointer.IntDeref(input.NumDocuments, apiretriever.DefaultNumDocuments),
				},
			},
		}
		if _, err = controllerutil.CreateOrUpdate(ctx, c, multiQueryRetriever, func() error {
			if input.ScoreThreshold != nil {
				multiQueryRetriever.Spec.ScoreThreshold = pointer.Float32(float32(*input.ScoreThreshold))
			}
			if hasRerankRetriever {
				// knowledgebaseRetriever -> (input.NumDocuments) doc -> multiQueryRetriever ->(input.NumDocuments * 4) doc ->  rerankRetriever -> (input.NumDocuments) doc -> output
				// rerankRetriever has more documents can make it more accurate
				multiQueryRetriever.Spec.NumDocuments = knowledgebaseRetriever.Spec.NumDocuments * 4 // will send 4 questions, the final number provided to the user is controlled by rerankRetriever
			} else {
				// knowledgebaseRetriever -> (input.NumDocuments) doc -> multiQueryRetriever ->(input.NumDocuments) doc -> output
				multiQueryRetriever.Spec.NumDocuments = *input.NumDocuments
			}
			return nil
		}); err != nil {
			return nil, err
		}
		retriever = &multiQueryRetriever.Spec.CommonRetrieverConfig
	}
	if hasRerankRetriever {
		rerankRetriever := &apiretriever.RerankRetriever{
			ObjectMeta: metav1.ObjectMeta{
				Name:      input.Name,
				Namespace: input.Namespace,
			},
			Spec: apiretriever.RerankRetrieverSpec{
				CommonRetrieverConfig: apiretriever.CommonRetrieverConfig{
					ScoreThreshold: pointer.Float32(float32(pointer.Float64Deref(input.ScoreThreshold, apiretriever.DefaultScoreThreshold))),
					NumDocuments:   pointer.IntDeref(input.NumDocuments, apiretriever.DefaultNumDocuments),
				},
			},
		}
		var model *v1alpha1.Model
		if pointer.StringDeref(input.RerankModel, "") != "" {
			model = &v1alpha1.Model{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pointer.StringDeref(input.RerankModel, ""),
					Namespace: input.Namespace,
				},
			}
			rerankRetriever.Spec.Model = model.TypedObjectReference()
		}
		if _, err = controllerutil.CreateOrUpdate(ctx, c, rerankRetriever, func() error {
			if input.ScoreThreshold != nil {
				rerankRetriever.Spec.ScoreThreshold = pointer.Float32(float32(*input.ScoreThreshold))
			}
			rerankRetriever.Spec.NumDocuments = pointer.IntDeref(input.NumDocuments, rerankRetriever.Spec.NumDocuments)
			if model != nil {
				rerankRetriever.Spec.Model = model.TypedObjectReference()
			} else {
				rerankRetriever.Spec.Model = nil
			}
			return nil
		}); err != nil {
			return nil, err
		}
		retriever = &rerankRetriever.Spec.CommonRetrieverConfig
		if rerankRetriever.Spec.Model != nil {
			rerankModel = rerankRetriever.Spec.Model.Name
		}
	}
	if !hasMultiQueryRetriever {
		multiQueryRetriever := &apiretriever.MultiQueryRetriever{
			ObjectMeta: metav1.ObjectMeta{
				Name:      input.Name,
				Namespace: input.Namespace,
			},
		}
		_ = c.Delete(ctx, multiQueryRetriever)
	}
	if !hasRerankRetriever {
		reRankRetriever := &apiretriever.RerankRetriever{
			ObjectMeta: metav1.ObjectMeta{
				Name:      input.Name,
				Namespace: input.Namespace,
			},
		}
		_ = c.Delete(ctx, reRankRetriever)
	}
	// create or update agent for tools
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

	// update application
	app = &v1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
		},
	}
	appCopy := app.DeepCopy()
	_ = mutateApp(appCopy, input, hasMultiQueryRetriever, hasRerankRetriever)
	if !equality.Semantic.DeepEqual(app.Spec, appCopy.Spec) {
		if _, err = controllerutil.CreateOrUpdate(ctx, c, app, func() error {
			return mutateApp(app, input, hasMultiQueryRetriever, hasRerankRetriever)
		}); err != nil {
			return nil, err
		}
	}

	return cr2app(prompt, chainConfig, retriever, app, agent, documentLoader, pointer.Bool(hasRerankRetriever), pointer.Bool(hasMultiQueryRetriever), pointer.String(rerankModel))
}

func mutateApp(app *v1alpha1.Application, input generated.UpdateApplicationConfigInput, hasMultiQueryRetriever, hasRerankRetriever bool) error {
	app.Spec.Nodes = redefineNodes(input.Knowledgebase, input.Namespace, input.Name, input.Llm, input.Tools, hasMultiQueryRetriever, hasRerankRetriever, input.EnableUploadFile)
	app.Spec.Prologue = pointer.StringDeref(input.Prologue, app.Spec.Prologue)
	app.Spec.ShowRespInfo = pointer.BoolDeref(input.ShowRespInfo, app.Spec.ShowRespInfo)
	app.Spec.ShowRetrievalInfo = pointer.BoolDeref(input.ShowRetrievalInfo, app.Spec.ShowRetrievalInfo)
	app.Spec.ShowNextGuide = pointer.BoolDeref(input.ShowNextGuide, app.Spec.ShowNextGuide)
	app.Spec.DocNullReturn = pointer.StringDeref(input.DocNullReturn, app.Spec.DocNullReturn)
	app.Spec.ChatTimeoutSecond = pointer.Float64Deref(input.ChatTimeout, v1alpha1.DefaultChatTimeoutSeconds)
	if input.EnableUploadFile != nil {
		app.Spec.EnableUploadFile = input.EnableUploadFile
	}
	return nil
}

// redefineNodes redefine nodes in application
func redefineNodes(knowledgebase *string, namespace string, name string, llmName string, tools []*generated.ToolInput, hasMultiQueryRetriever, hasRerankRetriever bool, enableUploadFile *bool) (nodes []v1alpha1.Node) {
	nodes = []v1alpha1.Node{
		{
			NodeConfig: v1alpha1.NodeConfig{
				Name:        "Input",
				DisplayName: "用户输入",
				Description: "用户输入节点，必须",
				Ref: &v1alpha1.TypedObjectReference{
					Kind:      "Input",
					Name:      "Input",
					Namespace: &namespace,
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
					APIGroup:  pointer.String("prompt.arcadia.kubeagi.k8s.com.cn"),
					Kind:      "Prompt",
					Name:      name,
					Namespace: &namespace,
				},
			},
			NextNodeName: []string{"chain-node"},
		},
	}
	if pointer.BoolDeref(enableUploadFile, false) {
		nodes = append(nodes, v1alpha1.Node{
			NodeConfig: v1alpha1.NodeConfig{
				Name:        "documentloader-node",
				DisplayName: "documentloader",
				Description: "文档加载，可选",
				Ref: &v1alpha1.TypedObjectReference{
					APIGroup:  pointer.String("arcadia.kubeagi.k8s.com.cn"),
					Kind:      "DocumentLoader",
					Name:      name,
					Namespace: &namespace,
				},
			},
			NextNodeName: []string{"chain-node"},
		})
	}
	nodes = append(nodes, v1alpha1.Node{
		NodeConfig: v1alpha1.NodeConfig{
			Name:        "llm-node",
			DisplayName: "llm",
			Description: "设定大模型的访问信息",
			Ref: &v1alpha1.TypedObjectReference{
				APIGroup:  pointer.String("arcadia.kubeagi.k8s.com.cn"),
				Kind:      "LLM",
				Name:      llmName,
				Namespace: &namespace,
			},
		},
		NextNodeName: []string{"chain-node"},
	})
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
					APIGroup:  pointer.String("chain.arcadia.kubeagi.k8s.com.cn"),
					Kind:      "LLMChain",
					Name:      name,
					Namespace: &namespace,
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
						APIGroup:  pointer.String("arcadia.kubeagi.k8s.com.cn"),
						Kind:      "KnowledgeBase",
						Name:      pointer.StringDeref(knowledgebase, ""),
						Namespace: &namespace,
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
						APIGroup:  pointer.String("retriever.arcadia.kubeagi.k8s.com.cn"),
						Kind:      "KnowledgeBaseRetriever",
						Name:      name,
						Namespace: &namespace,
					},
				},
				NextNodeName: []string{"chain-node"},
			})
		knowledgebaseRetrierNextNodeName := "chain-node"
		switch {
		case hasMultiQueryRetriever:
			knowledgebaseRetrierNextNodeName = "multiqueryretriever-node"
		case !hasMultiQueryRetriever && hasRerankRetriever:
			knowledgebaseRetrierNextNodeName = "rerankretriever-node"
		case !hasMultiQueryRetriever && !hasRerankRetriever:
			knowledgebaseRetrierNextNodeName = "chain-node"
		}
		nodes[len(nodes)-1].NextNodeName = []string{knowledgebaseRetrierNextNodeName}
		if hasMultiQueryRetriever {
			nextNodeName := "chain-node"
			if hasRerankRetriever {
				nextNodeName = "rerankretriever-node"
			}
			nodes = append(nodes,
				v1alpha1.Node{
					NodeConfig: v1alpha1.NodeConfig{
						Name:        "multiqueryretriever-node",
						DisplayName: "多查询retriever",
						Description: "多查询retriever",
						Ref: &v1alpha1.TypedObjectReference{
							APIGroup:  pointer.String("retriever.arcadia.kubeagi.k8s.com.cn"),
							Kind:      "MultiQueryRetriever",
							Name:      name,
							Namespace: &namespace,
						},
					},
					NextNodeName: []string{nextNodeName},
				})
		}
		if hasRerankRetriever {
			nodes = append(nodes,
				v1alpha1.Node{
					NodeConfig: v1alpha1.NodeConfig{
						Name:        "rerankretriever-node",
						DisplayName: "rerank retriever",
						Description: "rerank retriever",
						Ref: &v1alpha1.TypedObjectReference{
							APIGroup:  pointer.String("retriever.arcadia.kubeagi.k8s.com.cn"),
							Kind:      "RerankRetriever",
							Name:      name,
							Namespace: &namespace,
						},
					},
					NextNodeName: []string{"chain-node"},
				})
		}
		nodes = append(nodes,
			v1alpha1.Node{
				NodeConfig: v1alpha1.NodeConfig{
					Name:        "chain-node",
					DisplayName: "RetrievalQA chain",
					Description: "chain是langchain的核心概念RetrievalQAChain用于从retriever中提取信息，供llm调用",
					Ref: &v1alpha1.TypedObjectReference{
						APIGroup:  pointer.String("chain.arcadia.kubeagi.k8s.com.cn"),
						Kind:      "RetrievalQAChain",
						Name:      name,
						Namespace: &namespace,
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
					APIGroup:  pointer.String("arcadia.kubeagi.k8s.com.cn"),
					Kind:      "Agent",
					Name:      name,
					Namespace: &namespace,
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
				Kind:      "Output",
				Name:      "Output",
				Namespace: &namespace,
			},
		},
	})
	return nodes
}

func UploadIcon(ctx context.Context, client client.Client, icon, appName, namespace string) (string, error) {
	if strings.HasPrefix(icon, "data:image") {
		imgBytes, err := pkgutils.ParseBase64ImageBytes(icon)
		if err != nil {
			return "", err
		}

		system, err := config.GetSystemDatasource(ctx)
		if err != nil {
			return "", err
		}

		endpoint := system.Spec.Endpoint.DeepCopy()
		if endpoint != nil && endpoint.AuthSecret != nil {
			endpoint.AuthSecret.WithNameSpace(namespace)
		}
		ds, err := datasource.NewOSS(ctx, client, endpoint)
		if err != nil {
			return "", err
		}
		iconName := fmt.Sprintf("application/%s/icon/%s", appName, appName)
		_, err = ds.Client.PutObject(ctx, namespace, iconName, bytes.NewReader(imgBytes), int64(len(imgBytes)), minio.PutObjectOptions{})
		return icon, err
	}
	return icon, nil
}
