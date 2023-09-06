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
package chromadb

import (
	"context"

	"github.com/tmc/langchaingo/embeddings"
)

type wrappedEmbeddingFunction struct {
	embeddings.Embedder
}

func (embedder wrappedEmbeddingFunction) CreateEmbedding(documents []string) ([][]float32, error) {
	vectors, err := embedder.EmbedDocuments(context.TODO(), documents)
	if err != nil {
		return nil, err
	}
	target := make([][]float32, len(vectors))
	for row := 0; row < len(vectors); row++ {
		target[row] = make([]float32, len(vectors[row]))
		for col := 0; col < len(vectors[row]); col++ {
			target[row][col] = float32(vectors[row][col])
		}
	}
	return target, nil
}

func (embedder wrappedEmbeddingFunction) CreateEmbeddingWithModel(documents []string, model string) ([][]float32, error) {
	return embedder.CreateEmbedding(documents)
}
