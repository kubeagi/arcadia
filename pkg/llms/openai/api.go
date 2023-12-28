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

package openai

import (
	"context"
	"errors"
	"fmt"
	"time"

	langchainllms "github.com/tmc/langchaingo/llms"
	langchainopenai "github.com/tmc/langchaingo/llms/openai"

	"github.com/kubeagi/arcadia/pkg/llms"
)

const (
	OpenaiModelAPIURL    = "https://api.openai.com/v1"
	OpenaiDefaultTimeout = 300 * time.Second
)

var _ llms.LLM = (*OpenAI)(nil)

type OpenAI struct {
	apiKey  string
	baseURL string
}

func NewOpenAI(apiKey string, baseURL string) (*OpenAI, error) {
	if baseURL == "" {
		baseURL = OpenaiModelAPIURL
	}

	if apiKey == "" {
		// TODO: maybe we should consider local pseudo-openAI LLM worker that doesn't require an apiKey?
		return nil, fmt.Errorf("auth is empty")
	}

	return &OpenAI{
		apiKey:  apiKey,
		baseURL: baseURL,
	}, nil
}

func (o OpenAI) Type() llms.LLMType {
	return llms.OpenAI
}

func (o *OpenAI) Call(data []byte) (llms.Response, error) {
	return nil, errors.New("not implemented yet")
}

// Validate OpenAI service
func (o *OpenAI) Validate(ctx context.Context, options ...langchainllms.CallOption) (llms.Response, error) {
	// validate against models
	llm, err := langchainopenai.New(
		langchainopenai.WithBaseURL(o.baseURL),
		langchainopenai.WithToken(o.apiKey),
	)
	if err != nil {
		return nil, fmt.Errorf("init openai client: %w", err)
	}

	resp, err := llm.Call(ctx, "Hello", options...)
	if err != nil {
		return nil, err
	}

	return &Response{
		Code:    200,
		Data:    resp,
		Msg:     "",
		Success: true,
	}, nil
}
