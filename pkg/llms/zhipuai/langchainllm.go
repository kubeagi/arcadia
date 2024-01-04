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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"time"

	"github.com/r3labs/sse/v2"
	langchainllm "github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
	"k8s.io/klog/v2"
)

var (
	ErrEmptyResponse = errors.New("no response")
	ErrEmptyPrompt   = errors.New("empty prompt")
)

var (
	_ langchainllm.LLM = (*ZhiPuAILLM)(nil)
)

type ZhiPuAILLM struct {
	ZhiPuAI
	RetryTimes int
}

func (z *ZhiPuAILLM) GetNumTokens(text string) int {
	return langchainllm.CountTokens("gpt2", text)
}

func (z *ZhiPuAILLM) Call(ctx context.Context, prompt string, options ...langchainllm.CallOption) (string, error) {
	r, err := z.Generate(ctx, []string{prompt}, options...)
	if err != nil {
		return "", fmt.Errorf("failed to generate: %w", err)
	}
	if len(r) == 0 {
		return "", ErrEmptyResponse
	}
	return r[0].Text, nil
}

func (z *ZhiPuAILLM) Generate(ctx context.Context, prompts []string, options ...langchainllm.CallOption) ([]*langchainllm.Generation, error) {
	opts := langchainllm.CallOptions{}
	for _, opt := range options {
		opt(&opts)
	}
	generations := make([]*langchainllm.Generation, 0, len(prompts))
	params := DefaultModelParams()
	if opts.TopP > 0 && opts.TopP < 1 {
		params.TopP = float32(opts.TopP)
	}
	if opts.Temperature > 0 && opts.Temperature < 1 {
		params.Temperature = float32(opts.Temperature)
	}
	if opts.Model != "" {
		params.Model = opts.Model
	}
	if len(prompts) == 0 {
		return nil, ErrEmptyPrompt
	}
	for _, prompt := range prompts {
		params.Prompt = append(params.Prompt, Prompt{Role: User, Content: prompt})
		klog.Infof("get prompts: %#v\n", params.Prompt)
		client := NewZhiPuAI(z.apiKey)
		needStream := opts.StreamingFunc != nil
		if needStream {
			res := bytes.NewBuffer(nil)
			err := client.SSEInvoke(params, func(event *sse.Event) {
				if string(event.Event) == "finish" {
					return
				}
				_, _ = res.Write(event.Data)
				_ = opts.StreamingFunc(ctx, event.Data)
			})
			if err != nil {
				return nil, err
			}
			return []*langchainllm.Generation{
				{
					Text: res.String(),
				},
			}, nil
		}
		var resp *Response
		var err error
		i := 0
		for {
			i++
			resp, err = client.Invoke(params)
			if err != nil {
				return nil, err
			}
			if resp == nil {
				return nil, ErrEmptyResponse
			}
			if resp.Data == nil {
				klog.Errorf("zhipullm get empty response: msg:%s code:%d\n", resp.Msg, resp.Code)
				if i <= z.RetryTimes && (resp.Code == CodeConcurrencyHigh || resp.Code == CodefrequencyHigh || resp.Code == CodeTimesHigh) {
					r := rand.Intn(5)
					klog.Infof("zhipullm triggers retry[%d], sleep %d seconds, then recall...\n", i, r)
					time.Sleep(time.Duration(r) * time.Second)
					continue
				}
				return nil, ErrEmptyResponse
			}
			if len(resp.Data.Choices) == 0 {
				return nil, ErrEmptyResponse
			}
			break
		}
		generationInfo := make(map[string]any, reflect.ValueOf(resp.Data.Usage).NumField())
		generationInfo["TotalTokens"] = resp.Data.Usage.TotalTokens
		var s string
		if err := json.Unmarshal([]byte(resp.Data.Choices[0].Content), &s); err != nil {
			return nil, err
		}
		msg := &schema.AIChatMessage{
			Content: strings.TrimSpace(s),
		}
		generations = append(generations, &langchainllm.Generation{
			Message:        msg,
			Text:           msg.Content,
			GenerationInfo: generationInfo,
		})
	}
	return generations, nil
}
