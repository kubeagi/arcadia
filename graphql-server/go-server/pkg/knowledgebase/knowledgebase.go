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

package knowledgebase

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
)

func knowledgebase2model(obj *unstructured.Unstructured) *model.KnowledgeBase {
	labels := make(map[string]interface{})
	for k, v := range obj.GetLabels() {
		labels[k] = v
	}
	annotations := make(map[string]interface{})
	for k, v := range obj.GetAnnotations() {
		annotations[k] = v
	}
	displayName, _, _ := unstructured.NestedString(obj.Object, "spec", "displayName")
	description, _, _ := unstructured.NestedString(obj.Object, "spec", "description")
	embedder, _, _ := unstructured.NestedMap(obj.Object, "spec", "embedder")
	embeddernp, _ := embedder["namespace"].(string)
	vectorStore, _, _ := unstructured.NestedMap(obj.Object, "spec", "vectorStore")
	vectorStorenp := vectorStore["namespace"].(string)
	apiversion := obj.GetAPIVersion()
	fileGroupDetails, _, _ := unstructured.NestedSlice(obj.Object, "status", "fileGroupDetail")
	var filegroupdetails []*model.Filegroupdetail
	for _, filegroupdetail := range fileGroupDetails {
		fileDetails := filegroupdetail.(map[string]interface{})["fileDetails"].([]interface{})
		var filedetails []*model.Filedetail
		fns := filegroupdetail.(map[string]interface{})["source"].(map[string]interface{})["namespace"].(string)
		for _, filedetailsmap := range fileDetails {
			filedetail := &model.Filedetail{
				Path:  filedetailsmap.(map[string]interface{})["path"].(string),
				Phase: filedetailsmap.(map[string]interface{})["phase"].(string),
			}
			filedetails = append(filedetails, filedetail)
		}
		filegroupdetail := &model.Filegroupdetail{
			Source: &model.TypedObjectReference{
				Kind:      filegroupdetail.(map[string]interface{})["source"].(map[string]interface{})["kind"].(string),
				Name:      filegroupdetail.(map[string]interface{})["source"].(map[string]interface{})["name"].(string),
				Namespace: &fns,
			},
			Filedetails: filedetails,
		}
		filegroupdetails = append(filegroupdetails, filegroupdetail)
	}
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

	md := model.KnowledgeBase{
		Name:        obj.GetName(),
		Namespace:   obj.GetNamespace(),
		Labels:      labels,
		Annotations: annotations,
		Embedder: &model.TypedObjectReference{
			APIGroup:  &apiversion,
			Kind:      embedder["kind"].(string),
			Name:      embedder["name"].(string),
			Namespace: &embeddernp,
		},
		VectorStore: &model.TypedObjectReference{
			APIGroup:  &apiversion,
			Kind:      vectorStore["kind"].(string),
			Name:      vectorStore["name"].(string),
			Namespace: &vectorStorenp,
		},
		FileGroupDetails: filegroupdetails,
		DisplayName:      displayName,
		Description:      &description,
		Status:           &status,
		UpdateTimestamp:  &updateTime,
	}
	return &md
}

func CreateKnowledgeBase(ctx context.Context, c dynamic.Interface, name, namespace, displayname, discription string, vectorstore, embedder v1alpha1.TypedObjectReference, filegroups []v1alpha1.FileGroup) (*model.KnowledgeBase, error) {
	knowledgebase := v1alpha1.KnowledgeBase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "KnowledgeBase",
			APIVersion: v1alpha1.GroupVersion.String(),
		},
		Spec: v1alpha1.KnowledgeBaseSpec{
			CommonSpec: v1alpha1.CommonSpec{
				DisplayName: displayname,
				Description: discription,
			},
			Embedder:    &embedder,
			VectorStore: &vectorstore,
			FileGroups:  filegroups,
		},
	}

	unstructuredKnowledgeBase, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&knowledgebase)
	if err != nil {
		return nil, err
	}
	obj, err := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "knowledgebases"}).
		Namespace(namespace).Create(ctx, &unstructured.Unstructured{Object: unstructuredKnowledgeBase}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	kb := knowledgebase2model(obj)
	return kb, nil
}

func UpdateKnowledgeBase(ctx context.Context, c dynamic.Interface, name, namespace, displayname string) (*model.KnowledgeBase, error) {
	resource := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "knowledgebases"})
	obj, err := resource.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	obj.Object["spec"].(map[string]interface{})["displayName"] = displayname
	updatedObject, err := resource.Namespace(namespace).Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	kb := knowledgebase2model(updatedObject)
	return kb, nil
}

func DeleteKnowledgeBase(ctx context.Context, c dynamic.Interface, name, namespace, labelSelector, fieldSelector string) (*string, error) {
	resource := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "knowledgebases"})
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

func ReadKnowledgeBase(ctx context.Context, c dynamic.Interface, name, namespace string) (*model.KnowledgeBase, error) {
	resource := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "knowledgebases"})
	u, err := resource.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return knowledgebase2model(u), nil
}

func ListKnowledgeBases(ctx context.Context, c dynamic.Interface, namespace, labelSelector, fieldSelector string) ([]*model.KnowledgeBase, error) {
	dsSchema := schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "knowledgebases"}
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
	result := make([]*model.KnowledgeBase, len(us.Items))
	for idx, u := range us.Items {
		result[idx] = knowledgebase2model(&u)
	}
	return result, nil
}
