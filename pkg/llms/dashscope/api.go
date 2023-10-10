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

	"github.com/kubeagi/arcadia/pkg/llms"
)

const (
	DashScopeChatURL = "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation"
)

type Model string

const (
	// 通义千问对外开源的 14B / 7B 规模参数量的经过人类指令对齐的 chat 模型
	QWEN14BChat Model = "qwen-14b-chat"
	QWEN7BChat  Model = "qwen-7b-chat"
	// LLaMa2 系列大语言模型由 Meta 开发并公开发布，其规模从 70 亿到 700 亿参数不等。在灵积上提供的 llama2-7b-chat-v2 和 llama2-13b-chat-v2，分别为 7B 和 13B 规模的 LLaMa2 模型，针对对话场景微调优化后的版本。
	LLAMA27BCHATV2  Model = "llama2-7b-chat-v2"
	LLAMA213BCHATV2 Model = "llama2-13b-chat-v2"
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
	return do(context.TODO(), DashScopeChatURL, z.apiKey, data, z.sse)
}

func (z *DashScope) Validate() (llms.Response, error) {
	return nil, errors.New("not implemented")
}
