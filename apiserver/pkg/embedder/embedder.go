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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/pointer"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	graphqlutils "github.com/kubeagi/arcadia/apiserver/pkg/utils"
	"github.com/kubeagi/arcadia/apiserver/pkg/worker"
	"github.com/kubeagi/arcadia/pkg/embeddings"
	"github.com/kubeagi/arcadia/pkg/utils"
)

func embedder2modelConverter(ctx context.Context, c dynamic.Interface) func(*unstructured.Unstructured) (generated.PageNode, error) {
	return func(u *unstructured.Unstructured) (generated.PageNode, error) {
		return Embedder2model(ctx, c, u)
	}
}

// Embedder2model convert unstructured `CR Embedder` to graphql model `Embedder`
func Embedder2model(ctx context.Context, c dynamic.Interface, obj *unstructured.Unstructured) (*generated.Embedder, error) {
	embedder := &v1alpha1.Embedder{}
	if err := utils.UnstructuredToStructured(obj, embedder); err != nil {
		return nil, err
	}

	id := string(embedder.GetUID())
	creationtimestamp := embedder.GetCreationTimestamp().Time

	embedderType := string(embedder.Spec.Type)
	provider := string(embedder.Spec.Provider.GetType())

	// conditioned status
	condition := embedder.Status.GetCondition(v1alpha1.TypeReady)
	updateTime := condition.LastTransitionTime.Time
	status := common.GetObjStatus(embedder)
	message := string(condition.Message)
	// Use worker's status&message if LLM's provider is `Worker`
	if embedder.Spec.Provider.GetType() == v1alpha1.ProviderTypeWorker {
		w, err := worker.ReadWorker(ctx, c, embedder.Name, embedder.Namespace)
		if err == nil {
			status = *w.Status
			message = *w.Message
		}
	}

	// get embedder's api url
	var baseURL string
	switch embedder.Spec.Provider.GetType() {
	case v1alpha1.ProviderTypeWorker:
		baseURL, _ = common.GetAPIServer(ctx, c, true)
	case v1alpha1.ProviderType3rdParty:
		baseURL = embedder.Spec.Endpoint.URL
	}

	md := generated.Embedder{
		ID:                &id,
		Name:              obj.GetName(),
		Namespace:         obj.GetNamespace(),
		Creator:           pointer.String(embedder.Spec.Creator),
		CreationTimestamp: &creationtimestamp,
		Labels:            graphqlutils.MapStr2Any(obj.GetLabels()),
		Annotations:       graphqlutils.MapStr2Any(obj.GetAnnotations()),
		DisplayName:       &embedder.Spec.DisplayName,
		Description:       &embedder.Spec.Description,
		Type:              &embedderType,
		Provider:          &provider,
		BaseURL:           baseURL,
		Models:            embedder.GetModelList(),
		Status:            &status,
		Message:           &message,
		UpdateTimestamp:   &updateTime,
	}
	return &md, nil
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
				Endpoint: &v1alpha1.Endpoint{
					URL: input.Endpointinput.URL,
				},
			},
			Type: embeddings.EmbeddingType(servicetype),
		},
	}

	// create auth secret
	if input.Endpointinput.Auth != nil {
		// create auth secret
		secret := common.MakeAuthSecretName(embedder.Name, "embedder")
		err := common.MakeAuthSecret(ctx, c, generated.TypedObjectReferenceInput{
			Kind:      "Secret",
			Name:      secret,
			Namespace: &input.Namespace,
		}, input.Endpointinput.Auth, nil)
		if err != nil {
			return nil, err
		}
		embedder.Spec.Endpoint.AuthSecret = &v1alpha1.TypedObjectReference{
			Kind:      "Secret",
			Name:      secret,
			Namespace: &input.Namespace,
		}
	}
	common.SetCreator(ctx, &embedder.Spec.CommonSpec)

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
			Kind:      "Secret",
			Name:      secret,
			Namespace: &input.Namespace,
		}, input.Endpointinput.Auth, obj)
		if err != nil {
			return nil, err
		}
	}

	return Embedder2model(ctx, c, obj)
}

func UpdateEmbedder(ctx context.Context, c dynamic.Interface, input *generated.UpdateEmbedderInput) (*generated.Embedder, error) {
	obj, err := common.ResouceGet(ctx, c, generated.TypedObjectReferenceInput{
		APIGroup:  &common.ArcadiaAPIGroup,
		Kind:      "Embedder",
		Name:      input.Name,
		Namespace: &input.Namespace,
	}, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	updatedEmbedder := &v1alpha1.Embedder{}
	if err := utils.UnstructuredToStructured(obj, updatedEmbedder); err != nil {
		return nil, err
	}

	updatedEmbedder.SetLabels(graphqlutils.MapAny2Str(input.Labels))
	updatedEmbedder.SetAnnotations(graphqlutils.MapAny2Str(input.Annotations))

	if input.DisplayName != nil {
		updatedEmbedder.Spec.CommonSpec.DisplayName = *input.DisplayName
	}
	if input.Description != nil {
		updatedEmbedder.Spec.CommonSpec.Description = *input.Description
	}
	if input.Type != nil {
		updatedEmbedder.Spec.Type = embeddings.EmbeddingType(*input.Type)
	}

	// Update endpoint
	if input.Endpointinput != nil {
		endpoint, err := common.MakeEndpoint(ctx, c, generated.TypedObjectReferenceInput{
			APIGroup:  &updatedEmbedder.APIVersion,
			Kind:      updatedEmbedder.Kind,
			Name:      updatedEmbedder.Name,
			Namespace: &updatedEmbedder.Namespace,
		}, *input.Endpointinput)
		if err != nil {
			return nil, err
		}
		updatedEmbedder.Spec.Provider.Endpoint = &endpoint
	}

	unstructuredEmbedder, err := runtime.DefaultUnstructuredConverter.ToUnstructured(updatedEmbedder)
	if err != nil {
		return nil, err
	}

	updatedObject, err := common.ResouceUpdate(ctx, c, generated.TypedObjectReferenceInput{
		APIGroup:  &common.ArcadiaAPIGroup,
		Kind:      "Embedder",
		Namespace: &updatedEmbedder.Namespace,
		Name:      updatedEmbedder.Name,
	}, unstructuredEmbedder, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return Embedder2model(ctx, c, updatedObject)
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

func ListEmbedders(ctx context.Context, c dynamic.Interface, input generated.ListCommonInput, listOpts ...common.ListOptionsFunc) (*generated.PaginatedResult, error) {
	// listOpts in this graphql query
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

	dsSchema := schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "embedders"}
	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	}
	us, err := c.Resource(dsSchema).Namespace(input.Namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}
	filter := make([]common.ResourceFilter, 0)
	if keyword != "" {
		filter = append(filter, common.FilterEmbedderByKeyword(keyword))
	}
	result, err := common.ListReources(us, page, pageSize, embedder2modelConverter(ctx, c), filter...)
	if err != nil {
		return nil, err
	}

	for i := range result.Nodes {
		result.Nodes[i] = opts.ConvertFunc(result.Nodes[i])
	}
	return result, nil
}

func ReadEmbedder(ctx context.Context, c dynamic.Interface, name, namespace string) (*generated.Embedder, error) {
	resource := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "embedders"})
	u, err := resource.Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return Embedder2model(ctx, c, u)
}
