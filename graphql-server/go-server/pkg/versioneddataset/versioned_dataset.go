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

package versioneddataset

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kubeagi/arcadia/api/v1alpha1"
	"github.com/kubeagi/arcadia/graphql-server/go-server/graph/generated"
	"github.com/kubeagi/arcadia/graphql-server/go-server/pkg/minio"
	"github.com/kubeagi/arcadia/pkg/utils/minioutils"
)

var (
	versioneddatasetSchem = schema.GroupVersionResource{
		Group:    v1alpha1.GroupVersion.Group,
		Version:  v1alpha1.GroupVersion.Version,
		Resource: "versioneddatasets",
	}
	dataCount = 0
)

func versionedDataset2model(obj *unstructured.Unstructured) (*generated.VersionedDataset, error) {
	vds := &generated.VersionedDataset{}
	vds.Name = obj.GetName()
	vds.Namespace = obj.GetNamespace()
	if r := obj.GetLabels(); len(r) > 0 {
		l := make(map[string]any)
		for k, v := range r {
			l[k] = v
		}
		vds.Labels = l
	}
	if r := obj.GetAnnotations(); len(r) > 0 {
		a := make(map[string]any)
		for k, v := range r {
			a[k] = v
		}
		vds.Annotations = a
	}
	versioneddataset := &v1alpha1.VersionedDataset{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, versioneddataset); err != nil {
		return nil, err
	}
	vds.CreationTimestamp = obj.GetCreationTimestamp().Time
	vds.Creator = &versioneddataset.Spec.Creator
	vds.DisplayName = versioneddataset.Spec.DisplayName
	vds.Description = &versioneddataset.Spec.Description
	vds.Dataset = generated.TypedObjectReference{
		APIGroup:  versioneddataset.Spec.Dataset.APIGroup,
		Kind:      versioneddataset.Spec.Dataset.Kind,
		Name:      versioneddataset.Spec.Dataset.Name,
		Namespace: versioneddataset.Spec.Dataset.Namespace,
	}
	now := time.Now()
	vds.UpdateTimestamp = &now

	vds.Version = versioneddataset.Spec.Version

	first := true
	for _, cond := range versioneddataset.Status.Conditions {
		if cond.Type == v1alpha1.TypeReady {
			syncStatus := string(cond.Reason)
			vds.SyncStatus = &syncStatus
		}
		if cond.Type == v1alpha1.TypeDataProcessing {
			dataProcessStatus := string(cond.Reason)
			vds.DataProcessStatus = &dataProcessStatus
		}
		if !cond.LastTransitionTime.IsZero() {
			if first || vds.UpdateTimestamp.Before(cond.LastSuccessfulTime.Time) {
				vds.UpdateTimestamp = &cond.LastTransitionTime.Time
				first = false
			}
		}
	}

	vds.Released = int(versioneddataset.Spec.Released)
	return vds, nil
}

func VersionFiles(ctx context.Context, c dynamic.Interface, input *generated.VersionedDataset, filter *generated.FileFilter) (*generated.PaginatedResult, error) {
	prefix := fmt.Sprintf("dataset/%s/%s/", input.Dataset.Name, input.Version)
	minioClient, _, err := minio.GetClients()
	if err != nil {
		return nil, err
	}
	keyword := ""
	if filter != nil && filter.Keyword != nil {
		keyword = *filter.Keyword
	}
	objectInfoList := minioutils.ListObjectCompleteInfo(ctx, input.Namespace, prefix, minioClient, -1)
	sort.Slice(objectInfoList, func(i, j int) bool {
		return objectInfoList[i].LastModified.After(objectInfoList[j].LastModified)
	})

	result := make([]generated.PageNode, 0)
	for _, obj := range objectInfoList {
		if keyword == "" || strings.Contains(obj.Key, keyword) {
			result = append(result, generated.F{
				Path:     obj.Key,
				FileType: obj.ContentType,
				Count:    &dataCount,
				Time:     &obj.LastModified,
			})
		}
	}
	page, size := 1, 10
	if filter != nil && filter.Page != nil && *filter.Page > 0 {
		page = *filter.Page
	}
	if filter != nil && filter.PageSize != nil && *filter.PageSize > 0 {
		size = *filter.PageSize
	}

	total := len(result)
	end := page * size
	if end > total {
		end = total
	}
	start := (page - 1) * size
	if start < total {
		result = result[start:end]
	} else {
		result = make([]generated.PageNode, 0)
	}
	return &generated.PaginatedResult{
		TotalCount:  total,
		HasNextPage: end < total,
		Nodes:       result,
		Page:        &page,
		PageSize:    &size,
	}, nil
}

func ListVersionedDatasets(ctx context.Context, c dynamic.Interface, input *generated.ListVersionedDatasetInput) (*generated.PaginatedResult, error) {
	listOptions := metav1.ListOptions{}
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
	ns := "default"
	if input.Namespace != nil {
		ns = *input.Namespace
	}
	list, err := c.Resource(versioneddatasetSchem).Namespace(ns).List(ctx, listOptions)
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
	for _, u := range list.Items {
		uu, _ := versionedDataset2model(&u)
		if input.DisplayName != nil && uu.DisplayName != *input.DisplayName {
			continue
		}
		if input.Keyword != nil {
			if strings.Contains(uu.Name, *input.Keyword) {
				goto add
			}
			if strings.Contains(uu.Namespace, *input.Keyword) {
				goto add
			}
			if strings.Contains(uu.DisplayName, *input.Keyword) {
				goto add
			}
			for _, v := range uu.Annotations {
				if strings.Contains(v.(string), *input.Keyword) {
					goto add
				}
			}
			continue
		}
	add:
		result = append(result, uu)
	}
	total := len(result)
	end := page * size
	if end > total {
		end = total
	}
	start := (page - 1) * size
	if start < total {
		result = result[start:end]
	} else {
		result = make([]generated.PageNode, 0)
	}
	return &generated.PaginatedResult{
		TotalCount:  total,
		HasNextPage: end < total,
		Nodes:       result,
		Page:        &page,
		PageSize:    &size,
	}, nil
}

func GetVersionedDataset(ctx context.Context, c dynamic.Interface, name, namespace string) (*generated.VersionedDataset, error) {
	obj, err := c.Resource(versioneddatasetSchem).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return versionedDataset2model(obj)
}

func DeleteVersionedDatasets(ctx context.Context, c dynamic.Interface, input *generated.DeleteVersionedDatasetInput) (*string, error) {
	none := ""
	listOptions := metav1.ListOptions{}
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
	err := c.Resource(versioneddatasetSchem).Namespace(input.Namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, listOptions)
	return &none, err
}

func UpdateVersionedDataset(ctx context.Context, c dynamic.Interface, input *generated.UpdateVersionedDatasetInput) (*generated.VersionedDataset, error) {
	obj, err := c.Resource(versioneddatasetSchem).Namespace(input.Namespace).Get(ctx, input.Name, metav1.GetOptions{})
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
	if input.Released != nil {
		_ = unstructured.SetNestedField(obj.Object, *input.Released, "spec", "released")
	}
	displayname, _, _ := unstructured.NestedString(obj.Object, "spec", "displayName")
	description, _, _ := unstructured.NestedString(obj.Object, "spec", "description")
	if input.DisplayName != "" && input.DisplayName != displayname {
		_ = unstructured.SetNestedField(obj.Object, input.DisplayName, "spec", "displayName")
	}
	if input.Description != nil && *input.Description != description {
		_ = unstructured.SetNestedField(obj.Object, *input.Description, "spec", "description")
	}
	fg := make([]any, 0)
	for _, item := range input.FileGroups {
		fg = append(fg, v1alpha1.FileGroup{
			Source: &v1alpha1.TypedObjectReference{
				Kind:      item.Source.Kind,
				Name:      item.Source.Name,
				Namespace: item.Source.Namespace,
			},
			Paths: item.Paths,
		})
	}
	if err = unstructured.SetNestedSlice(obj.Object, fg, "spec", "fileGroups"); err != nil {
		return nil, err
	}
	obj, err = c.Resource(versioneddatasetSchem).Namespace(input.Namespace).Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return versionedDataset2model(obj)
}

func CreateVersionedDataset(ctx context.Context, c dynamic.Interface, input *generated.CreateVersionedDatasetInput) (*generated.VersionedDataset, error) {
	vds := v1alpha1.VersionedDataset{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VersionedDataset",
			APIVersion: v1alpha1.GroupVersion.String(),
		},
	}
	vds.Name = input.Name
	vds.Namespace = input.Namespace
	vds.Spec = v1alpha1.VersionedDatasetSpec{
		Version: input.Version,
		Dataset: &v1alpha1.TypedObjectReference{
			Kind:      "Dataset",
			Name:      input.DatasetName,
			Namespace: &input.Namespace,
		},
		Released: 0,
		CommonSpec: v1alpha1.CommonSpec{
			DisplayName: input.DisplayName,
		},
	}
	if input.Description != nil {
		vds.Spec.Description = *input.Description
	}
	if len(input.Labels) > 0 {
		l := make(map[string]string)
		for k, v := range input.Labels {
			l[k] = v.(string)
		}
		vds.SetLabels(l)
	}
	if len(input.Annotations) > 0 {
		a := make(map[string]string)
		for k, v := range input.Annotations {
			a[k] = v.(string)
		}
		vds.SetAnnotations(a)
	}
	if len(input.FileGrups) > 0 {
		fg := make([]v1alpha1.FileGroup, 0)
		for _, item := range input.FileGrups {
			fg = append(fg, v1alpha1.FileGroup{
				Source: &v1alpha1.TypedObjectReference{
					Kind:      item.Source.Kind,
					Name:      item.Source.Name,
					Namespace: item.Source.Namespace,
				},
				Paths: item.Paths,
			})
		}
		vds.Spec.FileGroups = fg
	}
	if input.InheritedFrom != nil {
		vds.Spec.InheritedFrom = *input.InheritedFrom
		labelSelector := fmt.Sprintf("%s=%s", v1alpha1.LabelVersionedDatasetVersionOwner, input.DatasetName)
		// 选中目标的versionDataset
		versionList, err := c.Resource(versioneddatasetSchem).Namespace(input.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err != nil {
			return nil, err
		}

		for _, item := range versionList.Items {
			v := &v1alpha1.VersionedDataset{}
			if err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, v); err != nil {
				return nil, err
			}
			if v.Spec.Version == *input.InheritedFrom {
				for _, cond := range v.Status.Conditions {
					if !(cond.Type == v1alpha1.TypeReady && cond.Status == v1.ConditionTrue) {
						return nil, fmt.Errorf("inherit from a version with an incorrect synchronization state will not be created. reason: %s, errMsg: %s", cond.Reason, cond.Message)
					}
				}
			}
		}
	}
	o, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&vds)
	if err != nil {
		return nil, err
	}
	obj, err := c.Resource(versioneddatasetSchem).Namespace(input.Namespace).Create(ctx, &unstructured.Unstructured{Object: o}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return versionedDataset2model(obj)
}
