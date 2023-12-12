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

package embedder

import (
	"context"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/graphql-server/go-server/graph/generated"
	"github.com/kubeagi/arcadia/graphql-server/go-server/pkg/common"
	graphqlutils "github.com/kubeagi/arcadia/graphql-server/go-server/pkg/utils"
	"github.com/kubeagi/arcadia/pkg/utils"
)

func embedder2model(obj *unstructured.Unstructured) *generated.Embedder {
	url, _, _ := unstructured.NestedString(obj.Object, "spec", "endpoint", "url")
	authsecret, _, _ := unstructured.NestedString(obj.Object, "spec", "endpoint", "authSecret", "name")
	authsecretNamespace, _, _ := unstructured.NestedString(obj.Object, "spec", "endpoint", "authSecret", "namespace")
	displayName, _, _ := unstructured.NestedString(obj.Object, "spec", "displayName")
	servicetype, _, _ := unstructured.NestedString(obj.Object, "spec", "type")
	updateTime := metav1.Now().Time
	conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if found && len(conditions) > 0 {
		condition, ok := conditions[0].(map[string]interface{})
		if ok {
			timeStr, _ := condition["lastTransitionTime"].(string)
			updateTime, _ = utils.RFC3339Time(timeStr)
		}
	}
	endpoint := generated.Endpoint{
		URL: &url,
		AuthSecret: &generated.TypedObjectReference{
			Kind:      "Secret",
			Name:      authsecret,
			Namespace: &authsecretNamespace,
		},
	}

	md := generated.Embedder{
		Name:            obj.GetName(),
		Namespace:       obj.GetNamespace(),
		Labels:          graphqlutils.MapStr2Any(obj.GetLabels()),
		Annotations:     graphqlutils.MapStr2Any(obj.GetAnnotations()),
		DisplayName:     &displayName,
		Endpoint:        &endpoint,
		Type:            &servicetype,
		UpdateTimestamp: &updateTime,
	}
	return &md
}

func CreateEmbedder(ctx context.Context, c dynamic.Interface, input generated.CreateEmbedderInput) (*generated.Embedder, error) {
	displayname, description, servicetype := "", "", ""

	if input.DisplayName != nil {
		displayname = *input.DisplayName
	}
	if input.Description != nil {
		description = *input.Description
	}
	if input.Type != nil {
		servicetype = *input.Type
	}

	embedder := v1alpha1.Embedder{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Embedder",
			APIVersion: v1alpha1.GroupVersion.String(),
		},
		Spec: v1alpha1.EmbedderSpec{
			CommonSpec: v1alpha1.CommonSpec{
				DisplayName: displayname,
				Description: description,
			},
			Provider: v1alpha1.Provider{
				Enpoint: &v1alpha1.Endpoint{
					URL: input.Endpointinput.URL,
				},
			},
			Type: v1alpha1.EmbeddingType(servicetype),
		},
	}

	// create auth secret
	if input.Endpointinput.Auth != nil {
		// create auth secret
		secret := common.MakeAuthSecretName(embedder.Name, "embedder")
		err := common.MakeAuthSecret(ctx, c, generated.TypedObjectReferenceInput{
			Name:      secret,
			Namespace: &input.Namespace,
		}, *input.Endpointinput.Auth, nil)
		if err != nil {
			return nil, err
		}
		embedder.Spec.Enpoint.AuthSecret = &v1alpha1.TypedObjectReference{
			Kind:      "Secret",
			Name:      secret,
			Namespace: &input.Namespace,
		}
	}

	unstructuredEmbedder, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&embedder)
	if err != nil {
		return nil, err
	}
	obj, err := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "embedders"}).
		Namespace(input.Namespace).Create(ctx, &unstructured.Unstructured{Object: unstructuredEmbedder}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	// update auth secret owner reference
	if input.Endpointinput.Auth != nil {
		// user obj as the owner
		secret := common.MakeAuthSecretName(embedder.Name, "embedder")
		err := common.MakeAuthSecret(ctx, c, generated.TypedObjectReferenceInput{
			Name:      secret,
			Namespace: &input.Namespace,
		}, *input.Endpointinput.Auth, obj)
		if err != nil {
			return nil, err
		}
	}

	ds := embedder2model(obj)
	return ds, nil
}

func UpdateEmbedder(ctx context.Context, c dynamic.Interface, name, namespace, displayname string) (*generated.Embedder, error) {
	resource := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "embedders"})
	obj, err := resource.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	obj.Object["spec"].(map[string]interface{})["displayName"] = displayname
	updatedObject, err := resource.Namespace(namespace).Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	ds := embedder2model(updatedObject)
	return ds, nil
}

func DeleteEmbedders(ctx context.Context, c dynamic.Interface, input *generated.DeleteCommonInput) (*string, error) {
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

	resource := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "embedders"})
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
func ListEmbedders(ctx context.Context, c dynamic.Interface, input generated.ListCommonInput) (*generated.PaginatedResult, error) {
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

	dsSchema := schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "embedders"}
	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	}
	us, err := c.Resource(dsSchema).Namespace(input.Namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}
	sort.Slice(us.Items, func(i, j int) bool {
		return us.Items[i].GetCreationTimestamp().After(us.Items[j].GetCreationTimestamp().Time)
	})

	totalCount := len(us.Items)

	result := make([]generated.PageNode, 0, pageSize)
	pageStart := (page - 1) * pageSize
	for index, u := range us.Items {
		// skip if smaller than the start index
		if index < pageStart {
			continue
		}
		m := embedder2model(&u)
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

func ReadEmbedder(ctx context.Context, c dynamic.Interface, name, namespace string) (*generated.Embedder, error) {
	resource := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "embedders"})
	u, err := resource.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return embedder2model(u), nil
}
