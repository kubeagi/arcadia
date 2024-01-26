/*
Copyright 2023 The KubeAGI Authors.

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

package langchainwrap

import (
	"context"
	"fmt"

	langchaingoembeddings "github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/config"
	"github.com/kubeagi/arcadia/pkg/embeddings"
	zhipuaiembeddings "github.com/kubeagi/arcadia/pkg/embeddings/zhipuai"
	"github.com/kubeagi/arcadia/pkg/llms/zhipuai"
	"github.com/kubeagi/arcadia/pkg/utils"
)

func GetLangchainEmbedder(ctx context.Context, e *v1alpha1.Embedder, c client.Client, cli dynamic.Interface, model string, opts ...langchaingoembeddings.Option) (em langchaingoembeddings.Embedder, err error) {
	if err := utils.ValidateClient(c, cli); err != nil {
		return nil, err
	}
	switch e.Spec.Provider.GetType() {
	case v1alpha1.ProviderType3rdParty:
		switch e.Spec.Type { // nolint: gocritic
		case embeddings.ZhiPuAI:
			apiKey, err := e.AuthAPIKey(ctx, c, cli)
			if err != nil {
				return nil, err
			}
			return zhipuaiembeddings.NewZhiPuAI(
				zhipuaiembeddings.WithClient(*zhipuai.NewZhiPuAI(apiKey)),
			)
		case embeddings.OpenAI:
			apiKey, err := e.AuthAPIKey(ctx, c, cli)
			if err != nil {
				return nil, err
			}

			// When apitype is OpenAI,there are two possible sources:
			// 1. From official OpenAI
			// 2. From kubeagi which provides OpenAI compatible apis
			// Both only provides 1 embedding model,so get the 1st one should be fine if the model is not specified.
			if model == "" {
				models := e.GetModelList()
				model = models[0]
			}

			llm, err := openai.New(openai.WithModel(model), openai.WithBaseURL(e.Get3rdPartyEmbedderBaseURL()), openai.WithToken(apiKey))
			if err != nil {
				return nil, err
			}
			return langchaingoembeddings.NewEmbedder(llm, opts...)
		}
	case v1alpha1.ProviderTypeWorker:
		gateway, err := config.GetGateway(ctx, c, cli)
		if err != nil {
			return nil, err
		}
		if gateway == nil {
			return nil, fmt.Errorf("global config gateway not found")
		}
		workerRef := e.Spec.Worker
		if workerRef == nil {
			return nil, fmt.Errorf("embedder.spec.worker not defined")
		}
		worker := &v1alpha1.Worker{}
		if c != nil {
			if err := c.Get(ctx, types.NamespacedName{Namespace: workerRef.GetNamespace(e.GetNamespace()), Name: workerRef.Name}, worker); err != nil {
				return nil, err
			}
		} else {
			obj, err := cli.Resource(schema.GroupVersionResource{Group: v1alpha1.Group, Version: v1alpha1.Version, Resource: "workers"}).
				Namespace(workerRef.GetNamespace(e.GetNamespace())).Get(ctx, workerRef.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), worker)
			if err != nil {
				return nil, err
			}
		}
		modelRef := worker.Spec.Model
		if modelRef == nil {
			return nil, fmt.Errorf("worker.spec.model not defined")
		}
		modelName := worker.MakeRegistrationModelName()
		llm, err := openai.New(openai.WithModel(modelName), openai.WithBaseURL(gateway.APIServer), openai.WithToken("fake"))
		if err != nil {
			return nil, err
		}
		return langchaingoembeddings.NewEmbedder(llm, opts...)
	}
	return nil, fmt.Errorf("unknown provider type")
}
