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
package defaultobject

import (
	"time"

	"github.com/kubeagi/arcadia/graphql-server/go-server/graph/generated"
)

var (
	DefaultInt     = 0
	DefaultString  = ""
	DefaultTime    = time.Now()
	DefaultDataset = generated.Dataset{
		Labels:          map[string]interface{}{},
		Annotations:     map[string]interface{}{},
		Creator:         &DefaultString,
		UpdateTimestamp: &DefaultTime,
		Field:           &DefaultString,
	}

	DefaultVersioneddataset = generated.VersionedDataset{
		Labels:            map[string]interface{}{},
		Annotations:       map[string]interface{}{},
		Creator:           &DefaultString,
		UpdateTimestamp:   &DefaultTime,
		SyncStatus:        &DefaultString,
		DataProcessStatus: &DefaultString,
	}

	DefaultPaginatedResult = generated.PaginatedResult{
		HasNextPage: false,
		Nodes:       []generated.PageNode{},
		Page:        &DefaultInt,
		PageSize:    &DefaultInt,
		TotalCount:  0,
	}

	DefaultDatasource = generated.Datasource{
		Labels:      map[string]interface{}{},
		Annotations: map[string]interface{}{},
		Creator:     &DefaultString,
		Description: &DefaultString,
		Endpoint:    &generated.Endpoint{},
		Oss:         &generated.Oss{},
		Status:      &DefaultString,
		FileCount:   &DefaultInt,
	}

	DefaultEmbedder = generated.Embedder{
		Labels:          map[string]interface{}{},
		Annotations:     map[string]interface{}{},
		Creator:         &DefaultString,
		Description:     &DefaultString,
		Endpoint:        &generated.Endpoint{},
		ServiceType:     &DefaultString,
		UpdateTimestamp: &DefaultTime,
	}

	DefaultKnowledgebase = generated.KnowledgeBase{
		Labels:      map[string]interface{}{},
		Annotations: map[string]interface{}{},
		Creator:     &DefaultString,
		Description: &DefaultString,
		Embedder: &generated.TypedObjectReference{
			APIGroup:  &DefaultString,
			Namespace: &DefaultString,
		},
		VectorStore: &generated.TypedObjectReference{
			APIGroup:  &DefaultString,
			Namespace: &DefaultString,
		},
		FileGroups: []*generated.Filegroup{},
		Status:     &DefaultString,
	}

	DefaultModel = generated.Model{
		Labels:          map[string]interface{}{},
		Annotations:     map[string]interface{}{},
		Creator:         &DefaultString,
		Description:     &DefaultString,
		UpdateTimestamp: &DefaultTime,
	}
)
