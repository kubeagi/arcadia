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
	"errors"
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	"github.com/kubeagi/arcadia/apiserver/pkg/utils"
)

func dataset2modelConverter(obj client.Object) (generated.PageNode, error) {
	dataset, ok := obj.(*v1alpha1.Dataset)
	if !ok {
		return nil, errors.New("convert client.Object to Dataset err")
	}
	return dataset2model(dataset)
}

func dataset2model(dataset *v1alpha1.Dataset) (*generated.Dataset, error) {
	ds := &generated.Dataset{}
	ds.Name = dataset.GetName()
	ds.Namespace = dataset.GetNamespace()
	n := dataset.GetCreationTimestamp()
	ds.CreationTimestamp = &n.Time
	ds.UpdateTimestamp = &n.Time
	ds.Labels = utils.MapStr2Any(dataset.GetLabels())
	ds.Annotations = utils.MapStr2Any(dataset.GetAnnotations())
	ds.Creator = &dataset.Spec.Creator
	ds.DisplayName = &dataset.Spec.DisplayName
	ds.ContentType = dataset.Spec.ContentType
	ds.Field = &dataset.Spec.Field
	first := true
	for _, cond := range dataset.Status.Conditions {
		cond := cond
		if !cond.LastSuccessfulTime.IsZero() {
			if first || ds.UpdateTimestamp.Before(cond.LastTransitionTime.Time) {
				ds.UpdateTimestamp = &cond.LastTransitionTime.Time
				first = false
			}
		}
	}
	return ds, nil
}

func CreateDataset(ctx context.Context, c client.Client, input *generated.CreateDatasetInput) (*generated.Dataset, error) {
	dataset := &v1alpha1.Dataset{
		ObjectMeta: v1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
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

	err := c.Create(ctx, dataset)
	if err != nil {
		return nil, err
	}
	return dataset2model(dataset)
}

func ListDatasets(ctx context.Context, c client.Client, input *generated.ListDatasetInput) (*generated.PaginatedResult, error) {
	listOptions := &client.ListOptions{Namespace: input.Namespace}
	if input.Name != nil {
		f, err := fields.ParseSelector(fmt.Sprintf("metadata.name=%s", *input.Name))
		if err != nil {
			return nil, err
		}
		listOptions.FieldSelector = f
	} else {
		if input.LabelSelector != nil {
			l, err := labels.Parse(*input.LabelSelector)
			if err != nil {
				return nil, err
			}
			listOptions.LabelSelector = l
		}
		if input.FieldSelector != nil {
			f, err := fields.ParseSelector(*input.FieldSelector)
			if err != nil {
				return nil, err
			}
			listOptions.FieldSelector = f
		}
	}
	datasetList := &v1alpha1.DatasetList{}
	err := c.List(ctx, datasetList, listOptions)
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
	items := make([]client.Object, len(datasetList.Items))
	for i := range datasetList.Items {
		items[i] = &datasetList.Items[i]
	}
	return common.ListReources(items, page, size, dataset2modelConverter, filters...)
}

func UpdateDataset(ctx context.Context, c client.Client, input *generated.UpdateDatasetInput) (*generated.Dataset, error) {
	dataset := &v1alpha1.Dataset{}
	err := c.Get(ctx, types.NamespacedName{Namespace: input.Namespace, Name: input.Name}, dataset)
	if err != nil {
		return nil, err
	}
	dataset.SetLabels(utils.MapAny2Str(input.Labels))
	dataset.SetAnnotations(utils.MapAny2Str(input.Annotations))
	displayname := dataset.Spec.DisplayName
	description := dataset.Spec.Description
	if input.DisplayName != nil && *input.DisplayName != displayname {
		dataset.Spec.DisplayName = *input.DisplayName
	}
	if input.Description != nil && *input.Description != description {
		dataset.Spec.Description = *input.Description
	}
	err = c.Update(ctx, dataset)
	if err != nil {
		return nil, err
	}
	return dataset2model(dataset)
}

func DeleteDatasets(ctx context.Context, c client.Client, input *generated.DeleteCommonInput) (*string, error) {
	opts, err := common.DeleteAllOptions(input)
	if err != nil {
		return nil, err
	}
	err = c.DeleteAllOf(ctx, &v1alpha1.Dataset{}, opts...)
	return nil, err
}

func GetDataset(ctx context.Context, c client.Client, name, namespace string) (*generated.Dataset, error) {
	dataset := &v1alpha1.Dataset{}
	err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, dataset)
	if err != nil {
		return nil, err
	}
	return dataset2model(dataset)
}
