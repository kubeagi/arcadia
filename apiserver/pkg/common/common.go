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

package common

import (
	"context"
	"errors"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/auth"
	"github.com/kubeagi/arcadia/pkg/config"
	"github.com/kubeagi/arcadia/pkg/datasource"
)

var (
	DefaultNamespace  = "default"
	ErrNoResourceKind = errors.New("must provide resource kind")

	// Common status
	StatusTrue  = "True"
	StatusFalse = "False"
)

// ModelType
var (
	ModelTypeAll       = "llm,embedding"
	ModelTypeLLM       = "llm"
	ModelTypeEmbedding = "embedding"
)

// ModelSource
var (
	ModelSourceLocal       = "local"
	ModelSourceModelscope  = "modelscope"
	ModelSourceHuggingface = "huggingface"
)

func SystemDatasourceOSS(ctx context.Context, mgrClient client.Client) (*datasource.OSS, error) {
	systemDatasource, err := config.GetSystemDatasource(ctx, mgrClient)
	if err != nil {
		return nil, err
	}
	endpoint := systemDatasource.Spec.Endpoint.DeepCopy()
	if endpoint.AuthSecret != nil && endpoint.AuthSecret.Namespace == nil {
		endpoint.AuthSecret.WithNameSpace(systemDatasource.Namespace)
	}
	return datasource.NewOSS(ctx, mgrClient, endpoint)
}

// SystemEmbeddingSuite returns the embedder and vectorstore which are built-in in system config
// Embedder and vectorstore are both required when generating a new embedding.That's why we call it a `EmbeddingSuit`
func SystemEmbeddingSuite(ctx context.Context, cli client.Client) (*v1alpha1.Embedder, *v1alpha1.VectorStore, error) {
	// get the built-in system embedder
	emd, err := config.GetEmbedder(ctx, cli)
	if err != nil {
		return nil, nil, err
	}
	embedder := &v1alpha1.Embedder{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: *emd.Namespace, Name: emd.Name}, embedder); err != nil {
		return nil, nil, err
	}
	// get the built-in system vectorstore
	vs, err := config.GetVectorStore(ctx, cli)
	if err != nil {
		return nil, nil, err
	}
	vectorStore := &v1alpha1.VectorStore{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: *vs.Namespace, Name: vs.Name}, vectorStore); err != nil {
		return nil, nil, err
	}
	return embedder, vectorStore, nil
}

// GetAPIServer returns the api server url to access arcadia's worker
// if external is true,then this func will return the external api server
func GetAPIServer(ctx context.Context, cli client.Client, external bool) (string, error) {
	gateway, err := config.GetGateway(ctx, cli)
	if err != nil {
		return "", err
	}
	api := gateway.APIServer
	if external {
		api = gateway.ExternalAPIServer
	}
	return api, nil
}

// GetObjStatus is used to calculate the state of the resource, unified management,
// in general, a resource will only record its own state,
// then the state calculation of this resource, should be written to this function.
// But for some special resources. For example, VersionedDataset,
// he needs to calculate multiple states, it is not suitable for this function.
func GetObjStatus(obj client.Object) string {
	if obj == nil {
		return ""
	}
	if obj.GetDeletionTimestamp() != nil {
		return "Deleting"
	}

	var (
		condition v1alpha1.Condition
	)
	switch v := obj.(type) {
	case *v1alpha1.Datasource:
		condition = v.Status.GetCondition(v1alpha1.TypeReady)
	case *v1alpha1.Embedder:
		condition = v.Status.GetCondition(v1alpha1.TypeReady)
	case *v1alpha1.KnowledgeBase:
		condition = v.Status.GetCondition(v1alpha1.TypeReady)
	case *v1alpha1.LLM:
		condition = v.Status.GetCondition(v1alpha1.TypeReady)
	case *v1alpha1.Model:
		condition = v.Status.GetCondition(v1alpha1.TypeReady)
	case *v1alpha1.Worker:
		condition = v.Status.GetCondition(v1alpha1.TypeReady)
		// Worker can better represent the state of resources through Reason.
		status := string(condition.Reason)
		// When replicas is zero but status is not `Offline`, it must be in `OfflineInProgress`
		if v.Spec.Replicas == nil || *v.Spec.Replicas == 0 {
			if status != "Offline" {
				status = "OfflineInProgress"
			}
		}
		return status
	case *v1alpha1.Application:
		condition = v.Status.GetCondition(v1alpha1.TypeReady)
	default:
		return ""
	}

	return string(condition.Status)
}

// PageNodeConvertFunc convert `any` to a `PageNode`
type PageNodeConvertFunc func(any) generated.PageNode

var (
	DefaultPageNodeConvertFunc = func(node any) generated.PageNode {
		pageNode, ok := node.(generated.PageNode)
		if !ok {
			return nil
		}
		return pageNode
	}
)

var (
	// UnlimitedPageSize which means all
	UnlimitedPageSize = -1
)

// ListOptions for graphql list
type ListOptions struct {
	ConvertFunc PageNodeConvertFunc
}

// DefaultListOptions initialize a ListOptions with default settings
func DefaultListOptions() *ListOptions {
	return &ListOptions{
		ConvertFunc: DefaultPageNodeConvertFunc,
	}
}

type ListOptionsFunc func(options *ListOptions)

// WithPageNodeConvertFunc update the PageNodeConvertFunc
func WithPageNodeConvertFunc(convertFunc PageNodeConvertFunc) ListOptionsFunc {
	return func(option *ListOptions) {
		option.ConvertFunc = convertFunc
	}
}

func SetCreator(ctx context.Context, common *v1alpha1.CommonSpec) {
	currentUser, _ := ctx.Value(auth.UserNameContextKey).(string)
	common.Creator = currentUser
}

func PagePosition(page, size, total int) (int, int) {
	// if pageSize is -1 which means unlimited pageSize,return all
	if size == UnlimitedPageSize {
		page = 1
		size = total
	}
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	start := (page - 1) * size
	if start > total {
		start = total
	}
	end := page * size
	if end > total {
		end = total
	}
	return start, end
}

func TypedObjectReferenceToInput(ref generated.TypedObjectReference) generated.TypedObjectReferenceInput {
	return generated.TypedObjectReferenceInput{
		APIGroup:  ref.APIGroup,
		Kind:      ref.Kind,
		Namespace: ref.Namespace,
		Name:      ref.Name,
	}
}

func GetAppCategory(app *v1alpha1.Application) []*string {
	category := make([]*string, 0)
	categoryStr, ok := app.GetAnnotations()[v1alpha1.AppCategoryAnnotationKey]
	if ok && len(categoryStr) > 0 {
		for _, v := range strings.Split(categoryStr, ",") {
			v := v
			category = append(category, pointer.String(strings.TrimSpace(v)))
		}
	}
	return category
}

func DeleteAllOptions(input *generated.DeleteCommonInput) ([]client.DeleteAllOfOption, error) {
	if input.Namespace == "" {
		return nil, errors.New("namespace is empty, please check your request args")
	}
	if pointer.StringDeref(input.Name, "") == "" && pointer.StringDeref(input.LabelSelector, "") == "" && pointer.StringDeref(input.FieldSelector, "") == "" {
		return nil, errors.New("no name, no LabelSelector, no FieldSelector, please check your request args")
	}
	opts := []client.DeleteAllOfOption{
		client.InNamespace(input.Namespace),
	}
	fieldsSelector := ""
	if pointer.StringDeref(input.Name, "") != "" {
		fieldsSelector = fmt.Sprintf("metadata.name=%s", *input.Name)
	}
	if pointer.StringDeref(input.FieldSelector, "") != "" {
		fieldsSelector = *input.FieldSelector
	}
	if fieldsSelector != "" {
		f, err := fields.ParseSelector(fieldsSelector)
		if err != nil {
			return nil, err
		}
		opts = append(opts, client.MatchingFieldsSelector{Selector: f})
	}
	if pointer.StringDeref(input.LabelSelector, "") != "" {
		l, err := labels.Parse(*input.LabelSelector)
		if err != nil {
			return nil, err
		}
		opts = append(opts, client.MatchingLabelsSelector{Selector: l})
	}
	return opts, nil
}

func NewListOptions(input generated.ListCommonInput) ([]client.ListOption, error) {
	if input.Namespace == "" {
		input.Namespace = metav1.NamespaceDefault
	}
	opts := []client.ListOption{
		client.InNamespace(input.Namespace),
	}
	if pointer.StringDeref(input.LabelSelector, "") != "" {
		l, err := labels.Parse(*input.LabelSelector)
		if err != nil {
			return nil, err
		}
		opts = append(opts, client.MatchingLabelsSelector{Selector: l})
	}
	if pointer.StringDeref(input.FieldSelector, "") != "" {
		f, err := fields.ParseSelector(*input.FieldSelector)
		if err != nil {
			return nil, err
		}
		opts = append(opts, client.MatchingFieldsSelector{Selector: f})
	}
	return opts, nil
}
