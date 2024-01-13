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

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
)

type (
	ResourceFilter    func(*unstructured.Unstructured) bool
	ResourceConverter func(*unstructured.Unstructured) (generated.PageNode, error)
)

func FilterByName(name string) ResourceFilter {
	return func(u *unstructured.Unstructured) bool {
		return u.GetName() == name
	}
}

// Dataset Filter
func FilterDatasetByDisplayName(displayName string) ResourceFilter {
	return func(u *unstructured.Unstructured) bool {
		ds := v1alpha1.Dataset{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &ds); err != nil {
			return false
		}
		return ds.Spec.DisplayName == displayName
	}
}

func FilterDatasetByKeyword(keyword string) ResourceFilter {
	return func(u *unstructured.Unstructured) bool {
		ds := v1alpha1.Dataset{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &ds); err != nil {
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
	return func(u *unstructured.Unstructured) bool {
		displayName, _, _ := unstructured.NestedString(u.Object, "spec", "displayName")
		return strings.Contains(displayName, keyword) || strings.Contains(u.GetName(), keyword)
	}
}

// Datasource
func FilterDatasourceByKeyword(keyword string) ResourceFilter {
	return func(u *unstructured.Unstructured) bool {
		datasource := v1alpha1.Datasource{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &datasource); err != nil {
			return false
		}
		return strings.Contains(datasource.Name, keyword) || strings.Contains(datasource.Spec.DisplayName, keyword)
	}
}

// Embedder
func FilterEmbedderByKeyword(keyword string) ResourceFilter {
	return func(u *unstructured.Unstructured) bool {
		embedder := v1alpha1.Embedder{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &embedder); err != nil {
			return false
		}
		return strings.Contains(embedder.Name, keyword) || strings.Contains(embedder.Spec.DisplayName, keyword)
	}
}

// KnowledgeBase
func FilterKnowledgeByName(name string) ResourceFilter {
	return func(u *unstructured.Unstructured) bool {
		kb := v1alpha1.KnowledgeBase{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &kb); err != nil {
			return false
		}
		return strings.Contains(kb.Name, name)
	}
}
func FilterKnowledgeByDisplayName(displayName string) ResourceFilter {
	return func(u *unstructured.Unstructured) bool {
		kb := v1alpha1.KnowledgeBase{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &kb); err != nil {
			return false
		}
		return strings.Contains(kb.Spec.DisplayName, displayName)
	}
}

// LLM
func FilterLLMByKeyword(keyword string) ResourceFilter {
	return func(u *unstructured.Unstructured) bool {
		llm := v1alpha1.LLM{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &llm); err != nil {
			return false
		}
		return strings.Contains(llm.Name, keyword) || strings.Contains(llm.Spec.DisplayName, keyword)
	}
}

// Model
func FilterModelByKeyword(keyword string) ResourceFilter {
	return func(u *unstructured.Unstructured) bool {
		model := v1alpha1.Model{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &model); err != nil {
			return false
		}
		return strings.Contains(model.Name, keyword) || strings.Contains(model.Spec.DisplayName, keyword)
	}
}

// VersionedData
func FilterVersionedDatasetByDisplayName(displayName string) ResourceFilter {
	return func(u *unstructured.Unstructured) bool {
		v := v1alpha1.VersionedDataset{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &v); err != nil {
			return false
		}
		return v.Spec.DisplayName == displayName
	}
}

func FilterVersionedDatasetByKeyword(keyword string) ResourceFilter {
	return func(u *unstructured.Unstructured) bool {
		v := v1alpha1.VersionedDataset{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &v); err != nil {
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
	return func(u *unstructured.Unstructured) bool {
		w := v1alpha1.Worker{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &w); err != nil {
			return false
		}
		return strings.Contains(w.Name, keyword) || strings.Contains(w.Spec.DisplayName, keyword)
	}
}

func FilterWorkerByType(c dynamic.Interface, namespace, modelType string) ResourceFilter {
	cache := make(map[string]string)
	models, err := c.Resource(SchemaOf(&ArcadiaAPIGroup, "model")).
		Namespace(namespace).List(context.Background(), v1.ListOptions{})
	if err == nil {
		for _, m := range models.Items {
			types, _, _ := unstructured.NestedString(m.Object, "spec", "types")
			cache[m.GetName()] = types
		}
	}
	return func(u *unstructured.Unstructured) bool {
		w := v1alpha1.Worker{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &w); err != nil {
			return false
		}
		// TODO: how do we do if the model and worek have different namespace?
		return strings.Contains(cache[w.Spec.Model.Name], modelType)
	}
}
