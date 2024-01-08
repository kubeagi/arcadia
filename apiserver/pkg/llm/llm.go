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

package llm

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
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	graphqlutils "github.com/kubeagi/arcadia/apiserver/pkg/utils"
	"github.com/kubeagi/arcadia/apiserver/pkg/worker"
	"github.com/kubeagi/arcadia/pkg/llms"
	"github.com/kubeagi/arcadia/pkg/utils"
)

// LLM2model convert unstructured `CR LLM` to graphql model `Llm`
func LLM2model(ctx context.Context, c dynamic.Interface, obj *unstructured.Unstructured) *generated.Llm {
	llm := &v1alpha1.LLM{}
	if err := utils.UnstructuredToStructured(obj, llm); err != nil {
		return &generated.Llm{}
	}

	id := string(llm.GetUID())
	creationtimestamp := llm.GetCreationTimestamp().Time

	llmType := string(llm.Spec.Type)
	provider := string(llm.Spec.Provider.GetType())

	// conditioned status
	condition := llm.Status.GetCondition(v1alpha1.TypeReady)
	updateTime := condition.LastTransitionTime.Time
	status := common.GetObjStatus(llm)
	message := string(condition.Message)
	// Use worker's status&message if LLM's provider is `Worker`
	if llm.Spec.Provider.GetType() == v1alpha1.ProviderTypeWorker {
		w, err := worker.ReadWorker(ctx, c, llm.Name, llm.Namespace)
		if err == nil {
			if w.Status != nil {
				status = *w.Status
			}
			if w.Message != nil {
				message = *w.Message
			}
		}
	}

	// get llm's api url
	var baseURL string
	switch llm.Spec.Provider.GetType() {
	case v1alpha1.ProviderTypeWorker:
		baseURL, _ = common.GetAPIServer(ctx, c, true)
	case v1alpha1.ProviderType3rdParty:
		baseURL = llm.Spec.Endpoint.URL
	}

	md := generated.Llm{
		ID:                &id,
		Name:              obj.GetName(),
		Namespace:         obj.GetNamespace(),
		Creator:           &llm.Spec.Creator,
		CreationTimestamp: &creationtimestamp,
		Labels:            graphqlutils.MapStr2Any(obj.GetLabels()),
		Annotations:       graphqlutils.MapStr2Any(obj.GetAnnotations()),
		DisplayName:       &llm.Spec.DisplayName,
		Description:       &llm.Spec.Description,
		Type:              &llmType,
		Status:            &status,
		Message:           &message,
		Provider:          &provider,
		BaseURL:           baseURL,
		Models:            llm.GetModelList(),
		UpdateTimestamp:   &updateTime,
	}
	return &md
}

// ListLLMs return a list of LLMs based on input params
func ListLLMs(ctx context.Context, c dynamic.Interface, input generated.ListCommonInput, listOpts ...common.ListOptionsFunc) (*generated.PaginatedResult, error) {
	opts := common.DefaultListOptions()
	for _, optFunc := range listOpts {
		optFunc(opts)
	}

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
	if input.PageSize != nil {
		pageSize = *input.PageSize
	}

	us, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "LLM")).Namespace(input.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(us.Items, func(i, j int) bool {
		return us.Items[i].GetCreationTimestamp().After(us.Items[j].GetCreationTimestamp().Time)
	})

	result := make([]generated.PageNode, 0, len(us.Items))
	for _, u := range us.Items {
		m := LLM2model(ctx, c, &u)
		if keyword != "" && !strings.Contains(m.Name, keyword) && !strings.Contains(*m.DisplayName, keyword) {
			continue
		}
		result = append(result, opts.ConvertFunc(m))
	}
	totalCount := len(result)
	pageStart, end := common.PagePosition(page, pageSize, totalCount)
	return &generated.PaginatedResult{
		TotalCount:  totalCount,
		HasNextPage: end < totalCount,
		Nodes:       result[pageStart:end],
	}, nil
}

// ReadLLM
func ReadLLM(ctx context.Context, c dynamic.Interface, name, namespace string) (*generated.Llm, error) {
	resource, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "LLM")).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return LLM2model(ctx, c, resource), nil
}

func CreateLLM(ctx context.Context, c dynamic.Interface, input generated.CreateLLMInput) (*generated.Llm, error) {
	displayName, description, APIType := "", "", ""
	if input.DisplayName != nil {
		displayName = *input.DisplayName
	}
	if input.Description != nil {
		description = *input.Description
	}
	if input.Type != nil {
		APIType = *input.Type
	}

	llm := v1alpha1.LLM{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "LLM",
			APIVersion: v1alpha1.GroupVersion.String(),
		},
		Spec: v1alpha1.LLMSpec{
			CommonSpec: v1alpha1.CommonSpec{
				DisplayName: displayName,
				Description: description,
			},
			Provider: v1alpha1.Provider{
				Endpoint: &v1alpha1.Endpoint{
					URL: input.Endpointinput.URL,
				},
			},
			Type: llms.LLMType(APIType),
		},
	}
	common.SetCreator(ctx, &llm.Spec.CommonSpec)

	// create auth secret
	if input.Endpointinput.Auth != nil {
		secret := common.MakeAuthSecretName(llm.Name, "llm")
		err := common.MakeAuthSecret(ctx, c, generated.TypedObjectReferenceInput{
			Kind:      "Secret",
			Name:      secret,
			Namespace: &input.Namespace,
		}, input.Endpointinput.Auth, nil)
		if err != nil {
			return nil, err
		}
		llm.Spec.Endpoint.AuthSecret = &v1alpha1.TypedObjectReference{
			Kind:      "Secret",
			Name:      secret,
			Namespace: &input.Namespace,
		}
	}

	unstructuredLLM, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&llm)
	if err != nil {
		return nil, err
	}

	obj, err := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "llms"}).
		Namespace(input.Namespace).Create(ctx, &unstructured.Unstructured{Object: unstructuredLLM}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	// update auth secret owner reference
	if input.Endpointinput.Auth != nil {
		// user obj as the owner
		secret := common.MakeAuthSecretName(llm.Name, "LLM")
		err := common.MakeAuthSecret(ctx, c, generated.TypedObjectReferenceInput{
			Kind:      "Secret",
			Name:      secret,
			Namespace: &input.Namespace,
		}, input.Endpointinput.Auth, obj)
		if err != nil {
			return nil, err
		}
	}

	// create *generated.Llm
	genLLM := LLM2model(ctx, c, obj)
	return genLLM, nil
}

func UpdateLLM(ctx context.Context, c dynamic.Interface, input *generated.UpdateLLMInput) (*generated.Llm, error) {
	obj, err := common.ResouceGet(ctx, c, generated.TypedObjectReferenceInput{
		APIGroup:  &common.ArcadiaAPIGroup,
		Kind:      "LLM",
		Name:      input.Name,
		Namespace: &input.Namespace,
	}, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	updatedLLM := &v1alpha1.LLM{}
	if err := utils.UnstructuredToStructured(obj, updatedLLM); err != nil {
		return nil, err
	}

	updatedLLM.SetLabels(graphqlutils.MapAny2Str(input.Labels))
	updatedLLM.SetAnnotations(graphqlutils.MapAny2Str(input.Annotations))

	if input.DisplayName != nil {
		updatedLLM.Spec.CommonSpec.DisplayName = *input.DisplayName
	}
	if input.Description != nil {
		updatedLLM.Spec.CommonSpec.Description = *input.Description
	}
	if input.Type != nil {
		updatedLLM.Spec.Type = llms.LLMType(*input.Type)
	}

	// Update endpoint
	if input.Endpointinput != nil {
		endpoint, err := common.MakeEndpoint(ctx, c, generated.TypedObjectReferenceInput{
			APIGroup:  &updatedLLM.APIVersion,
			Kind:      updatedLLM.Kind,
			Name:      updatedLLM.Name,
			Namespace: &updatedLLM.Namespace,
		}, *input.Endpointinput)
		if err != nil {
			return nil, err
		}
		updatedLLM.Spec.Provider.Endpoint = &endpoint
	}

	unstructuredLLM, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&updatedLLM)
	if err != nil {
		return nil, err
	}

	updatedObject, err := common.ResouceUpdate(ctx, c, generated.TypedObjectReferenceInput{
		APIGroup:  &common.ArcadiaAPIGroup,
		Kind:      "LLM",
		Namespace: &updatedLLM.Namespace,
		Name:      updatedLLM.Name,
	}, unstructuredLLM, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	ds := LLM2model(ctx, c, updatedObject)
	return ds, nil
}

func DeleteLLMs(ctx context.Context, c dynamic.Interface, input *generated.DeleteCommonInput) (*string, error) {
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

	resource := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "llms"})
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
