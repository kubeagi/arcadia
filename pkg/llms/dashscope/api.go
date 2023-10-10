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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/kubeagi/arcadia/pkg/llms"
)

const (
	DashScopeChatURL          = "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation"
	DashScopeTextEmbeddingURL = "https://dashscope.aliyuncs.com/api/v1/services/embeddings/text-embedding/text-embedding"
	DashScopeTaskURL          = "https://dashscope.aliyuncs.com/api/v1/tasks/"
)

type Model string

const (
	// 通义千问对外开源的 14B / 7B 规模参数量的经过人类指令对齐的 chat 模型
	QWEN14BChat Model = "qwen-14b-chat"
	QWEN7BChat  Model = "qwen-7b-chat"
	// LLaMa2 系列大语言模型由 Meta 开发并公开发布，其规模从 70 亿到 700 亿参数不等。在灵积上提供的 llama2-7b-chat-v2 和 llama2-13b-chat-v2，分别为 7B 和 13B 规模的 LLaMa2 模型，针对对话场景微调优化后的版本。
	LLAMA27BCHATV2   Model = "llama2-7b-chat-v2"
	LLAMA213BCHATV2  Model = "llama2-13b-chat-v2"
	EmbeddingV1      Model = "text-embedding-v1"       // 通用文本向量 同步调用
	EmbeddingAsyncV1 Model = "text-embedding-async-v1" // 通用文本向量 批处理调用
)

var _ llms.LLM = (*DashScope)(nil)

type DashScope struct {
	apiKey string
	sse    bool
}

func NewDashScope(apiKey string, sse bool) *DashScope {
	return &DashScope{
		apiKey: apiKey,
		sse:    sse,
	}
}

func (z DashScope) Type() llms.LLMType {
	return llms.DashScope
}

// Call wraps a common AI api call
func (z *DashScope) Call(data []byte) (llms.Response, error) {
	params := ModelParams{}
	if err := params.Unmarshal(data); err != nil {
		return nil, err
	}
	return do(context.TODO(), DashScopeChatURL, z.apiKey, data, z.sse, false)
}

func (z *DashScope) Validate() (llms.Response, error) {
	return nil, errors.New("not implemented")
}

func (z *DashScope) CreateEmbedding(ctx context.Context, inputTexts []string, query bool) ([]Embeddings, error) {
	textType := TextTypeDocument
	if query {
		textType = TextTypeQuery
	}
	reqBody := EmbeddingRequest{
		Model: EmbeddingV1,
		Input: EmbeddingInput{
			EmbeddingInputSync: &EmbeddingInputSync{
				Texts: inputTexts,
			},
		},
		Parameters: EmbeddingParameters{
			TextType: textType,
		},
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	resp, err := req(ctx, DashScopeTextEmbeddingURL, z.apiKey, data, false, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respData := &EmbeddingResponse{}
	if err := json.NewDecoder(resp.Body).Decode(respData); err != nil {
		return nil, err
	}
	if respData.StatusCode != 200 && respData.StatusCode != 0 {
		return nil, errors.New(respData.Message)
	}
	return respData.Output.Embeddings, nil
}

func (z *DashScope) CreateEmbeddingAsync(ctx context.Context, inputURL string, query bool) (taskID string, err error) {
	textType := TextTypeDocument
	if query {
		textType = TextTypeQuery
	}
	reqBody := EmbeddingRequest{
		Model: EmbeddingAsyncV1,
		Input: EmbeddingInput{
			EmbeddingInputAsync: &EmbeddingInputAsync{
				URL: inputURL,
			},
		},
		Parameters: EmbeddingParameters{
			TextType: textType,
		},
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	resp, err := req(ctx, DashScopeTextEmbeddingURL, z.apiKey, data, false, true)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respData := &EmbeddingResponse{}
	if err := json.NewDecoder(resp.Body).Decode(respData); err != nil {
		return "", err
	}
	if respData.StatusCode != 200 && respData.StatusCode != 0 {
		return "", errors.New(respData.Message)
	}
	return respData.Output.TaskID, nil
}

func (z *DashScope) GetTaskDetail(ctx context.Context, taskID string) (outURL string, err error) {
	resp, err := req(ctx, DashScopeTaskURL+taskID, z.apiKey, nil, false, false)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respData := &EmbeddingResponse{}
	if err := json.NewDecoder(resp.Body).Decode(respData); err != nil {
		return "", err
	}
	if respData.StatusCode != 200 && respData.StatusCode != 0 {
		return "", errors.New(respData.Message)
	}
	data := respData.Output.EmbeddingOutputASync
	if data == nil {
		return "", fmt.Errorf("cant find data in resp:%+v", respData)
	}
	if data.TaskStatus != TaskStatusSucceeded {
		return "", fmt.Errorf("taskStatus:%s, message:%s", data.TaskStatus, data.Message)
	}
	if data.URL != "" {
		return data.URL, nil
	}
	return "", errors.New(respData.Message)
}
