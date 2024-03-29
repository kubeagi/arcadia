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
	"sync"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/chat"
	"github.com/kubeagi/arcadia/apiserver/pkg/chat/storage"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	"github.com/kubeagi/arcadia/pkg/config"
)

var (
	once        sync.Once
	chatStorage storage.Storage
)

func app2gpt(app *v1alpha1.Application, c client.Client) (*generated.Gpt, error) {
	if app == nil {
		return nil, errors.New("no app found")
	}

	gpt := &generated.Gpt{
		Name:               pointer.String(strings.Join([]string{app.Namespace, app.Name}, "/")),
		DisplayName:        pointer.String(app.Spec.DisplayName),
		Description:        pointer.String(app.Spec.Description),
		Hot:                pointer.Int64(getHot(app, c)),
		Creator:            pointer.String(app.Spec.Creator),
		Category:           common.GetAppCategory(app),
		Icon:               pointer.String(app.Spec.Icon),
		Prologue:           pointer.String(app.Spec.Prologue),
		ShowRespInfo:       pointer.Bool(app.Spec.ShowRespInfo),
		ShowRetrievalInfo:  pointer.Bool(app.Spec.ShowRetrievalInfo),
		ShowNextGuide:      pointer.Bool(app.Spec.ShowNextGuide),
		EnableUploadFile:   app.Spec.EnableUploadFile,
		NotReadyReasonCode: pointer.String(string(GetGPTNotReadyReasonCode(app))),
	}
	return gpt, nil
}

type GPTNotReadyReasonCode string

const (
	GPTIsReady            GPTNotReadyReasonCode = ""
	VectorStoreIsNotReady GPTNotReadyReasonCode = "VectorStoreIsNotReady"
	EmbedderIsNotReady    GPTNotReadyReasonCode = "EmbedderIsNotReady"
	LLMNotReady           GPTNotReadyReasonCode = "LLMNotReady"
	KnowledgeBaseNotReady GPTNotReadyReasonCode = "KnowledgeBaseNotReady"
	ConfigError           GPTNotReadyReasonCode = "ConfigError"
)

func GetGPTNotReadyReasonCode(application *v1alpha1.Application) GPTNotReadyReasonCode {
	isReady, msg := application.Status.IsReadyOrGetReadyMessage()
	if isReady {
		return GPTIsReady
	}
	_, conditionMessage, find := strings.Cut(msg, "[message]: ")
	if !find {
		return ConfigError
	}
	groupKind, detail, find := strings.Cut(conditionMessage, " || ")
	if !find {
		return ConfigError
	}
	group, kind, find := strings.Cut(groupKind, ":")
	if !find {
		return ConfigError
	}
	switch group {
	case "":
		switch kind {
		case "llm":
			return LLMNotReady
		case "knowledgebase":
			if strings.Contains(detail, "vectorstore") {
				return VectorStoreIsNotReady
			}
			if strings.Contains(detail, "embedder") {
				return EmbedderIsNotReady
			}
			return KnowledgeBaseNotReady
		default:
			return ConfigError
		}
	default:
		return ConfigError
	}
}

func getHot(app *v1alpha1.Application, cli client.Client) int64 {
	if chatStorage == nil {
		once.Do(
			func() {
				chatStorage = chat.NewChatServer(cli).Storage()
			})
	}
	if chatStorage == nil {
		return 0
	}
	res, err := chatStorage.CountMessages(app.Name, app.Namespace)
	if err != nil {
		return 0
	}
	return res
}

func GetGPT(ctx context.Context, c client.Client, name string) (*generated.Gpt, error) {
	namespace, name, found := strings.Cut(name, "/")
	if !found {
		// TODO how to return 404 or something? not 500
		return nil, fmt.Errorf("input arg name is not valid")
	}
	app := &v1alpha1.Application{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, app); err != nil {
		return nil, err
	}
	if !app.Spec.IsPublic {
		return nil, fmt.Errorf("not a valid app or the app is not public")
	}

	return app2gpt(app, c)
}

// ListGPT list all gpt
func ListGPT(ctx context.Context, c client.Client, input generated.ListGPTInput) (*generated.PaginatedResult, error) {
	keyword := pointer.StringDeref(input.Keyword, "")
	category := pointer.StringDeref(input.Category, "")
	page := pointer.IntDeref(input.Page, 1)
	pageSize := pointer.IntDeref(input.PageSize, 10)
	res := &v1alpha1.ApplicationList{}
	l := labels.Set{v1alpha1.AppPublicLabelKey: ""}
	if err := c.List(ctx, res, &client.ListOptions{LabelSelector: l.AsSelector(), Namespace: ""}); err != nil {
		return nil, err
	}
	filter := make([]common.ResourceFilter, 0)
	if keyword != "" {
		filter = append(filter, common.FilterApplicationByKeyword(keyword))
	}
	if category != "" {
		filter = append(filter, common.FilterApplicationByCategory(category))
	}
	items := make([]client.Object, len(res.Items))
	for i := range res.Items {
		items[i] = &res.Items[i]
	}
	return common.ListReources(items, page, pageSize, func(obj client.Object) (generated.PageNode, error) {
		app, ok := obj.(*v1alpha1.Application)
		if !ok {
			return nil, errors.New("can't convert obj to Application")
		}
		return app2gpt(app, c)
	}, filter...)
}

// ListGPTCategory list all categories
func ListGPTCategory(ctx context.Context, c client.Client) ([]*generated.GPTCategory, error) {
	categories, err := config.GetGPTsCategories(ctx, c)
	if err != nil {
		return nil, err
	}
	resp := make([]*generated.GPTCategory, len(categories))
	for i := range categories {
		resp[i] = &generated.GPTCategory{
			Name:   categories[i].Name,
			NameEn: categories[i].NameEn,
			ID:     categories[i].ID,
		}
	}
	return resp, nil
}

// GetGPTStore get gpt store info
func GetGPTStore(ctx context.Context, cli client.Client) (*generated.GPTStore, error) {
	cfg, err := common.GetGPTStoreConfig(ctx, cli)
	if err != nil {
		return nil, err
	}
	return &generated.GPTStore{
		URL:             cfg.URL,
		PublicNamespace: cfg.PublicNamespace,
	}, nil
}
