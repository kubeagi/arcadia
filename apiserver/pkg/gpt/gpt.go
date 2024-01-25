/*
Copyright 2024 KubeAGI.

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

package gpt

import (
	"context"
	"errors"
	"fmt"
	"strings"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/pointer"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/chat"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
)

func app2gpt(app *v1alpha1.Application) (*generated.Gpt, error) {
	if app == nil {
		return nil, errors.New("no app found")
	}

	gpt := &generated.Gpt{
		Name:        pointer.String(strings.Join([]string{app.Namespace, app.Name}, "/")),
		DisplayName: pointer.String(app.Spec.DisplayName),
		Description: pointer.String(app.Spec.Description),
		Hot:         pointer.Int(getHot(app)),
		Creator:     pointer.String(app.Spec.Creator),
		Category:    common.GetAppCategory(app),
		Icon:        pointer.String(app.Spec.Icon),
		Prologue:    pointer.String(app.Spec.Prologue),
	}
	return gpt, nil
}

func unstructred2gpt(objApp *unstructured.Unstructured) (*generated.Gpt, error) {
	app := &v1alpha1.Application{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(objApp.UnstructuredContent(), app); err != nil {
		return nil, err
	}
	return app2gpt(app)
}

func getHot(app *v1alpha1.Application) (res int) {
	for _, v := range chat.Conversations {
		if v.AppNamespce == app.Namespace && v.AppName == app.Name {
			res += len(v.Messages)
		}
	}
	return res
}

func GetGPT(ctx context.Context, c dynamic.Interface, name string) (*generated.Gpt, error) {
	namespace, name, found := strings.Cut(name, "/")
	if !found {
		// TODO how to return 404 or something? not 500
		return nil, fmt.Errorf("input arg name is not valid")
	}
	app := &v1alpha1.Application{}
	if err := getResource(ctx, c, common.SchemaOf(&common.ArcadiaAPIGroup, "Application"), namespace, name, app); err != nil {
		return nil, err
	}
	if !app.Spec.IsPublic {
		return nil, fmt.Errorf("not a valid app or the app is not public")
	}

	return app2gpt(app)
}

func ListGPT(ctx context.Context, c dynamic.Interface, input generated.ListGPTInput) (*generated.PaginatedResult, error) {
	keyword := pointer.StringDeref(input.Keyword, "")
	category := pointer.StringDeref(input.Category, "")
	page := pointer.IntDeref(input.Page, 1)
	pageSize := pointer.IntDeref(input.PageSize, 10)
	res, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Application")).Namespace("").List(ctx, metav1.ListOptions{
		LabelSelector: v1alpha1.AppPublicLabelKey,
	})
	if err != nil {
		return nil, err
	}
	filter := make([]common.ResourceFilter, 0)
	if keyword != "" {
		filter = append(filter, common.FilterApplicationByKeyword(keyword))
	}
	if category != "" {
		filter = append(filter, common.FilterApplicationByCategory(category))
	}
	return common.ListReources(res, page, pageSize, func(obj *unstructured.Unstructured) (generated.PageNode, error) {
		return unstructred2gpt(obj)
	}, filter...)
}

func getResource(ctx context.Context, c dynamic.Interface, resource schema.GroupVersionResource, namespace, name string, typedObj any) error {
	resourceInterface := c.Resource(resource).Namespace(namespace)
	obj, err := resourceInterface.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), typedObj)
}
