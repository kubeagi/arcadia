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

package model

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	miniogo "github.com/minio/minio-go/v7"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/graphql-server/go-server/graph/generated"
	"github.com/kubeagi/arcadia/graphql-server/go-server/pkg/minio"
	graphqlutils "github.com/kubeagi/arcadia/graphql-server/go-server/pkg/utils"
	"github.com/kubeagi/arcadia/pkg/utils"
	"github.com/kubeagi/arcadia/pkg/utils/minioutils"
)

func obj2model(obj *unstructured.Unstructured) *generated.Model {
	id := string(obj.GetUID())
	creationtimestamp := obj.GetCreationTimestamp().Time
	displayName, _, _ := unstructured.NestedString(obj.Object, "spec", "displayName")

	types, _, _ := unstructured.NestedString(obj.Object, "spec", "types")
	description, _, _ := unstructured.NestedString(obj.Object, "spec", "description")
	status := ""
	var updateTime time.Time
	conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if found && len(conditions) > 0 {
		condition, ok := conditions[0].(map[string]interface{})
		if ok {
			timeStr, _ := condition["lastTransitionTime"].(string)
			updateTime, _ = utils.RFC3339Time(timeStr)
			status, _ = condition["status"].(string)
		}
	} else {
		status = "unknow"
	}
	md := generated.Model{
		ID:                &id,
		Name:              obj.GetName(),
		Namespace:         obj.GetNamespace(),
		Labels:            graphqlutils.MapStr2Any(obj.GetLabels()),
		Annotations:       graphqlutils.MapStr2Any(obj.GetAnnotations()),
		DisplayName:       &displayName,
		Description:       &description,
		Status:            &status,
		Types:             types,
		CreationTimestamp: &creationtimestamp,
		UpdateTimestamp:   &updateTime,
	}
	return &md
}

func CreateModel(ctx context.Context, c dynamic.Interface, input generated.CreateModelInput) (*generated.Model, error) {
	displayName, description, types := "", "", ""
	if input.DisplayName != nil {
		displayName = *input.DisplayName
	}
	if input.Description != nil {
		description = *input.Description
	}
	if input.Types != "" {
		types = input.Types
	}

	model := v1alpha1.Model{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Model",
			APIVersion: v1alpha1.GroupVersion.String(),
		},
		Spec: v1alpha1.ModelSpec{
			CommonSpec: v1alpha1.CommonSpec{
				DisplayName: displayName,
				Description: description,
			},
			Types: types,
		},
	}
	unstructuredModel, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&model)
	if err != nil {
		return nil, err
	}
	obj, err := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "models"}).
		Namespace(input.Namespace).Create(ctx, &unstructured.Unstructured{Object: unstructuredModel}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	md := obj2model(obj)
	return md, nil
}

func UpdateModel(ctx context.Context, c dynamic.Interface, input *generated.UpdateModelInput) (*generated.Model, error) {
	resource := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "models"})
	obj, err := resource.Namespace(input.Namespace).Get(ctx, input.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	obj.SetLabels(graphqlutils.MapAny2Str(input.Labels))
	obj.SetAnnotations(graphqlutils.MapAny2Str(input.Annotations))

	displayname, _, _ := unstructured.NestedString(obj.Object, "spec", "displayName")
	if input.DisplayName != nil && *input.DisplayName != displayname {
		_ = unstructured.SetNestedField(obj.Object, *input.DisplayName, "spec", "displayName")
	}
	description, _, _ := unstructured.NestedString(obj.Object, "spec", "description")
	if input.Description != nil && *input.Description != description {
		_ = unstructured.SetNestedField(obj.Object, *input.Description, "spec", "description")
	}
	types, _, _ := unstructured.NestedString(obj.Object, "spec", "types")
	if input.Types != nil && *input.Types != types {
		_ = unstructured.SetNestedField(obj.Object, *input.Types, "spec", "types")
	}

	updatedObject, err := resource.Namespace(input.Namespace).Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	md := obj2model(updatedObject)
	return md, nil
}

func DeleteModels(ctx context.Context, c dynamic.Interface, input *generated.DeleteCommonInput) (*string, error) {
	name := ""
	labelSelector, fieldSelector := "", ""
	if input.Name != nil {
		name = *input.Name
	}
	if input.FieldSelector != nil {
		fieldSelector = *input.FieldSelector
	}
	if input.LabelSelector != nil {
		labelSelector = *input.LabelSelector
	}
	resource := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "models"})
	if name != "" {
		err := resource.Namespace(input.Namespace).Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil {
			return nil, err
		}
	} else {
		err := resource.Namespace(input.Namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
			LabelSelector: labelSelector,
			FieldSelector: fieldSelector,
		})
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func ListModels(ctx context.Context, c dynamic.Interface, input generated.ListCommonInput) (*generated.PaginatedResult, error) {
	keyword, labelSelector, fieldSelector := "", "", ""
	page, pageSize := 1, 10
	if input.Keyword != nil {
		keyword = *input.Keyword
	}
	if input.FieldSelector != nil {
		fieldSelector = *input.FieldSelector
	}
	if input.LabelSelector != nil {
		labelSelector = *input.LabelSelector
	}
	if input.Page != nil && *input.Page > 0 {
		page = *input.Page
	}
	if input.PageSize != nil && *input.PageSize > 0 {
		pageSize = *input.PageSize
	}

	dsSchema := schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "models"}
	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	}
	us, err := c.Resource(dsSchema).Namespace(input.Namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}
	// sort by creation time
	sort.Slice(us.Items, func(i, j int) bool {
		return us.Items[i].GetCreationTimestamp().After(us.Items[j].GetCreationTimestamp().Time)
	})

	totalCount := len(us.Items)

	result := make([]generated.PageNode, 0, pageSize)
	for _, u := range us.Items {
		m := obj2model(&u)
		// filter based on `keyword`
		if keyword != "" {
			if !strings.Contains(m.Name, keyword) && !strings.Contains(*m.DisplayName, keyword) {
				continue
			}
		}
		result = append(result, m)

		// break if page size matches
		if len(result) == pageSize {
			break
		}
	}

	end := page * pageSize
	if end > totalCount {
		end = totalCount
	}

	return &generated.PaginatedResult{
		TotalCount:  totalCount,
		HasNextPage: end < totalCount,
		Nodes:       result,
	}, nil
}

func ReadModel(ctx context.Context, c dynamic.Interface, name, namespace string) (*generated.Model, error) {
	resource := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "models"})
	u, err := resource.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return obj2model(u), nil
}

func ModelFiles(ctx context.Context, c dynamic.Interface, modelName, namespace string, input *generated.FileFilter) (*generated.PaginatedResult, error) {
	prefix := fmt.Sprintf("model/%s/", modelName)
	minioClient, _, err := minio.GetClients()
	if err != nil {
		return nil, err
	}
	keyword := ""
	if input != nil && input.Keyword != nil {
		keyword = *input.Keyword
	}
	objecttInfoList := minioutils.ListObjectCompleteInfo(ctx, namespace, prefix, minioClient)
	sort.Slice(objecttInfoList, func(i, j int) bool {
		return objecttInfoList[i].LastModified.After(objecttInfoList[j].LastModified)
	})

	result := make([]generated.PageNode, 0)
	for _, obj := range objecttInfoList {
		if keyword == "" || strings.Contains(obj.Key, keyword) {
			tf := generated.F{
				Path: strings.TrimPrefix(obj.Key, prefix),
				Time: &obj.LastModified,
			}
			size := utils.BytesToSizedStr(obj.Size)
			tf.Size = &size
			tags, err := minioClient.GetObjectTagging(ctx, namespace, obj.Key, miniogo.GetObjectTaggingOptions{})
			if err == nil {
				tagsMap := tags.ToMap()
				if v, ok := tagsMap[v1alpha1.ObjectTypeTag]; ok {
					tf.FileType = v
				}

				if v, ok := tagsMap[v1alpha1.ObjectCountTag]; ok {
					tf.Count = &v
				}
				if v, ok := tagsMap[minio.CreationTimestamp]; ok {
					if now, err := time.Parse(time.RFC3339, v); err == nil {
						tf.CreationTimestamp = &now
					}
				}
			}
			result = append(result, tf)
		}
	}
	page, size := 1, 10
	if input != nil && input.Page != nil && *input.Page > 0 {
		page = *input.Page
	}
	if input != nil && input.PageSize != nil && *input.PageSize > 0 {
		size = *input.PageSize
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
