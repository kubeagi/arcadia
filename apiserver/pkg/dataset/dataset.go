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

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	"github.com/kubeagi/arcadia/apiserver/pkg/utils"
)

func dataset2modelConverter(obj *unstructured.Unstructured) (generated.PageNode, error) {
	return dataset2model(obj)
}

func dataset2model(obj *unstructured.Unstructured) (*generated.Dataset, error) {
	ds := &generated.Dataset{}
	ds.Name = obj.GetName()
	ds.Namespace = obj.GetNamespace()
	n := obj.GetCreationTimestamp()
	ds.CreationTimestamp = &n.Time
	ds.UpdateTimestamp = &n.Time
	ds.Labels = utils.MapStr2Any(obj.GetLabels())
	ds.Annotations = utils.MapStr2Any(obj.GetAnnotations())
	dataset := &v1alpha1.Dataset{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, dataset); err != nil {
		return nil, err
	}
	ds.Creator = &dataset.Spec.Creator
	ds.DisplayName = &dataset.Spec.DisplayName
	ds.ContentType = dataset.Spec.ContentType
	ds.Field = &dataset.Spec.Field
	first := true
	for _, cond := range dataset.Status.Conditions {
		if !cond.LastSuccessfulTime.IsZero() {
			if first || ds.UpdateTimestamp.Before(cond.LastTransitionTime.Time) {
				ds.UpdateTimestamp = &cond.LastTransitionTime.Time
				first = false
			}
		}
	}
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
		},
	}
	if input.Filed != nil {
		dataset.Spec.Field = *input.Filed
	}
	if input.DisplayName != nil {
		dataset.Spec.DisplayName = *input.DisplayName
	}
	if input.Description != nil {
		dataset.Spec.Description = *input.Description
	}
	dataset.Labels = utils.MapAny2Str(input.Labels)
	dataset.Annotations = utils.MapAny2Str(input.Annotations)
	common.SetCreator(ctx, &dataset.Spec.CommonSpec)

	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(dataset)
	if err != nil {
		return nil, err
	}
	obj, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Dataset")).Namespace(input.Namespace).Create(ctx, &unstructured.Unstructured{Object: u}, v1.CreateOptions{})
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
	datastList, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Dataset")).Namespace(input.Namespace).List(ctx, listOptions)
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
	filters := make([]common.ResourceFilter, 0)
	if input.DisplayName != nil {
		filters = append(filters, common.FilterDatasetByDisplayName(*input.DisplayName))
	}
	if input.Keyword != nil {
		filters = append(filters, common.FilterDatasetByKeyword(*input.Keyword))
	}
	return common.ListReources(datastList, page, size, dataset2modelConverter, filters...)
}

func UpdateDataset(ctx context.Context, c dynamic.Interface, input *generated.UpdateDatasetInput) (*generated.Dataset, error) {
	obj, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Dataset")).Namespace(input.Namespace).Get(ctx, input.Name, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	obj.SetLabels(utils.MapAny2Str(input.Labels))
	obj.SetAnnotations(utils.MapAny2Str(input.Annotations))
	displayname, _, _ := unstructured.NestedString(obj.Object, "spec", "displayName")
	description, _, _ := unstructured.NestedString(obj.Object, "spec", "description")
	if input.DisplayName != nil && *input.DisplayName != displayname {
		_ = unstructured.SetNestedField(obj.Object, input.DisplayName, "spec", "displayName")
	}
	if input.Description != nil && *input.Description != description {
		_ = unstructured.SetNestedField(obj.Object, *input.Description, "spec", "description")
	}
	obj, err = c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Dataset")).Namespace(input.Namespace).Update(ctx, obj, v1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return dataset2model(obj)
}

func DeleteDatasets(ctx context.Context, c dynamic.Interface, input *generated.DeleteCommonInput) (*string, error) {
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
	err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Dataset")).Namespace(input.Namespace).DeleteCollection(ctx, v1.DeleteOptions{}, listOptions)
	return &none, err
}

func GetDataset(ctx context.Context, c dynamic.Interface, name, namespace string) (*generated.Dataset, error) {
	obj, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Dataset")).Namespace(namespace).Get(ctx, name, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return dataset2model(obj)
}
