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

package zhipuai

import (
	"context"
	"strings"

	"github.com/tmc/langchaingo/embeddings"

	llmzhipuai "github.com/kubeagi/arcadia/pkg/llms/zhipuai"
)

// To be compatible with tmc/langchaingo embeddings(https://github.com/tmc/langchaingo/tree/main/embeddings)
var _ embeddings.Embedder = ZhiPuAI{}

type ZhiPuAI struct {
	client *llmzhipuai.ZhiPuAI

	StripNewLines bool
	BatchSize     int
}

func NewZhiPuAI(opts ...Option) (ZhiPuAI, error) {
	o, err := applyClientOptions(opts...)
	if err != nil {
		return ZhiPuAI{}, err
	}

	return o, nil
}

// EmbedDocuments creates one vector embedding for each of the texts.
func (e ZhiPuAI) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	batchedTexts := embeddings.BatchTexts(
		embeddings.MaybeRemoveNewLines(texts, e.StripNewLines),
		e.BatchSize,
	)

	emb := make([][]float32, 0, len(texts))
	for _, texts := range batchedTexts {
		curTextEmbeddings, err := e.client.CreateEmbedding(ctx, texts)
		if err != nil {
			return nil, err
		}

		textLengths := make([]int, 0, len(texts))
		for _, text := range texts {
			textLengths = append(textLengths, len(text))
		}

		combined, err := embeddings.CombineVectors(curTextEmbeddings, textLengths)
		if err != nil {
			return nil, err
		}

		emb = append(emb, combined)
	}

	return emb, nil
}

// EmbedQuery embeds a single text.
func (e ZhiPuAI) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if e.StripNewLines {
		text = strings.ReplaceAll(text, "\n", " ")
	}

	emb, err := e.client.CreateEmbedding(ctx, []string{text})
	if err != nil {
		return nil, err
	}

	return emb[0], nil
}
