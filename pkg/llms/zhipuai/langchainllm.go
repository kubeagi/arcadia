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
	"math/rand"
	"reflect"
	"time"

	"github.com/r3labs/sse/v2"
	"github.com/tmc/langchaingo/callbacks"
	langchainllm "github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/schema"
	"k8s.io/klog/v2"
)

var (
	ErrEmptyResponse = errors.New("no response")
	ErrEmptyPrompt   = errors.New("empty prompt")
)

var (
	_ langchainllm.Model = (*ZhiPuAILLM)(nil)
)

type options struct {
	retryTimes       int
	callbacksHandler callbacks.Handler
}

type Option func(*options)

func WithRetryTimes(retryTimes int) Option {
	return func(o *options) {
		o.retryTimes = retryTimes
	}
}

func WithCallback(callbacksHandler callbacks.Handler) Option {
	return func(o *options) {
		o.callbacksHandler = callbacksHandler
	}
}

type ZhiPuAILLM struct {
	c       *ZhiPuAI
	options *options
}

func NewZhiPuAILLM(apiKey string, opts ...Option) *ZhiPuAILLM {
	z := &ZhiPuAILLM{
		c: NewZhiPuAI(apiKey),
		options: &options{
			// 2 times by default
			retryTimes: 2,
		},
	}
	for _, opt := range opts {
		opt(z.options)
	}
	return z
}

func (z *ZhiPuAILLM) GetNumTokens(text string) int {
	return langchainllm.CountTokens("gpt2", text)
}

func (z *ZhiPuAILLM) Call(ctx context.Context, prompt string, options ...langchainllm.CallOption) (string, error) {
	return langchainllm.GenerateFromSinglePrompt(ctx, z, prompt, options...)
}

func (z *ZhiPuAILLM) GenerateContent(ctx context.Context, messages []langchainllm.MessageContent, options ...langchainllm.CallOption) (*langchainllm.ContentResponse, error) {
	if z.options.callbacksHandler != nil {
		z.options.callbacksHandler.HandleLLMGenerateContentStart(ctx, messages)
	}
	opts := langchainllm.CallOptions{}
	for _, opt := range options {
		opt(&opts)
	}
	chatMsgs := make([]*openai.ChatMessage, 0, len(messages))
	for _, mc := range messages {
		msg := &openai.ChatMessage{MultiContent: mc.Parts}
		switch mc.Role {
		case schema.ChatMessageTypeAI:
			msg.Role = string(Assistant)
		case schema.ChatMessageTypeHuman:
			msg.Role = string(User)
		case schema.ChatMessageTypeGeneric:
			msg.Role = string(User)
		case schema.ChatMessageTypeSystem:
			fallthrough
		case schema.ChatMessageTypeFunction:
			fallthrough
		default:
			klog.Infof("unsupported role: %s, just skip", mc.Role)
			continue
		}

		chatMsgs = append(chatMsgs, msg)
	}
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
	if len(messages) == 0 {
		return nil, ErrEmptyPrompt
	}
	for _, prompt := range chatMsgs {
		content := prompt.Content
		if content == "" {
			for i := range prompt.MultiContent {
				c, ok := prompt.MultiContent[i].(langchainllm.TextContent)
				if ok {
					content = c.Text
				}
			}
		}
		params.Prompt = append(params.Prompt, Prompt{Role: User, Content: content})
	}
	needStream := opts.StreamingFunc != nil
	if needStream {
		res := bytes.NewBuffer(nil)
		err := z.c.SSEInvoke(params, func(event *sse.Event) {
			if string(event.Event) == "finish" {
				return
			}
			_, _ = res.Write(event.Data)
			_ = opts.StreamingFunc(ctx, event.Data)
		})
		if err != nil {
			return nil, err
		}
		return &langchainllm.ContentResponse{Choices: []*langchainllm.ContentChoice{{Content: res.String()}}}, nil
	}
	var resp *Response
	var err error
	i := 0
	for {
		i++
		resp, err = z.c.Invoke(params)
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, ErrEmptyResponse
		}
		if resp.Data == nil {
			klog.Errorf("empty response: msg:%s code:%d\n", resp.Msg, resp.Code)
			if i <= z.options.retryTimes {
				r := rand.Intn(5)
				klog.Infof("retry[%d], sleep %d seconds, then recall...\n", i, r)
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
	choices := make([]*langchainllm.ContentChoice, 0, len(resp.Data.Choices))
	for _, c := range resp.Data.Choices {
		var s string
		if err := json.Unmarshal([]byte(c.Content), &s); err == nil {
			choices = append(choices, &langchainllm.ContentChoice{
				Content:        s,
				GenerationInfo: generationInfo,
			})
		}
	}

	response := &langchainllm.ContentResponse{Choices: choices}
	if z.options.callbacksHandler != nil {
		z.options.callbacksHandler.HandleLLMGenerateContentEnd(ctx, response)
	}
	return response, nil
}
