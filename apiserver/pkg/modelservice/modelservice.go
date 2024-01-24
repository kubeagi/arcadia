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

package modelservice

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"k8s.io/client-go/dynamic"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	"github.com/kubeagi/arcadia/apiserver/pkg/embedder"
	"github.com/kubeagi/arcadia/apiserver/pkg/llm"
	"github.com/kubeagi/arcadia/apiserver/pkg/worker"
	llmspkg "github.com/kubeagi/arcadia/pkg/llms"
	"github.com/kubeagi/arcadia/pkg/llms/openai"
	"github.com/kubeagi/arcadia/pkg/llms/zhipuai"
)

// CreateModelService creates a 3rd_party model service
// If serviceType is llm,embedding,then a LLM and a Embedder will be created
// - Wrap all elements into *generated.ModelService
func CreateModelService(ctx context.Context, c dynamic.Interface, input generated.CreateModelServiceInput) (*generated.ModelService, error) {
	// - Get general info from input: displayName, description, types, name & namespace, etc.
	displayName, description, serviceType, APIType := "", "", "", ""

	if input.DisplayName != nil {
		displayName = *input.DisplayName
	}
	if input.Description != nil {
		description = *input.Description
	}
	if input.Types != nil {
		serviceType = *input.Types
	}
	if input.APIType != nil {
		APIType = *input.APIType
	}

	var modelSerivce = &generated.ModelService{}
	var llmModels, embeddingModels []string
	// Create LLM if serviceType contains llm
	if strings.Contains(serviceType, "llm") {
		llm, err := llm.CreateLLM(ctx, c, generated.CreateLLMInput{
			Name:          input.Name,
			Namespace:     input.Namespace,
			DisplayName:   &displayName,
			Description:   &description,
			Labels:        input.Labels,
			Annotations:   input.Annotations,
			Type:          &APIType,
			Endpointinput: input.Endpoint,
			Models:        input.LlmModels,
		})
		if err != nil {
			return nil, err
		}
		llmModels = llm.Models
		modelSerivce = LLM2ModelService(llm)
	}

	// Create Embedder if serviceType contains embedding
	if strings.Contains(serviceType, "embedding") {
		embedder, err := embedder.CreateEmbedder(ctx, c, generated.CreateEmbedderInput{
			Name:          input.Name,
			Namespace:     input.Namespace,
			DisplayName:   &displayName,
			Description:   &description,
			Labels:        input.Labels,
			Annotations:   input.Annotations,
			Type:          &APIType,
			Endpointinput: input.Endpoint,
			Models:        input.EmbeddingModels,
		})
		if err != nil {
			return nil, err
		}
		embeddingModels = embedder.Models
		modelSerivce = Embedder2ModelService(embedder)
	}

	// merge llm&embedder
	modelSerivce.Types = &serviceType
	modelSerivce.LlmModels = llmModels
	modelSerivce.EmbeddingModels = embeddingModels

	return modelSerivce, nil
}

// UpdateModelService updates a 3rd_party model service
func UpdateModelService(ctx context.Context, c dynamic.Interface, input *generated.UpdateModelServiceInput) (*generated.ModelService, error) {
	var updatedLLM *generated.Llm
	var updatedEmbedder *generated.Embedder

	ms, err := ReadModelService(ctx, c, input.Name, input.Namespace)
	if err != nil {
		return nil, errors.New("read model service failed: " + err.Error())
	}

	var newDisplayName, newDescription, newAPIType string
	var newLabels, newAnnotations map[string]interface{}

	if input.DisplayName != nil {
		newDisplayName = *input.DisplayName
	} else {
		newDisplayName = *ms.DisplayName
	}
	if input.Description != nil {
		newDescription = *input.Description
	} else {
		newDescription = *ms.Description
	}
	if input.APIType != nil {
		newAPIType = *input.APIType
	} else {
		newAPIType = *ms.APIType
	}
	if input.Labels != nil {
		newLabels = input.Labels
	} else {
		newLabels = ms.Labels
	}
	if input.Annotations != nil {
		newAnnotations = input.Annotations
	} else {
		newAnnotations = ms.Annotations
	}

	updateLLMInput := generated.UpdateLLMInput{
		Name:          input.Name,
		Namespace:     input.Namespace,
		DisplayName:   &newDisplayName,
		Description:   &newDescription,
		Labels:        newLabels,
		Annotations:   newAnnotations,
		Type:          &newAPIType,
		Endpointinput: &input.Endpoint,
		Models:        input.LlmModels,
	}

	updateEmbedderInput := generated.UpdateEmbedderInput{
		Name:          input.Name,
		Namespace:     input.Namespace,
		DisplayName:   &newDisplayName,
		Description:   &newDescription,
		Labels:        newLabels,
		Annotations:   newAnnotations,
		Type:          &newAPIType,
		Endpointinput: &input.Endpoint,
		Models:        input.EmbeddingModels,
	}

	// TODO: codes to delete/create llm/embedding resource if input.Types is changed. For now it will not work.
	var updatedModelSerivce = &generated.ModelService{}
	var llmModels, embeddingModels []string
	if strings.Contains(*ms.Types, "llm") {
		updatedLLM, err = llm.UpdateLLM(ctx, c, &updateLLMInput)
		if err != nil {
			return nil, errors.New("update LLM failed: " + err.Error())
		}
		llmModels = updatedLLM.Models
		updatedModelSerivce = LLM2ModelService(updatedLLM)
	}

	if strings.Contains(*ms.Types, "embedding") {
		updatedEmbedder, err = embedder.UpdateEmbedder(ctx, c, &updateEmbedderInput)
		if err != nil {
			return nil, errors.New("update embedding failed: " + err.Error())
		}
		embeddingModels = updatedEmbedder.Models
		updatedModelSerivce = Embedder2ModelService(updatedEmbedder)
	}

	// merge llm&embedder
	updatedModelSerivce.Types = ms.Types
	updatedModelSerivce.LlmModels = llmModels
	updatedModelSerivce.EmbeddingModels = embeddingModels

	return updatedModelSerivce, nil
}

// DeleteModelService deletes a 3rd_party model service
func DeleteModelService(ctx context.Context, c dynamic.Interface, input *generated.DeleteCommonInput) (*string, error) {
	// check types of the model service
	ms, err := ReadModelService(ctx, c, *input.Name, input.Namespace)
	if err != nil {
		return nil, err
	}
	if ms.Types == nil {
		return nil, errors.New("model service's type does not exist")
	}
	if strings.Contains(*ms.Types, "llm") {
		_, err := llm.DeleteLLMs(ctx, c, input)
		if err != nil {
			return nil, err
		}
	}
	if strings.Contains(*ms.Types, "embedding") {
		_, err := embedder.DeleteEmbedders(ctx, c, input)
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// ReadModelService get a 3rd_party model service
func ReadModelService(ctx context.Context, c dynamic.Interface, name string, namespace string) (*generated.ModelService, error) {
	var modelService = &generated.ModelService{}
	var serviceTypes []string
	var llmModels, embeddingModels []string
	llm, err := llm.ReadLLM(ctx, c, name, namespace)
	if err == nil {
		llmModels = llm.Models
		serviceTypes = append(serviceTypes, common.ModelTypeLLM)
		modelService = LLM2ModelService(llm)
	}
	embedder, err := embedder.ReadEmbedder(ctx, c, name, namespace)
	if err == nil {
		embeddingModels = embedder.Models
		serviceTypes = append(serviceTypes, common.ModelTypeEmbedding)
		modelService = Embedder2ModelService(embedder)
	}

	serviceTypeStr := strings.Join(serviceTypes, ",")
	modelService.Types = &serviceTypeStr
	modelService.LlmModels = llmModels
	modelService.EmbeddingModels = embeddingModels

	return modelService, nil
}

// ListModelServices based on input
func ListModelServices(ctx context.Context, c dynamic.Interface, input *generated.ListModelServiceInput) (*generated.PaginatedResult, error) {
	// use `UnlimitedPageSize` so we can get all llms and embeddings
	notWorkerSelector := fmt.Sprintf("%s=%s", v1alpha1.ProviderLabel, v1alpha1.ProviderType3rdParty)

	query := generated.ListCommonInput{
		Page:          input.Page,
		PageSize:      &common.UnlimitedPageSize,
		Namespace:     input.Namespace,
		Keyword:       input.Keyword,
		LabelSelector: &notWorkerSelector,
	}

	newNodeList := make([]generated.PageNode, 0)

	if input.ProviderType == nil || *input.ProviderType == string(v1alpha1.ProviderType3rdParty) {
		exist := make(map[string]int)
		if input.Types == nil || (*input.Types == common.ModelTypeAll || *input.Types == common.ModelTypeLLM) {
			llmList, err := llm.ListLLMs(ctx, c, query, common.WithPageNodeConvertFunc(func(a any) generated.PageNode {
				llm, ok := a.(*generated.Llm)
				if !ok {
					return nil
				}
				return LLM2ModelService(llm)
			}))
			if err != nil {
				return nil, err
			}
			for _, n := range llmList.Nodes {
				tmp := n.(*generated.ModelService)
				if input.APIType != nil && *input.APIType != *tmp.APIType {
					continue
				}
				newNodeList = append(newNodeList, tmp)
				exist[tmp.Name] = len(newNodeList) - 1
			}
		}

		if input.Types == nil || (*input.Types == common.ModelTypeAll || *input.Types == common.ModelTypeEmbedding) {
			embedderList, err := embedder.ListEmbedders(ctx, c, query, common.WithPageNodeConvertFunc(func(a any) generated.PageNode {
				embedder, ok := a.(*generated.Embedder)
				if !ok {
					return nil
				}
				return Embedder2ModelService(embedder)
			}))
			if err != nil {
				return nil, err
			}
			for _, n := range embedderList.Nodes {
				tmp := n.(*generated.ModelService)
				if input.APIType != nil && *input.APIType != *tmp.APIType {
					continue
				}
				if idx, ok := exist[tmp.Name]; ok {
					t := newNodeList[idx].(*generated.ModelService)
					t.Types = &common.ModelTypeAll
					t.EmbeddingModels = tmp.EmbeddingModels
					continue
				}
				newNodeList = append(newNodeList, tmp)
			}
		}
	}

	if input.ProviderType == nil || *input.ProviderType == string(v1alpha1.ProviderTypeWorker) {
		if input.APIType == nil || *input.APIType == string(llmspkg.OpenAI) {
			workerQuery := generated.ListWorkerInput{
				Page:      query.Page,
				PageSize:  &common.UnlimitedPageSize,
				Namespace: input.Namespace,
				Keyword:   input.Keyword,
			}
			workerList, err := worker.ListWorkers(ctx, c, workerQuery, false)
			if err != nil {
				return nil, err
			}
			for _, n := range workerList.Nodes {
				tmp := n.(*generated.Worker)
				newNodeList = append(newNodeList, Worker2ModelService(tmp))
			}
		}
	}

	sort.Slice(newNodeList, func(i, j int) bool {
		a := newNodeList[i].(*generated.ModelService)
		b := newNodeList[j].(*generated.ModelService)
		return a.CreationTimestamp.After(*b.CreationTimestamp)
	})

	// return ModelService with the actual Page and PageSize
	page, pageSize := 1, 10
	if input.Page != nil && *input.Page > 0 {
		page = *input.Page
	}
	if input.PageSize != nil && *input.PageSize > 0 {
		pageSize = *input.PageSize
	}

	totalCount := len(newNodeList)
	start, end := common.PagePosition(page, pageSize, totalCount)

	return &generated.PaginatedResult{
		TotalCount:  totalCount,
		HasNextPage: end < totalCount,
		Nodes:       newNodeList[start:end],
	}, nil
}

var (
	ErrWrongAuthFormat  = errors.New("wrong auth format, auth[\"apikey\"] should be string")
	ErrNoAuthProvided   = errors.New("no auth provided")
	ErrNoAPIKeyProvided = errors.New("no apiKey provided")
)

func CheckModelService(ctx context.Context, c dynamic.Interface, input generated.CreateModelServiceInput) (*generated.ModelService, error) {
	var err error
	if input.Endpoint.Auth != nil {
		var info string
		if input.Endpoint.Auth["apiKey"] == nil {
			return nil, ErrNoAPIKeyProvided
		}
		if _, ok := input.Endpoint.Auth["apiKey"].(string); !ok {
			return nil, ErrWrongAuthFormat
		}

		switch *input.APIType {
		case "openai":
			info, err = checkOpenAI(ctx, c, input)
		case "zhipuai":
			info, err = checkZhipuAI(ctx, c, input)
		default:
			err = fmt.Errorf("not support api type %s", *input.APIType)
		}

		if err != nil {
			return nil, err
		}
		return &generated.ModelService{
			// TODO: implement a ‘status' field for ModelService as a better place to store info instead of Description
			Name:        input.Name,
			Namespace:   input.Namespace,
			APIType:     input.APIType,
			Description: &info,
		}, nil
	}
	return nil, ErrNoAuthProvided
}

func checkOpenAI(ctx context.Context, c dynamic.Interface, input generated.CreateModelServiceInput) (string, error) {
	apiKey := input.Endpoint.Auth["apiKey"].(string)
	client, err := openai.NewOpenAI(apiKey, input.Endpoint.URL)
	if err != nil {
		return "", err
	}

	// TODO: able to validate openai models
	res, err := client.Validate(ctx, llms.WithModel(""))
	if err != nil {
		return "", err
	}
	return res.String(), nil
}

func checkZhipuAI(ctx context.Context, c dynamic.Interface, input generated.CreateModelServiceInput) (string, error) {
	apiKey := input.Endpoint.Auth["apiKey"].(string)
	client := zhipuai.NewZhiPuAI(apiKey)
	res, err := client.Validate(ctx)
	if err != nil {
		return "", err
	}
	return res.String(), nil
}
