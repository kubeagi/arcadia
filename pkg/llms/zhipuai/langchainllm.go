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
	"encoding/json"
	"errors"
	"strings"

	lanchainllm "github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
	"k8s.io/klog/v2"
)

var (
	ErrEmptyResponse       = errors.New("no response")
	ErrEmptyPrompt         = errors.New("empty prompt")
	ErrIncompleteEmbedding = errors.New("no all input got emmbedded")
)

type ZhiPuAILLM struct {
	ZhiPuAI
}

func (z ZhiPuAILLM) Call(ctx context.Context, prompt string, options ...lanchainllm.CallOption) (string, error) {
	r, err := z.Generate(ctx, []string{prompt}, options...)
	if err != nil {
		return "", err
	}
	if len(r) == 0 {
		return "", ErrEmptyResponse
	}
	return r[0].Text, nil
}

func (z ZhiPuAILLM) Generate(ctx context.Context, prompts []string, options ...lanchainllm.CallOption) ([]*lanchainllm.Generation, error) {
	params := DefaultModelParams()
	if len(prompts) == 0 {
		return nil, ErrEmptyPrompt
	}
	params.Prompt = []Prompt{
		{Role: User, Content: prompts[0]},
	}
	klog.Infoln("prompt:", prompts[0])
	client := NewZhiPuAI(z.apiKey)
	resp, err := client.Invoke(params)
	if err != nil {
		return nil, err
	}
	var s string
	klog.Infoln("resp:", resp.String())
	if err := json.Unmarshal([]byte(resp.Data.Choices[0].Content), &s); err != nil {
		return nil, err
	}
	return []*lanchainllm.Generation{
		{
			Text: strings.TrimSpace(s),
		},
	}, nil
}

func (z ZhiPuAILLM) GeneratePrompt(ctx context.Context, promptValues []schema.PromptValue, options ...lanchainllm.CallOption) (lanchainllm.LLMResult, error) {
	return lanchainllm.GeneratePrompt(ctx, z, promptValues, options...)
}

func (z ZhiPuAILLM) GetNumTokens(text string) int {
	// TODO implement me
	panic("implement me")
}
