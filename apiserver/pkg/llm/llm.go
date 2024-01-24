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

func LLM2modelConverter(ctx context.Context, c dynamic.Interface) func(*unstructured.Unstructured) (generated.PageNode, error) {
	return func(u *unstructured.Unstructured) (generated.PageNode, error) {
		return LLM2model(ctx, c, u)
	}
}

// LLM2model convert unstructured `CR LLM` to graphql model `Llm`
func LLM2model(ctx context.Context, c dynamic.Interface, obj *unstructured.Unstructured) (*generated.Llm, error) {
	llm := &v1alpha1.LLM{}
	if err := utils.UnstructuredToStructured(obj, llm); err != nil {
		return nil, err
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
	return &md, nil
}

// ListLLMs return a list of LLMs based on input params
func ListLLMs(ctx context.Context, c dynamic.Interface, input generated.ListCommonInput, listOpts ...common.ListOptionsFunc) (*generated.PaginatedResult, error) {
	opts := common.DefaultListOptions()
	for _, optFunc := range listOpts {
		optFunc(opts)
	}

	labelSelector, fieldSelector := "", ""
	page, pageSize := 1, 10
	filter := make([]common.ResourceFilter, 0)
	if input.Keyword != nil {
		filter = append(filter, common.FilterLLMByKeyword(*input.Keyword))
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
	result, err := common.ListReources(us, page, pageSize, LLM2modelConverter(ctx, c), filter...)
	if err != nil {
		return nil, err
	}

	for i := range result.Nodes {
		result.Nodes[i] = opts.ConvertFunc(result.Nodes[i])
	}
	return result, nil
}

// ReadLLM
func ReadLLM(ctx context.Context, c dynamic.Interface, name, namespace string) (*generated.Llm, error) {
	resource, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "LLM")).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return LLM2model(ctx, c, resource)
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
			Type:   llms.LLMType(APIType),
			Models: input.Models,
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
	return LLM2model(ctx, c, obj)
}

func UpdateLLM(ctx context.Context, c dynamic.Interface, input *generated.UpdateLLMInput) (*generated.Llm, error) {
	obj, err := common.ResourceGet(ctx, c, generated.TypedObjectReferenceInput{
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

	// update LLM's models if specified
	if input.Models != nil {
		updatedLLM.Spec.Models = input.Models
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
	return LLM2model(ctx, c, updatedObject)
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
