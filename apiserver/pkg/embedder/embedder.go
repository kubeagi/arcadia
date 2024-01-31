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
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	graphqlutils "github.com/kubeagi/arcadia/apiserver/pkg/utils"
	"github.com/kubeagi/arcadia/apiserver/pkg/worker"
	"github.com/kubeagi/arcadia/pkg/embeddings"
)

func embedder2modelConverter(ctx context.Context, c client.Client) func(obj client.Object) (generated.PageNode, error) {
	return func(obj client.Object) (generated.PageNode, error) {
		u, ok := obj.(*v1alpha1.Embedder)
		if !ok {
			return nil, errors.New("can't convert obj to Embedder")
		}
		return Embedder2model(ctx, c, u)
	}
}

// Embedder2model convert unstructured `CR Embedder` to graphql model `Embedder`
func Embedder2model(ctx context.Context, c client.Client, embedder *v1alpha1.Embedder) (*generated.Embedder, error) {
	id := string(embedder.GetUID())
	creationtimestamp := embedder.GetCreationTimestamp().Time

	embedderType := string(embedder.Spec.Type)
	provider := string(embedder.Spec.Provider.GetType())

	// conditioned status
	condition := embedder.Status.GetCondition(v1alpha1.TypeReady)
	updateTime := condition.LastTransitionTime.Time
	status := common.GetObjStatus(embedder)
	message := string(condition.Message)
	// Use worker's status&message if Embedder's provider is `Worker`
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
		Name:              embedder.GetName(),
		Namespace:         embedder.GetNamespace(),
		Creator:           pointer.String(embedder.Spec.Creator),
		CreationTimestamp: &creationtimestamp,
		Labels:            graphqlutils.MapStr2Any(embedder.GetLabels()),
		Annotations:       graphqlutils.MapStr2Any(embedder.GetAnnotations()),
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

func CreateEmbedder(ctx context.Context, c client.Client, input generated.CreateEmbedderInput) (*generated.Embedder, error) {
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

	embedder := &v1alpha1.Embedder{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
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
			Type:   embeddings.EmbeddingType(servicetype),
			Models: input.Models,
		},
	}

	// create auth secret
	if input.Endpointinput.Auth != nil {
		// create auth secret
		secret := common.GenerateAuthSecretName(embedder.Name, "embedder")
		err := common.MakeAuthSecret(ctx, c, input.Namespace, secret, input.Endpointinput.Auth, nil)
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

	err := c.Create(ctx, embedder)
	if err != nil {
		return nil, err
	}

	// update auth secret owner reference
	if input.Endpointinput.Auth != nil {
		// user embedder as the owner
		secret := common.GenerateAuthSecretName(embedder.Name, "embedder")
		err := common.MakeAuthSecret(ctx, c, input.Namespace, secret, input.Endpointinput.Auth, embedder)
		if err != nil {
			return nil, err
		}
	}

	return Embedder2model(ctx, c, embedder)
}

func UpdateEmbedder(ctx context.Context, c client.Client, input *generated.UpdateEmbedderInput) (*generated.Embedder, error) {
	updatedEmbedder := &v1alpha1.Embedder{}
	err := c.Get(ctx, types.NamespacedName{Namespace: input.Namespace, Name: input.Name}, updatedEmbedder)
	if err != nil {
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

	// update Embedder's models if specified
	if input.Models != nil {
		updatedEmbedder.Spec.Models = input.Models
	}

	// Update endpoint
	if input.Endpointinput != nil {
		endpoint, err := common.MakeEndpoint(ctx, c, updatedEmbedder, *input.Endpointinput)
		if err != nil {
			return nil, err
		}
		updatedEmbedder.Spec.Provider.Endpoint = &endpoint
	}
	err = c.Update(ctx, updatedEmbedder)
	if err != nil {
		return nil, err
	}
	return Embedder2model(ctx, c, updatedEmbedder)
}

func DeleteEmbedders(ctx context.Context, c client.Client, input *generated.DeleteCommonInput) (*string, error) {
	opts, err := common.DeleteAllOptions(input)
	if err != nil {
		return nil, err
	}
	if err := c.DeleteAllOf(ctx, &v1alpha1.Embedder{}, opts...); err != nil {
		return nil, err
	}
	return pointer.String("ok"), nil
}

func ListEmbedders(ctx context.Context, c client.Client, input generated.ListCommonInput, listOpts ...common.ListOptionsFunc) (*generated.PaginatedResult, error) {
	// listOpts in this graphql query
	opts := common.DefaultListOptions()
	for _, optFunc := range listOpts {
		optFunc(opts)
	}

	keyword := ""
	page, pageSize := 1, 10
	if input.Keyword != nil {
		keyword = *input.Keyword
	}
	if input.Page != nil && *input.Page > 0 {
		page = *input.Page
	}
	if input.PageSize != nil {
		pageSize = *input.PageSize
	}

	us := &v1alpha1.EmbedderList{}
	list, err := common.NewListOptions(input)
	if err != nil {
		return nil, err
	}
	err = c.List(ctx, us, list...)
	if err != nil {
		return nil, err
	}

	filter := make([]common.ResourceFilter, 0)
	if keyword != "" {
		filter = append(filter, common.FilterEmbedderByKeyword(keyword))
	}
	items := make([]client.Object, len(us.Items))
	for i := range us.Items {
		items[i] = &us.Items[i]
	}
	result, err := common.ListReources(items, page, pageSize, embedder2modelConverter(ctx, c), filter...)
	if err != nil {
		return nil, err
	}

	for i := range result.Nodes {
		result.Nodes[i] = opts.ConvertFunc(result.Nodes[i])
	}
	return result, nil
}

func ReadEmbedder(ctx context.Context, c client.Client, name, namespace string) (*generated.Embedder, error) {
	u := &v1alpha1.Embedder{}
	err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, u)
	if err != nil {
		return nil, err
	}
	return Embedder2model(ctx, c, u)
}
