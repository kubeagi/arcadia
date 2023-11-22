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
		Name:        DefaultString,
		Namespace:   DefaultString,
		DisplayName: DefaultString,
		ContentType: DefaultString,
		Versions: generated.PaginatedResult{
			HasNextPage: false,
			TotalCount:  DefaultInt,
		},
		VersionCount: DefaultInt,
	}

	DefaultVersioneddataset = generated.VersionedDataset{
		Name:        DefaultString,
		Namespace:   DefaultString,
		DisplayName: DefaultString,
		Dataset: generated.TypedObjectReference{
			Kind: DefaultString,
			Name: DefaultString,
		},
		CreationTimestamp: DefaultTime,
		Files: generated.PaginatedResult{
			HasNextPage: false,
			TotalCount:  DefaultInt,
		},
		Version:   DefaultString,
		FileCount: DefaultInt,
		Released:  DefaultInt,
	}

	DefaultPaginatedResult = generated.PaginatedResult{
		HasNextPage: false,
		Nodes:       []generated.PageNode{},
		TotalCount:  0,
	}

	DefaultDatasource = generated.Datasource{
		Name:            DefaultString,
		Namespace:       DefaultString,
		DisplayName:     DefaultString,
		UpdateTimestamp: DefaultTime,
	}

	DefaultEmbedder = generated.Embedder{
		Name:        DefaultString,
		Namespace:   DefaultString,
		DisplayName: DefaultString,
	}

	DefaultKnowledgebase = generated.KnowledgeBase{
		Name:            DefaultString,
		Namespace:       DefaultString,
		DisplayName:     DefaultString,
		UpdateTimestamp: DefaultTime,
	}

	DefaultModel = generated.Model{
		Name:        DefaultString,
		Namespace:   DefaultString,
		DisplayName: DefaultString,
		Modeltypes:  DefaultString,
	}
)
