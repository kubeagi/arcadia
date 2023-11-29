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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	model "github.com/kubeagi/arcadia/graphql-server/go-server/graph/generated"
	"github.com/kubeagi/arcadia/pkg/embeddings"
)

func embedder2model(obj *unstructured.Unstructured) *model.Embedder {
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
	servicetype, _, _ := unstructured.NestedString(obj.Object, "spec", "serviceType")
	updateTime := metav1.Now().Time
	conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if found && len(conditions) > 0 {
		condition, ok := conditions[0].(map[string]interface{})
		if ok {
			timeStr, _ := condition["lastTransitionTime"].(string)
			updateTime, _ = time.Parse(time.RFC3339, timeStr)
		}
	}
	endpoint := model.Endpoint{
		URL: &url,
		AuthSecret: &model.TypedObjectReference{
			Kind:      "Secret",
			Name:      authsecret,
			Namespace: &authsecretNamespace,
		},
	}

	md := model.Embedder{
		Name:            obj.GetName(),
		Namespace:       obj.GetNamespace(),
		Labels:          labels,
		Annotations:     annotations,
		DisplayName:     displayName,
		Endpoint:        &endpoint,
		ServiceType:     &servicetype,
		UpdateTimestamp: &updateTime,
	}
	return &md
}

func CreateEmbedder(ctx context.Context, c dynamic.Interface, name, namespace, url, authsecret, displayname, description, servicetype string) (*model.Embedder, error) {
	embedder := v1alpha1.Embedder{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Embedder",
			APIVersion: v1alpha1.GroupVersion.String(),
		},
		Spec: v1alpha1.EmbedderSpec{
			CommonSpec: v1alpha1.CommonSpec{
				DisplayName: displayname,
			},
			Provider: v1alpha1.Provider{
				Enpoint: &v1alpha1.Endpoint{
					URL: url,
					AuthSecret: &v1alpha1.TypedObjectReference{
						Kind:      "Secret",
						Name:      authsecret,
						Namespace: &namespace,
					},
				},
			},
			ServiceType: embeddings.EmbeddingType(servicetype),
		},
	}

	unstructuredEmbedder, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&embedder)
	if err != nil {
		return nil, err
	}
	obj, err := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "embedders"}).
		Namespace(namespace).Create(ctx, &unstructured.Unstructured{Object: unstructuredEmbedder}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	ds := embedder2model(obj)
	return ds, nil
}

func UpdateEmbedder(ctx context.Context, c dynamic.Interface, name, namespace, displayname string) (*model.Embedder, error) {
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

func DeleteEmbedder(ctx context.Context, c dynamic.Interface, name, namespace, labelSelector, fieldSelector string) (*string, error) {
	resource := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "embedders"})
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
func ListEmbedders(ctx context.Context, c dynamic.Interface, namespace, labelSelector, fieldSelector string) ([]*model.Embedder, error) {
	dsSchema := schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "embedders"}
	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	}
	us, err := c.Resource(dsSchema).Namespace(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}
	sort.Slice(us.Items, func(i, j int) bool {
		return us.Items[i].GetCreationTimestamp().After(us.Items[j].GetCreationTimestamp().Time)
	})
	result := make([]*model.Embedder, len(us.Items))
	for idx, u := range us.Items {
		result[idx] = embedder2model(&u)
	}
	return result, nil
}

func ReadEmbedder(ctx context.Context, c dynamic.Interface, name, namespace string) (*model.Embedder, error) {
	resource := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "embedders"})
	u, err := resource.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return embedder2model(u), nil
}
