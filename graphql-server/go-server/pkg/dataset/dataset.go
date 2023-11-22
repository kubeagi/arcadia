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

package dataset

import (
	"context"
	"fmt"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kubeagi/arcadia/api/v1alpha1"
	"github.com/kubeagi/arcadia/graphql-server/go-server/graph/generated"
)

var datasetSchema = schema.GroupVersionResource{
	Group:    v1alpha1.GroupVersion.Group,
	Version:  v1alpha1.GroupVersion.Version,
	Resource: "datasets",
}

func dataset2model(obj *unstructured.Unstructured) (*generated.Dataset, error) {
	ds := &generated.Dataset{}
	ds.Name = obj.GetName()
	ds.Namespace = obj.GetNamespace()
	if r := obj.GetLabels(); len(r) > 0 {
		l := make(map[string]any)
		for k, v := range r {
			l[k] = v
		}
		ds.Labels = l
	}
	if r := obj.GetAnnotations(); len(r) > 0 {
		a := make(map[string]any)
		for k, v := range r {
			a[k] = v
		}
		ds.Annotations = a
	}
	dataset := &v1alpha1.Dataset{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, dataset); err != nil {
		return nil, err
	}
	ds.Creator = &dataset.Spec.Creator
	ds.DisplayName = dataset.Spec.DisplayName
	ds.ContentType = dataset.Spec.ContentType
	ds.Field = &dataset.Spec.Field
	return ds, nil
}

func CreateDataset(ctx context.Context, c dynamic.Interface, input *generated.CreateDatasetInput) (*generated.Dataset, error) {
	dataset := &v1alpha1.Dataset{
		ObjectMeta: v1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
		},
		TypeMeta: v1.TypeMeta{
			Kind:       "Dataset",
			APIVersion: v1alpha1.GroupVersion.String(),
		},
		Spec: v1alpha1.DatasetSpec{
			ContentType: input.ContentType,
			CommonSpec: v1alpha1.CommonSpec{
				DisplayName: input.DisplayName,
			},
		},
	}
	if input.Description != nil {
		dataset.Spec.Description = *input.Description
	}
	if len(input.Labels) > 0 {
		l := make(map[string]string)
		for k, v := range input.Labels {
			l[k] = v.(string)
		}
		dataset.Labels = l
	}
	if len(input.Annotations) > 0 {
		a := make(map[string]string)
		for k, v := range input.Annotations {
			a[k] = v.(string)
		}
		dataset.Annotations = a
	}

	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(dataset)
	if err != nil {
		return nil, err
	}
	obj, err := c.Resource(datasetSchema).Namespace(input.Namespace).Create(ctx, &unstructured.Unstructured{Object: u}, v1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return dataset2model(obj)
}

func ListDatasets(ctx context.Context, c dynamic.Interface, input *generated.ListDatasetInput) (*generated.PaginatedResult, error) {
	listOptions := v1.ListOptions{}
	if input.Name != nil {
		listOptions.FieldSelector = fmt.Sprintf("metadata.name=%s", *input.Name)
	} else {
		if input.LabelSelector != nil {
			listOptions.LabelSelector = *input.LabelSelector
		}
		if input.FieldSelector != nil {
			listOptions.FieldSelector = *input.FieldSelector
		}
	}
	datastList, err := c.Resource(datasetSchema).Namespace(input.Namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}
	page, size := 1, 10
	if input.Page != nil && *input.Page > 0 {
		page = *input.Page
	}
	if input.PageSize != nil && *input.PageSize > 0 {
		size = *input.PageSize
	}
	result := make([]generated.PageNode, 0)
	for _, u := range datastList.Items {
		uu, _ := dataset2model(&u)
		if input.DisplayName != nil && uu.DisplayName != *input.DisplayName {
			continue
		}
		if input.Keyword != nil {
			ok := false
			if strings.Contains(uu.Name, *input.Keyword) {
				ok = true
			}
			if strings.Contains(uu.Namespace, *input.Keyword) {
				ok = true
			}
			if strings.Contains(uu.DisplayName, *input.Keyword) {
				ok = true
			}
			if strings.Contains(uu.ContentType, *input.Keyword) {
				ok = true
			}
			for _, v := range uu.Annotations {
				if strings.Contains(v.(string), *input.Keyword) {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		result = append(result, uu)
	}
	total := len(result)
	end := page * size
	if end > total {
		end = total
	}
	return &generated.PaginatedResult{
		TotalCount:  total,
		HasNextPage: end < total,
		Nodes:       result[(page-1)*size : end],
		Page:        &page,
		PageSize:    &size,
	}, nil
}

func UpdateDataset(ctx context.Context, c dynamic.Interface, input *generated.UpdateDatasetInput) (*generated.Dataset, error) {
	obj, err := c.Resource(datasetSchema).Namespace(input.Namespace).Get(ctx, input.Name, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	l := make(map[string]string)
	for k, v := range input.Labels {
		l[k] = v.(string)
	}
	a := make(map[string]string)
	for k, v := range input.Annotations {
		a[k] = v.(string)
	}
	obj.SetLabels(l)
	obj.SetAnnotations(a)
	displayname, _, _ := unstructured.NestedString(obj.Object, "spec", "displayName")
	description, _, _ := unstructured.NestedString(obj.Object, "spec", "description")
	if input.DisplayName != nil && *input.DisplayName != displayname {
		_ = unstructured.SetNestedField(obj.Object, input.DisplayName, "spec", "displayName")
	}
	if input.Description != nil && *input.Description != description {
		_ = unstructured.SetNestedField(obj.Object, *input.Description, "spec", "description")
	}
	obj, err = c.Resource(datasetSchema).Namespace(input.Namespace).Update(ctx, obj, v1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return dataset2model(obj)
}

func DeleteDatasets(ctx context.Context, c dynamic.Interface, input *generated.DeleteDatasetInput) (*string, error) {
	none := ""
	listOptions := v1.ListOptions{}
	if input.Name == nil && input.LabelSelector == nil && input.FieldSelector == nil {
		return &none, fmt.Errorf("no name, no labelselector, no fieldsleector, i don't know which one to delete")
	}
	if input.Name != nil {
		listOptions.FieldSelector = fmt.Sprintf("metadata.name=%s", *input.Name)
	} else {
		if input.LabelSelector != nil {
			listOptions.LabelSelector = *input.LabelSelector
		}
		if input.FieldSelector != nil {
			listOptions.FieldSelector = *input.FieldSelector
		}
	}
	err := c.Resource(datasetSchema).Namespace(input.Namespace).DeleteCollection(ctx, v1.DeleteOptions{}, listOptions)
	return &none, err
}

func GetDataset(ctx context.Context, c dynamic.Interface, name, namespace string) (*generated.Dataset, error) {
	obj, err := c.Resource(datasetSchema).Namespace(namespace).Get(ctx, name, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return dataset2model(obj)
}
