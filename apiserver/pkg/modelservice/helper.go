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
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	"github.com/kubeagi/arcadia/pkg/llms"
)

// Embedder2ModelService convert unstructured `CR Embedder` to graphql model `ModelService`
func Embedder2ModelService(embedder *generated.Embedder) *generated.ModelService {
	ms := &generated.ModelService{
		// metadata
		ID:                embedder.ID,
		Name:              embedder.Name,
		Namespace:         embedder.Namespace,
		CreationTimestamp: embedder.CreationTimestamp,
		UpdateTimestamp:   embedder.UpdateTimestamp,
		// common
		Creator:     embedder.Creator,
		DisplayName: embedder.DisplayName,
		Description: embedder.Description,
		// ProviderType: worker or 3rd_party
		ProviderType: embedder.Provider,
		// Model types: llm or embedding
		Types: &common.ModelTypeEmbedding,
		// APIType of this modelservice
		APIType: embedder.Type,
		// BaseURL of this Embedder
		BaseURL: embedder.BaseURL,
		// EmbeddingModels of this modelservice
		EmbeddingModels: embedder.Models,
		// Statuds of this model service
		Status:  embedder.Status,
		Message: embedder.Message,
	}
	return ms
}

// LLM2ModelService convert unstructured `CR LLM` to graphql model `ModelService`
func LLM2ModelService(llm *generated.Llm) *generated.ModelService {
	ms := &generated.ModelService{
		// metadata
		ID:                llm.ID,
		Name:              llm.Name,
		Namespace:         llm.Namespace,
		CreationTimestamp: llm.CreationTimestamp,
		UpdateTimestamp:   llm.UpdateTimestamp,
		// common
		Creator:     llm.Creator,
		DisplayName: llm.DisplayName,
		Description: llm.Description,
		// ProviderType: worker or 3rd_party
		ProviderType: llm.Provider,
		// Model types: llm or embedding
		Types: &common.ModelTypeLLM,
		// APIType of this modelservice
		APIType: llm.Type,
		// BaseURL of this modelservice
		BaseURL: llm.BaseURL,
		// EmbeddingModels of this modelservice
		LlmModels: llm.Models,
		// Statuds of this model service
		Status:  llm.Status,
		Message: llm.Message,
	}
	return ms
}

func Worker2ModelService(worker *generated.Worker) *generated.ModelService {
	ms := &generated.ModelService{
		ID:                worker.ID,
		Name:              worker.Name,
		Namespace:         worker.Namespace,
		CreationTimestamp: worker.CreationTimestamp,
		UpdateTimestamp:   worker.UpdateTimestamp,
		Creator:           worker.Creator,
		DisplayName:       worker.DisplayName,
		Description:       worker.Description,
		ProviderType:      new(string),
		Types:             new(string),
		APIType:           new(string),
		EmbeddingModels:   []string{*worker.ID},
		LlmModels:         []string{*worker.ID},
		Status:            worker.Status,
		Message:           worker.Message,
	}

	*ms.ProviderType = "worker"
	*ms.Types = worker.ModelTypes
	*ms.APIType = string(llms.OpenAI)
	return ms
}
