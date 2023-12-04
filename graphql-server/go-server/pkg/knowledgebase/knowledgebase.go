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
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/graphql-server/go-server/graph/generated"
	"github.com/kubeagi/arcadia/pkg/utils"
)

func knowledgebase2model(obj *unstructured.Unstructured) *generated.KnowledgeBase {
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
	description, _, _ := unstructured.NestedString(obj.Object, "spec", "description")
	embedder, _, _ := unstructured.NestedMap(obj.Object, "spec", "embedder")
	embeddernp, _ := embedder["namespace"].(string)
	vectorStore, _, _ := unstructured.NestedMap(obj.Object, "spec", "vectorStore")
	vectorStorenp := vectorStore["namespace"].(string)
	apiversion := obj.GetAPIVersion()
	fileGroupDetails, _, _ := unstructured.NestedSlice(obj.Object, "status", "fileGroupDetail")
	var filegroupdetails []*generated.Filegroupdetail
	for _, filegroupdetail := range fileGroupDetails {
		fileDetails := filegroupdetail.(map[string]interface{})["fileDetails"].([]interface{})
		var filedetails []*generated.Filedetail
		fns := filegroupdetail.(map[string]interface{})["source"].(map[string]interface{})["namespace"].(string)
		for _, filedetailsmap := range fileDetails {
			detail, ok := filedetailsmap.(map[string]interface{})
			if !ok {
				continue
			}
			fileType, _ := detail["type"].(string)
			count, _ := detail["count"].(string)
			size, _ := detail["size"].(string)
			path, _ := detail["path"].(string)
			phase, _ := detail["phase"].(string)

			var updateTime time.Time
			lastUpdateTime, ok := detail["lastUpdateTime"].(string)
			if ok {
				updateTime, _ = utils.RFC3339Time(lastUpdateTime)
			}

			filedetail := &generated.Filedetail{
				FileType:        fileType,
				Count:           count,
				Size:            size,
				Path:            path,
				Phase:           phase,
				UpdateTimestamp: &updateTime,
			}
			filedetails = append(filedetails, filedetail)
		}
		filegroupdetail := &generated.Filegroupdetail{
			Source: &generated.TypedObjectReference{
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
			updateTime, _ = utils.RFC3339Time(timeStr)
			status, _ = condition["status"].(string)
		}
	} else {
		status = "unknow"
	}

	md := generated.KnowledgeBase{
		ID:          &id,
		Name:        obj.GetName(),
		Namespace:   obj.GetNamespace(),
		Labels:      labels,
		Annotations: annotations,
		Embedder: &generated.TypedObjectReference{
			APIGroup:  &apiversion,
			Kind:      embedder["kind"].(string),
			Name:      embedder["name"].(string),
			Namespace: &embeddernp,
		},
		VectorStore: &generated.TypedObjectReference{
			APIGroup:  &apiversion,
			Kind:      vectorStore["kind"].(string),
			Name:      vectorStore["name"].(string),
			Namespace: &vectorStorenp,
		},
		FileGroupDetails:  filegroupdetails,
		DisplayName:       &displayName,
		Description:       &description,
		Status:            &status,
		CreationTimestamp: &creationtimestamp,
		UpdateTimestamp:   &updateTime,
	}
	return &md
}

func CreateKnowledgeBase(ctx context.Context, c dynamic.Interface, name, namespace, displayname, discription, embedder string, vectorstore v1alpha1.TypedObjectReference, filegroups []v1alpha1.FileGroup) (*generated.KnowledgeBase, error) {
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
			Embedder: &v1alpha1.TypedObjectReference{
				Kind:      "Embedder",
				Name:      embedder,
				Namespace: &namespace,
			},
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
	if kb.FileGroupDetails == nil {
		// fill in file group without any details
		details := make([]*generated.Filegroupdetail, len(filegroups))
		for index, fg := range filegroups {
			fgDetail := &generated.Filegroupdetail{
				Source: (*generated.TypedObjectReference)(fg.Source),
			}
			fileDetails := make([]*generated.Filedetail, len(fg.Paths))
			for findex, path := range fg.Paths {
				fileDetails[findex] = &generated.Filedetail{
					Path:  path,
					Phase: "",
				}
			}
			fgDetail.Filedetails = fileDetails
			details[index] = fgDetail
		}
		kb.FileGroupDetails = details
	}
	return kb, nil
}

func UpdateKnowledgeBase(ctx context.Context, c dynamic.Interface, input *generated.UpdateKnowledgeBaseInput) (*generated.KnowledgeBase, error) {
	resource := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "knowledgebases"})
	obj, err := resource.Namespace(input.Namespace).Get(ctx, input.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// Create an instance of the structured custom resource type
	structuredObj := &v1alpha1.KnowledgeBase{}
	if err = utils.UnstructuredToStructured(obj, structuredObj); err != nil {
		return nil, err
	}

	if input.DisplayName != nil && *input.DisplayName != structuredObj.Spec.DisplayName {
		obj.Object["spec"].(map[string]interface{})["displayName"] = *input.DisplayName
	}
	if input.Description != nil && *input.Description != structuredObj.Spec.Description {
		obj.Object["spec"].(map[string]interface{})["description"] = *input.Description
	}

	if input.FileGroups != nil {
		filegroups := make([]v1alpha1.FileGroup, len(input.FileGroups))
		for index, f := range input.FileGroups {
			filegroup := v1alpha1.FileGroup{
				Source: (*v1alpha1.TypedObjectReference)(&f.Source),
				Paths:  f.Path,
			}
			filegroups[index] = filegroup
		}
		obj.Object["spec"].(map[string]interface{})["fileGroups"] = filegroups
	}

	updatedObject, err := resource.Namespace(input.Namespace).Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	return knowledgebase2model(updatedObject), nil
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

func ReadKnowledgeBase(ctx context.Context, c dynamic.Interface, name, namespace string) (*generated.KnowledgeBase, error) {
	resource := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "knowledgebases"})
	u, err := resource.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return knowledgebase2model(u), nil
}

func ListKnowledgeBases(ctx context.Context, c dynamic.Interface, input generated.ListKnowledgeBaseInput) (*generated.PaginatedResult, error) {
	name, displayName, labelSelector, fieldSelector := "", "", "", ""
	page, pageSize := 1, 10
	if input.Name != nil {
		name = *input.Name
	}
	if input.DisplayName != nil {
		displayName = *input.DisplayName
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

	dsSchema := schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "knowledgebases"}
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
	result := make([]*generated.KnowledgeBase, len(us.Items))
	for idx, u := range us.Items {
		result[idx] = knowledgebase2model(&u)
	}

	var filteredResult []generated.PageNode
	for idx, u := range result {
		if (name == "" || strings.Contains(u.Name, name)) && (displayName == "" || strings.Contains(*u.DisplayName, displayName)) {
			filteredResult = append(filteredResult, result[idx])
		}
	}

	totalCount := len(filteredResult)
	end := page * pageSize
	if end > totalCount {
		end = totalCount
	}
	return &generated.PaginatedResult{
		TotalCount:  totalCount,
		HasNextPage: end < totalCount,
		Nodes:       filteredResult[(page-1)*pageSize : end],
	}, nil
}
