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
	"errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	graphqlutils "github.com/kubeagi/arcadia/apiserver/pkg/utils"
	"github.com/kubeagi/arcadia/apiserver/pkg/worker"
	"github.com/kubeagi/arcadia/pkg/llms"
)

func LLM2modelConverter(ctx context.Context, c client.Client) func(object client.Object) (generated.PageNode, error) {
	return func(u client.Object) (generated.PageNode, error) {
		llm, ok := u.(*v1alpha1.LLM)
		if !ok {
			return nil, errors.New("can't convert object to LLM")
		}
		return LLM2model(ctx, c, llm)
	}
}

// LLM2model convert unstructured `CR LLM` to graphql model `Llm`
func LLM2model(ctx context.Context, c client.Client, llm *v1alpha1.LLM) (*generated.Llm, error) {
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
		Name:              llm.GetName(),
		Namespace:         llm.GetNamespace(),
		Creator:           &llm.Spec.Creator,
		CreationTimestamp: &creationtimestamp,
		Labels:            graphqlutils.MapStr2Any(llm.GetLabels()),
		Annotations:       graphqlutils.MapStr2Any(llm.GetAnnotations()),
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
func ListLLMs(ctx context.Context, c client.Client, input generated.ListCommonInput, listOpts ...common.ListOptionsFunc) (*generated.PaginatedResult, error) {
	opts := common.DefaultListOptions()
	for _, optFunc := range listOpts {
		optFunc(opts)
	}

	page, pageSize := 1, 10
	filter := make([]common.ResourceFilter, 0)
	if input.Keyword != nil {
		filter = append(filter, common.FilterLLMByKeyword(*input.Keyword))
	}
	if input.Page != nil && *input.Page > 0 {
		page = *input.Page
	}
	if input.PageSize != nil {
		pageSize = *input.PageSize
	}

	us := &v1alpha1.LLMList{}
	options, err := common.NewListOptions(input)
	if err != nil {
		return nil, err
	}
	err = c.List(ctx, us, options...)
	if err != nil {
		return nil, err
	}
	items := make([]client.Object, len(us.Items))
	for i := range us.Items {
		items[i] = &us.Items[i]
	}
	result, err := common.ListReources(items, page, pageSize, LLM2modelConverter(ctx, c), filter...)
	if err != nil {
		return nil, err
	}

	for i := range result.Nodes {
		result.Nodes[i] = opts.ConvertFunc(result.Nodes[i])
	}
	return result, nil
}

// ReadLLM
func ReadLLM(ctx context.Context, c client.Client, name, namespace string) (*generated.Llm, error) {
	llm := &v1alpha1.LLM{}
	err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, llm)
	if err != nil {
		return nil, err
	}
	return LLM2model(ctx, c, llm)
}

func CreateLLM(ctx context.Context, c client.Client, input generated.CreateLLMInput) (*generated.Llm, error) {
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

	llm := &v1alpha1.LLM{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
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
		secret := common.GenerateAuthSecretName(llm.Name, "llm")
		err := common.MakeAuthSecret(ctx, c, input.Namespace, secret, input.Endpointinput.Auth, nil)
		if err != nil {
			return nil, err
		}
		llm.Spec.Endpoint.AuthSecret = &v1alpha1.TypedObjectReference{
			Kind:      "Secret",
			Name:      secret,
			Namespace: &input.Namespace,
		}
	}

	err := c.Create(ctx, llm)
	if err != nil {
		return nil, err
	}

	// update auth secret owner reference
	if input.Endpointinput.Auth != nil {
		// user obj as the owner
		secret := common.GenerateAuthSecretName(llm.Name, "LLM")
		err := common.MakeAuthSecret(ctx, c, input.Namespace, secret, input.Endpointinput.Auth, llm)
		if err != nil {
			return nil, err
		}
	}

	// create *generated.Llm
	return LLM2model(ctx, c, llm)
}

func UpdateLLM(ctx context.Context, c client.Client, input *generated.UpdateLLMInput) (*generated.Llm, error) {
	updatedLLM := &v1alpha1.LLM{}
	err := c.Get(ctx, types.NamespacedName{Namespace: input.Namespace, Name: input.Name}, updatedLLM)
	if err != nil {
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
		endpoint, err := common.MakeEndpoint(ctx, c, updatedLLM, *input.Endpointinput)
		if err != nil {
			return nil, err
		}
		updatedLLM.Spec.Provider.Endpoint = &endpoint
	}

	err = c.Update(ctx, updatedLLM)
	if err != nil {
		return nil, err
	}
	return LLM2model(ctx, c, updatedLLM)
}

func DeleteLLMs(ctx context.Context, c client.Client, input *generated.DeleteCommonInput) (*string, error) {
	opts, err := common.DeleteAllOptions(input)
	if err != nil {
		return nil, err
	}
	err = c.DeleteAllOf(ctx, &v1alpha1.LLM{}, opts...)
	return nil, err
}
