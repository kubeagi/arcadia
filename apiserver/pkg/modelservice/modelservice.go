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
	"strings"
	"time"

	"k8s.io/client-go/dynamic"

	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/embedder"
	"github.com/kubeagi/arcadia/apiserver/pkg/llm"
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
		// TBD: ID, Creator
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
