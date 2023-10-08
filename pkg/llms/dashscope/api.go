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
	QWEN14BChat Model = "qwen-14b-chat"
	QWEN7BChat  Model = "qwen-7b-chat"
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
