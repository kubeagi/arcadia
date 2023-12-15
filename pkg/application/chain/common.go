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

package chain

import (
	"context"
	"errors"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
	langchaingoschema "github.com/tmc/langchaingo/schema"
	"k8s.io/klog/v2"

	"github.com/kubeagi/arcadia/api/app-node/chain/v1alpha1"
)

func stream(res map[string]any) func(ctx context.Context, chunk []byte) error {
	return func(ctx context.Context, chunk []byte) error {
		if _, ok := res["_answer_stream"]; !ok {
			res["_answer_stream"] = make(chan string)
		}
		streamChan, ok := res["_answer_stream"].(chan string)
		if !ok {
			klog.Errorln("answer_stream is not chan string")
			return errors.New("answer_stream is not chan string")
		}
		klog.V(5).Infoln("stream out:", string(chunk))
		streamChan <- string(chunk)
		return nil
	}
}

func getChainOptions(config v1alpha1.CommonChainConfig) []chains.ChainCallOption {
	options := make([]chains.ChainCallOption, 0)
	if config.MaxTokens > 0 {
		options = append(options, chains.WithMaxTokens(config.MaxTokens))
	}
	if config.Temperature > 0 {
		options = append(options, chains.WithTemperature(config.Temperature))
	}
	if len(config.StopWords) > 0 {
		options = append(options, chains.WithStopWords(config.StopWords))
	}
	if config.TopK > 0 {
		options = append(options, chains.WithTopK(config.TopK))
	}
	if config.TopP > 0 {
		options = append(options, chains.WithTopP(config.TopP))
	}
	if config.Seed > 0 {
		options = append(options, chains.WithSeed(config.Seed))
	}
	if config.MinLength > 0 {
		options = append(options, chains.WithMinLength(config.MinLength))
	}
	if config.MaxLength > 0 {
		options = append(options, chains.WithMaxLength(config.MaxLength))
	}
	if config.RepetitionPenalty > 0 {
		options = append(options, chains.WithRepetitionPenalty(config.RepetitionPenalty))
	}
	if config.Model != "" {
		options = append(options, chains.WithModel(config.Model))
	}
	return options
}

func getMemory(llm llms.LanguageModel, config v1alpha1.Memory, history langchaingoschema.ChatMessageHistory) langchaingoschema.Memory {
	if config.MaxTokenLimit > 0 {
		return memory.NewConversationTokenBuffer(llm, config.MaxTokenLimit, memory.WithInputKey("question"), memory.WithOutputKey("text"), memory.WithChatHistory(history))
	}
	if config.ConversionWindowSize > 0 {
		return memory.NewConversationWindowBuffer(config.ConversionWindowSize, memory.WithInputKey("question"), memory.WithOutputKey("text"), memory.WithChatHistory(history))
	}
	return memory.NewSimple()
}
