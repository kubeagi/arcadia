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
	"strings"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"

	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/embedder"
	"github.com/kubeagi/arcadia/apiserver/pkg/llm"
	"github.com/kubeagi/arcadia/apiserver/pkg/worker"
	"github.com/kubeagi/arcadia/pkg/llms/openai"
	"github.com/kubeagi/arcadia/pkg/llms/zhipuai"
)

func CreateModelService(ctx context.Context, c dynamic.Interface, input generated.CreateModelServiceInput) (*generated.ModelService, error) {
	// - Get general info from input: displayName, description, types, name & namespace, etc.
	// - Create *generated.LLM, *generated.embedder accordingly
	// - Wrap all elements into *generated.ModelService
	displayName, description, serviceType, APIType := "", "", "", ""
	var genLLM *generated.Llm
	var genEmbed *generated.Embedder
	var creationTimestamp, updateTimestamp *time.Time
	var err error

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

	if strings.Contains(serviceType, "llm") {
		genLLM, err = llm.CreateLLM(ctx, c, generated.CreateLLMInput{
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
	}

	if strings.Contains(serviceType, "embedding") {
		genEmbed, err = embedder.CreateEmbedder(ctx, c, generated.CreateEmbedderInput{
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
	}

	if genLLM != nil {
		creationTimestamp = genLLM.CreationTimestamp
		updateTimestamp = genLLM.UpdateTimestamp
	} else if genEmbed != nil {
		creationTimestamp = genEmbed.CreationTimestamp
		updateTimestamp = genEmbed.UpdateTimestamp
	}

	ms := generated.ModelService{
		// fulfill all params
		// TBD: ID, Creator, Resource
		Name:              input.Name,
		Namespace:         input.Namespace,
		DisplayName:       &displayName,
		Description:       &description,
		Labels:            input.Labels,
		Annotations:       input.Annotations,
		Types:             &serviceType,
		APIType:           &APIType,
		CreationTimestamp: creationTimestamp,
		UpdateTimestamp:   updateTimestamp,
		LlmResource:       genLLM,
		EmbedderResource:  genEmbed,
	}
	return &ms, nil
}

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
		EmbedderResource:  updatedEmbedder,
		LlmResource:       updatedLLM,
		CreationTimestamp: creationTimestamp,
		UpdateTimestamp:   updateTimestamp,
	}
	return ds, nil
}

func DeleteModelService(ctx context.Context, c dynamic.Interface, input *generated.DeleteCommonInput) (*string, error) {
	var errText string
	_, err1 := embedder.DeleteEmbedders(ctx, c, input)
	if err1 != nil {
		errText += "embedder: " + err1.Error()
	}
	_, err2 := llm.DeleteLLMs(ctx, c, input)
	if err2 != nil {
		errText += " llm:" + err2.Error()
	}
	if errText != "" {
		return nil, errors.New("error occurred during deleting: " + errText)
	}
	return nil, nil
}

var (
	fixedPage = 1
	// because the data is to be paged, no parameters are provided,
	// and the default return is 10. Getting modelserve needs to get all the llms and embedding,
	// so a larger pageSize is provided here.
	fixedPageSize = 100
)

const (
	modelTypeAll       = "all"
	modelTypeLLM       = "llm"
	modelTypeEmbedding = "embedding"
)

func debugModelService(m *generated.ModelService) string {
	id := ""
	if m.ID != nil {
		id = *m.ID
	}

	creator := ""
	if m.Creator != nil {
		creator = *m.Creator
	}
	types := ""
	if m.Types != nil {
		types = *m.Types
	}
	return fmt.Sprintf("{id: %s, creator: %s, types: %s, apiType: %s, creationTimestamp: %s, updateTimestamp: %s}",
		id, creator, types, *m.APIType, m.CreationTimestamp, m.UpdateTimestamp)
}

func ListModelServices(ctx context.Context, c dynamic.Interface, input *generated.ListModelService) (*generated.PaginatedResult, error) {
	var (
		llmList, embedderList, workerList *generated.PaginatedResult
		err                               error
	)
	query := generated.ListCommonInput{Page: &fixedPage, PageSize: &fixedPageSize, Namespace: input.Namespace, Keyword: input.Keyword}

	// list all llms
	llmList, err = llm.ListLLMs(ctx, c, query)
	if err != nil {
		klog.Errorf("failed to list llm %s", err)
		return nil, err
	}

	// list all embedders
	embedderList, err = embedder.ListEmbedders(ctx, c, query)
	if err != nil {
		klog.Errorf("failed to list embedder %s", err)

		return nil, err
	}

	// list all workers
	workerList, err = worker.ListWorkers(ctx, c, generated.ListWorkerInput{Namespace: input.Namespace, Page: &fixedPage, PageSize: &fixedPageSize})
	if err != nil {
		klog.Errorf("failed to list worker %s", err)
		return nil, err
	}

	workerResource := make(map[string]*generated.Worker)
	for _, item := range workerList.Nodes {
		v := item.(*generated.Worker)
		workerResource[v.Name] = v
	}

	modelServiceList := make([]*generated.ModelService, 0)
	intersec := make(map[string]*generated.ModelService)
	klog.V(5).Infof("namespace: %s modetype: %s, providerType: %s, apiType: %s",
		input.Namespace, input.ModelType, input.ProviderType, input.APIType)

	// The overall idea is to get the llm-filtered list first when not filtering on modelType or when you want to filter on llm.
	// Or to filter embedder, first get the list of embedder.
	if input.ModelType == "" || input.ModelType == modelTypeAll || input.ModelType == modelTypeLLM {
		for idx := range llmList.Nodes {
			v := llmList.Nodes[idx].(*generated.Llm)
			klog.V(5).Infof("add llm modelservice: llm %s, type: %s, provider: %s. filter apiType: %s, filter providers: %s",
				v.Name, *v.Type, *v.Provider, input.APIType, input.ProviderType)
			if input.APIType != nil && *v.Type != *input.APIType {
				continue
			}
			if input.ProviderType != nil && *v.Provider != *input.ProviderType {
				continue
			}

			ms := &generated.ModelService{
				Name:              v.Name,
				Namespace:         input.Namespace,
				Creator:           v.Creator,
				Description:       v.Description,
				Types:             new(string),
				CreationTimestamp: v.CreationTimestamp,
				UpdateTimestamp:   v.UpdateTimestamp,
				APIType:           v.Type,
				LlmResource:       v,
				EmbedderResource:  new(generated.Embedder),
				Resource:          new(generated.Resources),
			}
			*ms.Types = modelTypeLLM
			intersec[v.Name] = ms

			modelServiceList = append(modelServiceList, ms)
			klog.V(5).Infof("add llm modelservice: append only llm modelService to list: %s", debugModelService(ms))
			if r, ok := workerResource[v.Name]; ok {
				klog.V(5).Infof(" set llm modelservice: %s resource: set modelservice resource: %+v", v.Name, r)
				*ms.Resource = r.Resources
				ms.ID = r.ID
			}
		}
	} else {
		for idx := range embedderList.Nodes {
			v := embedderList.Nodes[idx].(*generated.Embedder)
			klog.V(5).Infof("add embedder modelservice: embedder %s, type: %s, provider: %s. filter apiType: %s, filter providers: %s",
				v.Name, *v.Type, *v.Provider, input.APIType, input.ProviderType)

			if input.APIType != nil && *v.Type != *input.APIType {
				continue
			}
			if input.ProviderType != nil && *v.Provider != *input.ProviderType {
				continue
			}

			ms := &generated.ModelService{
				Name:              v.Name,
				Namespace:         input.Namespace,
				Creator:           v.Creator,
				Description:       v.Description,
				Types:             new(string),
				CreationTimestamp: v.CreationTimestamp,
				UpdateTimestamp:   v.UpdateTimestamp,
				APIType:           v.Type,
				EmbedderResource:  v,
				LlmResource:       new(generated.Llm),
				Resource:          new(generated.Resources),
			}
			*ms.Types = modelTypeEmbedding

			intersec[v.Name] = ms
			modelServiceList = append(modelServiceList, ms)
			klog.V(5).Infof("add embedder modelservice: append only embedder modelService to list: %s", debugModelService(ms))

			if r, ok := workerResource[v.Name]; ok {
				klog.V(5).Infof("set embedder modelservice: %s resource: %+v", v.Name, r)
				*ms.Resource = r.Resources
				ms.ID = r.ID
			}
		}
	}

	page, pageSize := 1, 10
	if input.Page != nil && *input.Page > 0 {
		page = *input.Page
	}
	if input.PageSize != nil && *input.PageSize > 0 {
		pageSize = *input.PageSize
	}

	// If we are filtering llm or embedding,
	// if the list obtained earlier is empty (e.g. llmList), then we don't need to judge the embeddings anymore.
	if len(modelServiceList) == 0 && input.ModelType != "" && input.ModelType != modelTypeAll {
		return &generated.PaginatedResult{
			HasNextPage: false,
			Nodes:       []generated.PageNode{},
			TotalCount:  0,
		}, nil
	}

	// After getting the list of llm's,
	// we need to determine if any embedder meets the filter condition and if any embedder can be merged into the modeservice of the llm.
	// If an embedder meets the filter condition, we need to determine whether to create a new modelservice or merge it into the modelservice of llm.
	// The following logic is the same.
	if input.ModelType == "" || input.ModelType == modelTypeAll || input.ModelType == modelTypeLLM {
		for idx := range embedderList.Nodes {
			v := embedderList.Nodes[idx].(*generated.Embedder)
			if input.APIType != nil && *v.Type != *input.APIType {
				continue
			}
			if input.ProviderType != nil && *v.Provider != *input.ProviderType {
				continue
			}
			llm, ok := intersec[v.Name]
			if !ok && input.ModelType != "" && input.ModelType != modelTypeAll {
				continue
			}
			if !ok || *llm.APIType != *v.Type || *llm.LlmResource.Provider != *v.Provider {
				if !ok {
					klog.V(5).Infof("match check llm: embedder %s has no matching llm's, add modelservice", v.Name)
				}
				if ok && (*llm.APIType != *v.Type || *llm.LlmResource.Provider != *v.Provider) {
					klog.V(5).Infof("match check llm: embedder %s type: %s, llm apiType: %s, llm provider: %s, embedder provider: %s. add modelservice",
						v.Name, *v.Type, *llm.APIType, *llm.LlmResource.Provider, *v.Provider)
				}

				ms := &generated.ModelService{
					Name:              v.Name,
					Namespace:         input.Namespace,
					Creator:           v.Creator,
					Description:       v.Description,
					Types:             new(string),
					CreationTimestamp: v.CreationTimestamp,
					UpdateTimestamp:   v.UpdateTimestamp,
					APIType:           v.Type,
					LlmResource:       new(generated.Llm),
					EmbedderResource:  v,
					Resource:          new(generated.Resources),
				}
				*ms.Types = modelTypeEmbedding
				if r, ok := workerResource[v.Name]; ok {
					klog.V(5).Infof("match check llm: set modelservice %s resource: %+v", v.Name, r)
					*ms.Resource = r.Resources
					ms.ID = r.ID
				}

				klog.V(5).Infof("match check llm: append only embedder modelService to list: %s", debugModelService(ms))
				modelServiceList = append(modelServiceList, ms)
				continue
			}

			*llm.Types = modelTypeLLM + "," + modelTypeEmbedding
			llm.EmbedderResource = v
			if llm.CreationTimestamp.After(*v.CreationTimestamp) {
				llm.CreationTimestamp = v.CreationTimestamp
			}
			if llm.UpdateTimestamp.Before(*v.UpdateTimestamp) {
				llm.UpdateTimestamp = v.UpdateTimestamp
			}
			klog.V(5).Infof("match check llm: embedder match llm %s", debugModelService(llm))
		}
	} else {
		for idx := range llmList.Nodes {
			v := llmList.Nodes[idx].(*generated.Llm)
			if input.APIType != nil && *v.Type != *input.APIType {
				continue
			}
			if input.ProviderType != nil && *v.Provider != *input.ProviderType {
				continue
			}

			embedder, ok := intersec[v.Name]
			if !ok && input.ModelType != "" && input.ModelType != modelTypeAll {
				continue
			}

			if !ok || embedder.Name != v.Name || *embedder.APIType != *v.Type || *embedder.EmbedderResource.Provider != *v.Provider {
				if !ok {
					klog.V(5).Infof("match check embedder: llm %s has no matching embedder's, add modelservice", v.Name)
				}
				if ok && (*embedder.APIType != *v.Type || *embedder.EmbedderResource.Provider != *v.Provider) {
					klog.V(5).Infof("match check embedder: llm %s type: %s, embedder apiType: %s, embedder provider: %s, llm provider: %s. add modelservice",
						v.Name, *v.Type, *embedder.APIType, *embedder.EmbedderResource.Provider, *v.Provider)
				}

				ms := &generated.ModelService{
					Name:              v.Name,
					Namespace:         input.Namespace,
					Creator:           v.Creator,
					Description:       v.Description,
					Types:             new(string),
					CreationTimestamp: v.CreationTimestamp,
					UpdateTimestamp:   v.UpdateTimestamp,
					APIType:           v.Type,
					LlmResource:       v,
					EmbedderResource:  new(generated.Embedder),
					Resource:          new(generated.Resources),
				}
				*ms.Types = modelTypeLLM
				if r, ok := workerResource[v.Name]; ok {
					klog.V(5).Infof("match check embedder: set modelservice %s resource: %+v", v.Name, r)
					*ms.Resource = r.Resources
					ms.ID = v.ID
				}
				klog.V(5).Infof("match check embedder: append only llm modelService to list: %s", debugModelService(ms))
				modelServiceList = append(modelServiceList, ms)
				continue
			}

			*embedder.Types = modelTypeLLM + "," + modelTypeEmbedding
			embedder.LlmResource = v
			if embedder.CreationTimestamp.After(*v.CreationTimestamp) {
				embedder.CreationTimestamp = v.CreationTimestamp
			}
			if embedder.UpdateTimestamp.Before(*v.UpdateTimestamp) {
				embedder.UpdateTimestamp = v.UpdateTimestamp
			}
			klog.V(5).Infof("match check llm: embedder match llm %s", debugModelService(embedder))
		}
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	total := len(modelServiceList)

	result := pageModelService(start, end, &modelServiceList)
	nodes := make([]generated.PageNode, len(result))
	for idx := range result {
		nodes[idx] = result[idx]
	}
	return &generated.PaginatedResult{
		HasNextPage: end < total,
		Nodes:       nodes,
		Page:        &page,
		PageSize:    &pageSize,
		TotalCount:  total,
	}, nil
}

func GetModelService(ctx context.Context, c dynamic.Interface, name, namespace, apiType string) (*generated.ModelService, error) {
	ms := &generated.ModelService{
		ID:                new(string),
		Name:              name,
		Namespace:         namespace,
		Creator:           new(string),
		Description:       new(string),
		Types:             new(string),
		CreationTimestamp: new(time.Time),
		UpdateTimestamp:   new(time.Time),
		APIType:           &apiType,
		LlmResource:       new(generated.Llm),
		EmbedderResource:  new(generated.Embedder),
		Resource:          new(generated.Resources),
	}
	exist := false
	if r1, err := llm.ReadLLM(ctx, c, name, namespace); err == nil && *r1.Type == apiType {
		exist = true
		if r1.CreationTimestamp != nil {
			*ms.CreationTimestamp = *r1.CreationTimestamp
		}
		if r1.UpdateTimestamp != nil {
			*ms.UpdateTimestamp = *r1.UpdateTimestamp
		}
		*ms.Types = "llm"
		if r1.Description != nil {
			*ms.Description = *r1.Description
		}
		*ms.LlmResource = *r1
	}

	if r2, err := embedder.ReadEmbedder(ctx, c, name, namespace); err == nil && *r2.Type == apiType {
		exist = true
		if r2.CreationTimestamp != nil && (ms.CreationTimestamp == nil || r2.CreationTimestamp.Before(*ms.CreationTimestamp)) {
			*ms.CreationTimestamp = *r2.CreationTimestamp
		}
		if r2.UpdateTimestamp != nil && (ms.UpdateTimestamp == nil || r2.UpdateTimestamp.After(*ms.UpdateTimestamp)) {
			*ms.UpdateTimestamp = *r2.UpdateTimestamp
		}
		if *ms.Description == "" && r2.Description != nil {
			*ms.Description = *r2.Description
		}
		if *ms.Types == modelTypeLLM {
			*ms.Types += "," + modelTypeEmbedding
		} else {
			*ms.Types = modelTypeEmbedding
		}
		*ms.EmbedderResource = *r2
	}

	if r3, err := worker.ReadWorker(ctx, c, name, namespace); err == nil {
		*ms.Resource = r3.Resources
		*ms.ID = *r3.ID
	}
	if !exist {
		return nil, fmt.Errorf("not found modelService %s", name)
	}
	return ms, nil
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
	client := openai.NewOpenAI(apiKey, input.Endpoint.URL)
	res, err := client.Validate()
	if err != nil {
		return "", err
	}
	return res.String(), nil
}

func checkZhipuAI(ctx context.Context, c dynamic.Interface, input generated.CreateModelServiceInput) (string, error) {
	apiKey := input.Endpoint.Auth["apiKey"].(string)
	client := zhipuai.NewZhiPuAI(apiKey)
	res, err := client.Validate()
	if err != nil {
		return "", err
	}
	return res.String(), nil
}
