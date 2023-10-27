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
package dashscope

import (
	"context"
	"errors"

	"github.com/tmc/langchaingo/embeddings"

	"github.com/kubeagi/arcadia/pkg/llms/dashscope"
)

var _ embeddings.Embedder = (*DashScopeEmbedder)(nil)

type DashScopeEmbedder struct {
	*dashscope.DashScope
}

const (
	MaxTextLength = 25 // https://help.aliyun.com/zh/dashscope/developer-reference/text-embedding-quick-start
)

func NewDashScopeEmbedder(apiKey string) *DashScopeEmbedder {
	return &DashScopeEmbedder{
		DashScope: dashscope.NewDashScope(apiKey, false),
	}
}
func (d DashScopeEmbedder) EmbedDocuments(ctx context.Context, texts []string) (res [][]float32, err error) {
	res = make([][]float32, 0, len(texts))
	for i := 0; i < len(texts); i += MaxTextLength {
		end := i + MaxTextLength
		if end > len(texts) {
			end = len(texts)
		}
		data := texts[i:end]
		embedding, err := d.CreateEmbedding(ctx, data, false)
		if err != nil {
			return res, err
		}
		if len(embedding) == 0 {
			return res, errors.New("embedding is empty")
		}
		for j := 0; j < len(embedding); j++ {
			res = append(res, embedding[j].Embedding)
		}
	}
	return res, nil
}

func (d DashScopeEmbedder) EmbedQuery(ctx context.Context, text string) (res []float32, err error) {
	embedding, err := d.CreateEmbedding(ctx, []string{text}, true)
	if err != nil {
		return nil, err
	}
	if len(embedding) != 1 {
		return nil, errors.New("embedding is empty")
	}
	res = embedding[0].Embedding
	return res, nil
}
