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
	"time"

	"github.com/tmc/langchaingo/llms"
	"k8s.io/client-go/dynamic"

	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	"github.com/kubeagi/arcadia/apiserver/pkg/embedder"
	"github.com/kubeagi/arcadia/apiserver/pkg/llm"
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
		})
		if err != nil {
			return nil, err
		}
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
		})
		if err != nil {
			return nil, err
		}

		modelSerivce = Embedder2ModelService(embedder)
	}

	modelSerivce.Types = &serviceType

	return modelSerivce, nil
}

// UpdateModelService updates a 3rd_party model service
func UpdateModelService(ctx context.Context, c dynamic.Interface, input generated.UpdateModelServiceInput) (*generated.ModelService, error) {
	name, namespace, displayName := "", "", ""
	if input.Name != "" {
		name = input.Name
	}
	if input.Namespace != "" {
		namespace = input.Namespace
	}
	if input.DisplayName != nil {
		displayName = *input.DisplayName
	}

	updatedLLM, err := llm.UpdateLLM(ctx, c, name, namespace, displayName)
	if err != nil {
		return nil, err
	}

	updatedEmbedder, err := embedder.UpdateEmbedder(ctx, c, name, namespace, displayName)
	if err != nil {
		return nil, err
	}

	var creationTimestamp, updateTimestamp *time.Time

	if updatedLLM != nil {
		creationTimestamp = updatedLLM.CreationTimestamp
		updateTimestamp = updatedLLM.UpdateTimestamp
	} else if updatedEmbedder != nil {
		creationTimestamp = updatedLLM.CreationTimestamp
		updateTimestamp = updatedLLM.UpdateTimestamp
	}

	ds := &generated.ModelService{
		Name:              input.Name,
		Namespace:         input.Namespace,
		DisplayName:       input.DisplayName,
		Description:       input.Description,
		Labels:            input.Labels,
		Annotations:       input.Annotations,
		Types:             input.Types,
		APIType:           input.APIType,
		CreationTimestamp: creationTimestamp,
		UpdateTimestamp:   updateTimestamp,
	}
	return ds, nil
}

// DeleteModelService deletes a 3rd_party model service
func DeleteModelService(ctx context.Context, c dynamic.Interface, input *generated.DeleteCommonInput) (*string, error) {
	_, err := embedder.DeleteEmbedders(ctx, c, input)
	if err != nil {
		return nil, err
	}
	_, err = llm.DeleteLLMs(ctx, c, input)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// GetModelService get a 3rd_party model service
func ReadModelService(ctx context.Context, c dynamic.Interface, name string, namespace string) (*generated.ModelService, error) {
	var modelService = &generated.ModelService{}

	llm, err := llm.ReadLLM(ctx, c, name, namespace)
	if err == nil {
		modelService = LLM2ModelService(llm)
	}
	embedder, err := embedder.ReadEmbedder(ctx, c, name, namespace)
	if err == nil {
		modelService = Embedder2ModelService(embedder)
	}

	if llm != nil && embedder != nil {
		modelService.Types = &common.ModelTypeAll
	}

	return modelService, nil
}

// ListModelServices based on input
func ListModelServices(ctx context.Context, c dynamic.Interface, input *generated.ListModelServiceInput) (*generated.PaginatedResult, error) {
	// use `UnlimitedPageSize` so we can get all llms and embeddings
	query := generated.ListCommonInput{Page: input.Page, PageSize: &common.UnlimitedPageSize, Namespace: input.Namespace, Keyword: input.Keyword}

	// list all llms
	llmList, err := llm.ListLLMs(ctx, c, query, common.WithPageNodeConvertFunc(func(a any) generated.PageNode {
		llm, ok := a.(*generated.Llm)
		if !ok {
			return nil
		}
		// convert llm to modelserivce
		return LLM2ModelService(llm)
	}))
	if err != nil {
		return nil, err
	}

	// list all embedders
	embedderList, err := embedder.ListEmbedders(ctx, c, query, common.WithPageNodeConvertFunc(func(a any) generated.PageNode {
		embedder, ok := a.(*generated.Embedder)
		if !ok {
			return nil
		}
		// convert embedder to modelserivce
		return Embedder2ModelService(embedder)
	}))
	if err != nil {
		return nil, err
	}

	// serviceList keeps all model services with combined
	serviceMapList := make(map[string]*generated.ModelService)
	for _, node := range append(llmList.Nodes, embedderList.Nodes...) {
		ms, _ := node.(*generated.ModelService)
		curr, ok := serviceMapList[ms.Name]
		// if llm & embedder has same name,we treat it as `ModelTypeAll(llm,embedding)`
		if ok {
			ms.Types = &common.ModelTypeAll
			// combine models provided by this model service
			ms.LlmModels = append(ms.LlmModels, curr.LlmModels...)
			ms.EmbeddingModels = append(ms.EmbeddingModels, curr.EmbeddingModels...)
		}
		serviceMapList[ms.Name] = ms
	}

	// newNodeList
	newNodeList := make([]*generated.ModelService, 0, len(serviceMapList))
	for _, node := range serviceMapList {
		newNodeList = append(newNodeList, node)
	}
	// sort by creation timestamp
	sort.Slice(newNodeList, func(i, j int) bool {
		return newNodeList[i].CreationTimestamp.After(*newNodeList[j].CreationTimestamp)
	})

	// return ModelService with the actual Page and PageSize
	page, pageSize := 1, 10
	if input.Page != nil && *input.Page > 0 {
		page = *input.Page
	}
	if input.PageSize != nil && *input.PageSize > 0 {
		pageSize = *input.PageSize
	}

	var totalCount int

	result := make([]generated.PageNode, 0, pageSize)
	pageStart := (page - 1) * pageSize

	// index is the actual result length
	var index int
	for _, service := range newNodeList {
		// Add filter conditions here
		// 1. filter service types: llm or embedding or both
		if input.Types != nil && *input.Types != "" {
			if !strings.Contains(*service.Types, *input.Types) {
				continue
			}
		}
		// 2. filter provider type: worker or 3rd_party
		if input.ProviderType != nil && *input.ProviderType != "" {
			if service.ProviderType == nil || *service.ProviderType != *input.ProviderType {
				continue
			}
		}
		// 3. filter api type: openai or zhipuai
		if input.APIType != nil && *input.APIType != "" {
			if service.APIType == nil || *service.APIType != *input.APIType {
				continue
			}
		}

		// increase totalCount when service meets the filter conditions
		totalCount++

		// append result
		if index >= pageStart && len(result) < pageSize {
			result = append(result, service)
		}

		index++
	}

	end := page * pageSize
	if end > totalCount {
		end = totalCount
	}

	return &generated.PaginatedResult{
		TotalCount:  totalCount,
		HasNextPage: end < totalCount,
		Nodes:       result,
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
