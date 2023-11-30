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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	model "github.com/kubeagi/arcadia/graphql-server/go-server/graph/generated"
)

func obj2model(obj *unstructured.Unstructured) *model.Model {
	labels := make(map[string]interface{})
	for k, v := range obj.GetLabels() {
		labels[k] = v
	}
	annotations := make(map[string]interface{})
	for k, v := range obj.GetAnnotations() {
		annotations[k] = v
	}
	id := string(obj.GetUID())
	creationtimestamp := obj.GetCreationTimestamp().Time
	displayName, _, _ := unstructured.NestedString(obj.Object, "spec", "displayName")

	modeltype, _, _ := unstructured.NestedString(obj.Object, "spec", "types")
	description, _, _ := unstructured.NestedString(obj.Object, "spec", "description")
	status := ""
	var updateTime time.Time
	conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if found && len(conditions) > 0 {
		condition, ok := conditions[0].(map[string]interface{})
		if ok {
			timeStr, _ := condition["lastTransitionTime"].(string)
			updateTime, _ = time.Parse(time.RFC3339, timeStr)
			status, _ = condition["status"].(string)
		}
	} else {
		status = "unknow"
	}
	md := model.Model{
		ID:                &id,
		Name:              obj.GetName(),
		Namespace:         obj.GetNamespace(),
		Labels:            labels,
		Annotations:       annotations,
		DisplayName:       displayName,
		Description:       &description,
		Status:            &status,
		Modeltypes:        modeltype,
		CreationTimestamp: &creationtimestamp,
		UpdateTimestamp:   &updateTime,
	}
	return &md
}

func CreateModel(ctx context.Context, c dynamic.Interface, name, namespace, displayName, description, modeltypes string) (*model.Model, error) {
	model := v1alpha1.Model{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Model",
			APIVersion: v1alpha1.GroupVersion.String(),
		},
		Spec: v1alpha1.ModelSpec{
			DisplayName: displayName,
			Description: description,
			Types:       modeltypes,
		},
	}
	unstructuredModel, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&model)
	if err != nil {
		return nil, err
	}
	obj, err := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "models"}).
		Namespace(namespace).Create(ctx, &unstructured.Unstructured{Object: unstructuredModel}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	md := obj2model(obj)
	return md, nil
}

func UpdateModel(ctx context.Context, c dynamic.Interface, name, namespace, displayname string) (*model.Model, error) {
	resource := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "models"})
	obj, err := resource.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	obj.Object["spec"].(map[string]interface{})["displayName"] = displayname
	updatedObject, err := resource.Namespace(namespace).Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	md := obj2model(updatedObject)
	return md, nil
}

func DeleteModel(ctx context.Context, c dynamic.Interface, name, namespace, labelSelector, fieldSelector string) (*string, error) {
	resource := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "models"})
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

func ListModels(ctx context.Context, c dynamic.Interface, namespace, labelSelector, fieldSelector string) ([]*model.Model, error) {
	dsSchema := schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "models"}
	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	}
	us, err := c.Resource(dsSchema).Namespace(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}
	result := make([]*model.Model, len(us.Items))
	for idx, u := range us.Items {
		result[idx] = obj2model(&u)
	}
	return result, nil
}

func ReadModel(ctx context.Context, c dynamic.Interface, name, namespace string) (*model.Model, error) {
	resource := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "models"})
	u, err := resource.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return obj2model(u), nil
}
