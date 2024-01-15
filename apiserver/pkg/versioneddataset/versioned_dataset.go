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

	miniogo "github.com/minio/minio-go/v7"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/client"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	graphqlutils "github.com/kubeagi/arcadia/apiserver/pkg/utils"
	"github.com/kubeagi/arcadia/pkg/utils"
)

var (
	versioneddatasetSchem = schema.GroupVersionResource{
		Group:    v1alpha1.GroupVersion.Group,
		Version:  v1alpha1.GroupVersion.Version,
		Resource: "versioneddatasets",
	}
)

func versionedDataset2modelConverter(obj *unstructured.Unstructured) (generated.PageNode, error) {
	return versionedDataset2model(obj)
}

func versionedDataset2model(obj *unstructured.Unstructured) (*generated.VersionedDataset, error) {
	vds := &generated.VersionedDataset{}
	id := string(obj.GetUID())
	vds.ID = &id
	vds.Name = obj.GetName()
	vds.Namespace = obj.GetNamespace()
	vds.Labels = graphqlutils.MapStr2Any(obj.GetLabels())
	vds.Annotations = graphqlutils.MapStr2Any(obj.GetAnnotations())
	versioneddataset := &v1alpha1.VersionedDataset{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, versioneddataset); err != nil {
		return nil, err
	}
	vds.CreationTimestamp = obj.GetCreationTimestamp().Time
	vds.UpdateTimestamp = &vds.CreationTimestamp
	vds.Creator = &versioneddataset.Spec.Creator
	vds.DisplayName = &versioneddataset.Spec.DisplayName
	vds.Description = &versioneddataset.Spec.Description
	vds.Dataset = generated.TypedObjectReference{
		APIGroup:  versioneddataset.Spec.Dataset.APIGroup,
		Kind:      versioneddataset.Spec.Dataset.Kind,
		Name:      versioneddataset.Spec.Dataset.Name,
		Namespace: versioneddataset.Spec.Dataset.Namespace,
	}

	vds.Version = versioneddataset.Spec.Version
	vds.SyncStatus = new(string)
	vds.SyncMsg = new(string)
	vds.DataProcessStatus = new(string)
	vds.DataProcessMsg = new(string)
	first := true
	for _, cond := range versioneddataset.Status.Conditions {
		if cond.Type == v1alpha1.TypeReady {
			*vds.SyncStatus = string(cond.Reason)
			*vds.SyncMsg = cond.Message
		}
		if cond.Type == v1alpha1.TypeDataProcessing {
			*vds.DataProcessStatus = string(cond.Reason)
			*vds.DataProcessMsg = cond.Message
		}
		if !cond.LastTransitionTime.IsZero() {
			if first || vds.UpdateTimestamp.Before(cond.LastSuccessfulTime.Time) {
				vds.UpdateTimestamp = &cond.LastTransitionTime.Time
				first = false
			}
		}
	}

	if s := common.GetObjStatus(obj); s != "" {
		*vds.SyncStatus = s
	}
	vds.Released = int(versioneddataset.Spec.Released)
	return vds, nil
}

func VersionFiles(ctx context.Context, c dynamic.Interface, input *generated.VersionedDataset, filter *generated.FileFilter) (*generated.PaginatedResult, error) {
	prefix := fmt.Sprintf("dataset/%s/%s/", input.Dataset.Name, input.Version)
	keyword := ""
	if filter != nil && filter.Keyword != nil {
		keyword = *filter.Keyword
	}

	systemClient, err := client.GetClient(nil)
	if err != nil {
		return nil, err
	}
	oss, err := common.SystemDatasourceOSS(ctx, nil, systemClient)
	if err != nil {
		return nil, err
	}
	anyObjectInfoList, err := oss.ListObjects(ctx, input.Namespace, miniogo.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})
	if err != nil {
		return nil, err
	}
	objectInfoList := anyObjectInfoList.([]miniogo.ObjectInfo)
	sort.Slice(objectInfoList, func(i, j int) bool {
		return objectInfoList[i].LastModified.After(objectInfoList[j].LastModified)
	})

	result := make([]generated.PageNode, 0)
	for _, obj := range objectInfoList {
		if keyword == "" || strings.Contains(obj.Key, keyword) {
			tf := generated.F{
				Path: strings.TrimPrefix(obj.Key, prefix),
				Time: &obj.LastModified,
			}

			size := utils.BytesToSizedStr(obj.Size)
			tf.Size = &size

			// parse tags
			tags, err := oss.Client.GetObjectTagging(ctx, input.Namespace, obj.Key, miniogo.GetObjectTaggingOptions{})
			if err == nil {
				tagsMap := tags.ToMap()
				if v, ok := tagsMap[v1alpha1.ObjectTypeTag]; ok {
					tf.FileType = v
				}

				if v, ok := tagsMap[v1alpha1.ObjectCountTag]; ok {
					tf.Count = &v
				}
			}

			if v, ok := obj.UserTags[common.CreationTimestamp]; ok {
				if now, err := time.Parse(time.RFC3339, v); err == nil {
					tf.CreationTimestamp = &now
				}
			}
			result = append(result, tf)
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
	page, size := 1, 10
	if input.Page != nil && *input.Page > 0 {
		page = *input.Page
	}
	if input.PageSize != nil && *input.PageSize > 0 {
		size = *input.PageSize
	}
	ns := "default"
	if input.Namespace != nil {
		ns = *input.Namespace
	}
	filter := make([]common.ResourceFilter, 0)
	if input.DisplayName != nil {
		filter = append(filter, common.FilterVersionedDatasetByDisplayName(*input.DisplayName))
	}
	if input.Keyword != nil {
		filter = append(filter, common.FilterVersionedDatasetByKeyword(*input.Keyword))
	}
	list, err := c.Resource(versioneddatasetSchem).Namespace(ns).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}
	return common.ListReources(list, page, size, versionedDataset2modelConverter, filter...)
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
	obj.SetLabels(graphqlutils.MapAny2Str(input.Labels))
	obj.SetAnnotations(graphqlutils.MapAny2Str(input.Annotations))
	if input.Released != nil {
		_ = unstructured.SetNestedField(obj.Object, *input.Released, "spec", "released")
	}
	displayname, _, _ := unstructured.NestedString(obj.Object, "spec", "displayName")
	description, _, _ := unstructured.NestedString(obj.Object, "spec", "description")
	if input.DisplayName != nil && *input.DisplayName != displayname {
		_ = unstructured.SetNestedField(obj.Object, *input.DisplayName, "spec", "displayName")
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
	}
	if input.DisplayName != nil {
		vds.Spec.DisplayName = *input.DisplayName
	}
	if input.Description != nil {
		vds.Spec.Description = *input.Description
	}
	common.SetCreator(ctx, &vds.Spec.CommonSpec)
	vds.SetLabels(graphqlutils.MapAny2Str(input.Labels))
	vds.SetAnnotations(graphqlutils.MapAny2Str(input.Annotations))
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
				isReady := false
				var errMessage error
				for _, cond := range v.Status.Conditions {
					if cond.Type == v1alpha1.TypeReady && cond.Status == v1.ConditionTrue {
						isReady = true
						break
					}
					if cond.Type == v1alpha1.TypeReady && cond.Status != v1.ConditionTrue {
						errMessage = fmt.Errorf("inherit from a version with an incorrect synchronization state will not be created. reason: %s, errMsg: %s", cond.Reason, cond.Message)
					}
				}
				if !isReady {
					return nil, errMessage
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
