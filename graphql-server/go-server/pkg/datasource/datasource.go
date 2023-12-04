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

package datasource

import (
	"context"
	"fmt"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/graphql-server/go-server/graph/generated"
	"github.com/kubeagi/arcadia/pkg/utils"
)

var dsSchema = schema.GroupVersionResource{
	Group:    v1alpha1.GroupVersion.Group,
	Version:  v1alpha1.GroupVersion.Version,
	Resource: "datasources",
}

func datasource2model(obj *unstructured.Unstructured) *generated.Datasource {
	labels := make(map[string]interface{})
	for k, v := range obj.GetLabels() {
		labels[k] = v
	}
	annotations := make(map[string]interface{})
	for k, v := range obj.GetAnnotations() {
		annotations[k] = v
	}
	url, _, _ := unstructured.NestedString(obj.Object, "spec", "endpoint", "url")
	authsecret, _, _ := unstructured.NestedString(obj.Object, "spec", "endpoint", "authSecret", "name")
	authsecretNamespace, _, _ := unstructured.NestedString(obj.Object, "spec", "endpoint", "authSecret", "namespace")
	displayName, _, _ := unstructured.NestedString(obj.Object, "spec", "displayName")
	description, _, _ := unstructured.NestedString(obj.Object, "spec", "description")
	bucket, _, _ := unstructured.NestedString(obj.Object, "spec", "oss", "bucket")
	insecure, _, _ := unstructured.NestedBool(obj.Object, "spec", "endpoint", "insecure")
	status := "unknow"
	updateTime := metav1.Now().Time
	conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if found && len(conditions) > 0 {
		condition, ok := conditions[0].(map[string]interface{})
		if ok {
			timeStr, _ := condition["lastTransitionTime"].(string)
			updateTime, _ = utils.RFC3339Time(timeStr)
			status, _ = condition["status"].(string)
		}
	}
	endpoint := generated.Endpoint{
		URL: &url,
		AuthSecret: &generated.TypedObjectReference{
			Kind:      "Secret",
			Name:      authsecret,
			Namespace: &authsecretNamespace,
		},
		Insecure: &insecure,
	}
	oss := generated.Oss{
		Bucket: &bucket,
	}
	md := generated.Datasource{
		Name:            obj.GetName(),
		Namespace:       obj.GetNamespace(),
		Labels:          labels,
		Annotations:     annotations,
		DisplayName:     &displayName,
		Description:     &description,
		Endpoint:        &endpoint,
		Oss:             &oss,
		Status:          &status,
		UpdateTimestamp: &updateTime,
	}
	return &md
}

func CreateDatasource(ctx context.Context, c dynamic.Interface, name, namespace, url, authsecret, bucket, displayname, description string, insecure bool) (*generated.Datasource, error) {
	var datasource v1alpha1.Datasource
	if url != "" {
		datasource = v1alpha1.Datasource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Datasource",
				APIVersion: v1alpha1.GroupVersion.String(),
			},
			Spec: v1alpha1.DatasourceSpec{
				CommonSpec: v1alpha1.CommonSpec{
					DisplayName: displayname,
					Description: description,
				},
				Enpoint: &v1alpha1.Endpoint{
					URL: url,
					AuthSecret: &v1alpha1.TypedObjectReference{
						Kind:      "Secret",
						Name:      authsecret,
						Namespace: &namespace,
					},
					Insecure: insecure,
				},
				OSS: &v1alpha1.OSS{
					Bucket: bucket,
				},
			},
		}
	} else {
		datasource = v1alpha1.Datasource{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "Datasource",
				APIVersion: v1alpha1.GroupVersion.String(),
			},
			Spec: v1alpha1.DatasourceSpec{
				CommonSpec: v1alpha1.CommonSpec{
					DisplayName: displayname,
					Description: description,
				},
			},
		}
	}

	unstructuredDatasource, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&datasource)
	if err != nil {
		return nil, err
	}
	obj, err := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "datasources"}).
		Namespace(namespace).Create(ctx, &unstructured.Unstructured{Object: unstructuredDatasource}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	ds := datasource2model(obj)
	return ds, nil
}

func UpdateDatasource(ctx context.Context, c dynamic.Interface, name, namespace, displayname string) (*generated.Datasource, error) {
	resource := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "datasources"})
	obj, err := resource.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	obj.Object["spec"].(map[string]interface{})["displayName"] = displayname
	updatedObject, err := resource.Namespace(namespace).Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	ds := datasource2model(updatedObject)
	return ds, nil
}

func DeleteDatasource(ctx context.Context, c dynamic.Interface, name, namespace, labelSelector, fieldSelector string) (*string, error) {
	resource := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "datasources"})
	if name != "" {
		err := resource.Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil {
			return nil, err
		}
	} else {
		err := resource.Namespace(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
			LabelSelector: labelSelector,
			FieldSelector: fieldSelector,
		})
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}
func ListDatasources(ctx context.Context, c dynamic.Interface, input *generated.ListDatasourceInput) (*generated.PaginatedResult, error) {
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
	datasList, err := c.Resource(dsSchema).Namespace(input.Namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}
	sort.Slice(datasList.Items, func(i, j int) bool {
		return datasList.Items[i].GetCreationTimestamp().After(datasList.Items[j].GetCreationTimestamp().Time)
	})
	page, size := 1, 10
	if input.Page != nil && *input.Page > 0 {
		page = *input.Page
	}
	if input.PageSize != nil && *input.PageSize > 0 {
		size = *input.PageSize
	}
	result := make([]generated.PageNode, 0)
	for _, u := range datasList.Items {
		uu := datasource2model(&u)
		if input.DisplayName != nil && *uu.DisplayName != *input.DisplayName {
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
			if strings.Contains(*uu.DisplayName, *input.Keyword) {
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

func ReadDatasource(ctx context.Context, c dynamic.Interface, name, namespace string) (*generated.Datasource, error) {
	resource := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "datasources"})
	u, err := resource.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return datasource2model(u), nil
}
