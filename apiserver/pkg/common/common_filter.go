/*
Copyright 2024 KubeAGI.

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
package common

import (
	"context"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	evav1alpha1 "github.com/kubeagi/arcadia/api/evaluation/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
)

type (
	ResourceFilter    func(client.Object) bool
	ResourceConverter func(client.Object) (generated.PageNode, error)
)

func FilterByNameContains(name string) ResourceFilter {
	return func(u client.Object) bool {
		return strings.Contains(u.GetName(), name)
	}
}

// Dataset Filter
func FilterDatasetByDisplayName(displayName string) ResourceFilter {
	return func(u client.Object) bool {
		ds, ok := u.(*v1alpha1.Dataset)
		if !ok {
			return false
		}
		return ds.Spec.DisplayName == displayName
	}
}

func FilterDatasetByKeyword(keyword string) ResourceFilter {
	return func(u client.Object) bool {
		ds, ok := u.(*v1alpha1.Dataset)
		if !ok {
			return false
		}
		if strings.Contains(ds.Name, keyword) {
			return true
		}
		if strings.Contains(ds.Namespace, keyword) {
			return true
		}
		if strings.Contains(ds.Spec.DisplayName, keyword) {
			return true
		}
		if strings.Contains(ds.Spec.ContentType, keyword) {
			return true
		}
		for _, v := range ds.Annotations {
			if strings.Contains(v, keyword) {
				return true
			}
		}
		return false
	}
}

// Application
func FilterApplicationByKeyword(keyword string) ResourceFilter {
	return func(u client.Object) bool {
		app, ok := u.(*v1alpha1.Application)
		if !ok {
			return false
		}
		displayName := app.Spec.DisplayName
		return strings.Contains(displayName, keyword) || strings.Contains(u.GetName(), keyword)
	}
}

func FilterApplicationByCategory(category string) ResourceFilter {
	return func(u client.Object) bool {
		categoryStr, ok := u.GetLabels()[v1alpha1.AppCategoryLabelKey]
		if !ok {
			return false
		}
		return strings.Contains(categoryStr, category)
	}
}

// Datasource
func FilterDatasourceByKeyword(keyword string) ResourceFilter {
	return func(u client.Object) bool {
		datasource, ok := u.(*v1alpha1.Datasource)
		if !ok {
			return false
		}
		return strings.Contains(datasource.Name, keyword) || strings.Contains(datasource.Spec.DisplayName, keyword)
	}
}

// Embedder
func FilterEmbedderByKeyword(keyword string) ResourceFilter {
	return func(u client.Object) bool {
		embedder, ok := u.(*v1alpha1.Embedder)
		if !ok {
			return false
		}
		return strings.Contains(embedder.Name, keyword) || strings.Contains(embedder.Spec.DisplayName, keyword)
	}
}

// KnowledgeBase
func FilterKnowledgeByDisplayName(displayName string) ResourceFilter {
	return func(u client.Object) bool {
		kb, ok := u.(*v1alpha1.KnowledgeBase)
		if !ok {
			return false
		}
		return strings.Contains(kb.Spec.DisplayName, displayName)
	}
}

// LLM
func FilterLLMByKeyword(keyword string) ResourceFilter {
	return func(u client.Object) bool {
		llm, ok := u.(*v1alpha1.LLM)
		if !ok {
			return false
		}
		return strings.Contains(llm.Name, keyword) || strings.Contains(llm.Spec.DisplayName, keyword)
	}
}

// Model
func FilterModelByKeyword(keyword string) ResourceFilter {
	return func(u client.Object) bool {
		model, ok := u.(*v1alpha1.Model)
		if !ok {
			return false
		}
		return strings.Contains(model.Name, keyword) || strings.Contains(model.Spec.DisplayName, keyword)
	}
}

// VersionedData
func FilterVersionedDatasetByDisplayName(displayName string) ResourceFilter {
	return func(u client.Object) bool {
		v, ok := u.(*v1alpha1.VersionedDataset)
		if !ok {
			return false
		}
		return v.Spec.DisplayName == displayName
	}
}

func FilterVersionedDatasetByKeyword(keyword string) ResourceFilter {
	return func(u client.Object) bool {
		v, ok := u.(*v1alpha1.VersionedDataset)
		if !ok {
			return false
		}
		if strings.Contains(v.Name, keyword) {
			return true
		}
		if strings.Contains(v.Namespace, keyword) {
			return true
		}
		if strings.Contains(v.Spec.DisplayName, keyword) {
			return true
		}
		for _, vv := range v.Annotations {
			if strings.Contains(vv, keyword) {
				return true
			}
		}
		return false
	}
}

// Worekr
func FilterWorkerByKeyword(keyword string) ResourceFilter {
	return func(u client.Object) bool {
		w, ok := u.(*v1alpha1.Worker)
		if !ok {
			return false
		}
		return strings.Contains(w.Name, keyword) || strings.Contains(w.Spec.DisplayName, keyword)
	}
}

func FilterWorkerByType(c client.Client, namespace, modelType string) ResourceFilter {
	cache := make(map[string]string)
	models := &v1alpha1.ModelList{}
	err := c.List(context.Background(), models, client.InNamespace(namespace))
	if err == nil {
		for _, m := range models.Items {
			types := m.Spec.Types
			cache[m.GetName()] = types
		}
	}
	return func(u client.Object) bool {
		w, ok := u.(*v1alpha1.Worker)
		if !ok {
			return false
		}
		// TODO: how do we do if the model and worek have different namespace?
		return strings.Contains(cache[w.Spec.Model.Name], modelType)
	}
}

// RAG Filter

func FilterRAGByStatus(status string) ResourceFilter {
	return func(u client.Object) bool {
		rag, ok := u.(*evav1alpha1.RAG)
		if !ok {
			return false
		}
		ragStatus, _, _ := evav1alpha1.RagStatus(rag)
		return ragStatus == status
	}
}

func FilterByRAGKeyword(keyword string) ResourceFilter {
	return func(u client.Object) bool {
		rag, ok := u.(*evav1alpha1.RAG)
		if !ok {
			return false
		}
		return strings.Contains(rag.Name, keyword) ||
			strings.Contains(rag.Spec.DisplayName, keyword) ||
			strings.Contains(rag.Spec.Description, keyword)
	}
}
