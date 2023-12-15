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
	"k8s.io/client-go/dynamic"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	graphqlutils "github.com/kubeagi/arcadia/apiserver/pkg/utils"
	"github.com/kubeagi/arcadia/pkg/utils"
)

func unstructured2LLM(ctx context.Context, c dynamic.Interface, obj *unstructured.Unstructured) *generated.Llm {
	llm := &v1alpha1.LLM{}
	if err := utils.UnstructuredToStructured(obj, llm); err != nil {
		return &generated.Llm{}
	}

	id := string(llm.GetUID())
	creationtimestamp := llm.GetCreationTimestamp().Time

	// conditioned status
	condition := llm.Status.GetCondition(v1alpha1.TypeReady)
	updateTime := condition.LastTransitionTime.Time
	status := common.GetObjStatus(llm)
	message := string(condition.Message)

	llmType := string(llm.Spec.Type)
	provider := string(llm.Spec.Provider.GetType())

	// get llm's api url
	var baseURL string
	switch llm.Spec.Provider.GetType() {
	case v1alpha1.ProviderTypeWorker:
		baseURL, _ = common.GetAPIServer(ctx, c, true)
	case v1alpha1.ProviderType3rdParty:
		baseURL = llm.Spec.Enpoint.URL
	}

	md := generated.Llm{
		ID:                &id,
		Name:              obj.GetName(),
		Namespace:         obj.GetNamespace(),
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

func ListLLMs(ctx context.Context, c dynamic.Interface, input generated.ListCommonInput) (*generated.PaginatedResult, error) {
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

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	}

	us, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "LLM")).Namespace(input.Namespace).List(ctx, listOptions)
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
		m := unstructured2LLM(ctx, c, &u)
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

// ReadLLM
func ReadLLM(ctx context.Context, c dynamic.Interface, name, namespace string) (*generated.Llm, error) {
	resource, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "LLM")).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return unstructured2LLM(ctx, c, resource), nil
}
