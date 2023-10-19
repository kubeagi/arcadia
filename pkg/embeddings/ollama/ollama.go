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

package ollama

import (
	"context"

	llmzhipuai "github.com/kubeagi/arcadia/pkg/llms/zhipuai"
	"github.com/tmc/langchaingo/embeddings"
)

// To be compatible with tmc/langchaingo embeddings(https://github.com/tmc/langchaingo/tree/main/embeddings)
var _ embeddings.Embedder = Ollama{}

type Ollama struct {
	client *llmzhipuai.ZhiPuAI

	StripNewLines bool
	BatchSize     int
}

func (ollama Ollama) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, nil
}

func (ollama Ollama) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return nil, nil
}
