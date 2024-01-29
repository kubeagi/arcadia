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
	"fmt"
	"strings"
	"sync"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
	langchaingoschema "github.com/tmc/langchaingo/schema"
	"k8s.io/klog/v2"

	agent "github.com/kubeagi/arcadia/api/app-node/agent/v1alpha1"
	"github.com/kubeagi/arcadia/api/app-node/chain/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/appruntime/retriever"
	"github.com/kubeagi/arcadia/pkg/appruntime/tools/bingsearch"
)

func stream(res map[string]any) func(ctx context.Context, chunk []byte) error {
	return func(ctx context.Context, chunk []byte) error {
		logger := klog.FromContext(ctx)
		if _, ok := res["_answer_stream"]; !ok {
			logger.Info("no _answer_stream found, create a new one")
			res["_answer_stream"] = make(chan string)
		}
		streamChan, ok := res["_answer_stream"].(chan string)
		if !ok {
			err := fmt.Errorf("answer_stream is not chan string, but %T", res["_answer_stream"])
			logger.Error(err, "answer_stream is not chan string")
			return err
		}
		logger.V(5).Info("stream out:" + string(chunk))
		streamChan <- string(chunk)
		return nil
	}
}

func GetChainOptions(config v1alpha1.CommonChainConfig) []chains.ChainCallOption {
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
	if len(config.Model) != 0 {
		options = append(options, chains.WithModel(config.Model))
	}
	return options
}

func getMemory(llm llms.LLM, config v1alpha1.Memory, history langchaingoschema.ChatMessageHistory, inputKey, outputKey string) langchaingoschema.Memory {
	if inputKey == "" {
		inputKey = "question"
	}
	if outputKey == "" {
		outputKey = "text"
	}
	if config.MaxTokenLimit > 0 {
		return memory.NewConversationTokenBuffer(llm, config.MaxTokenLimit, memory.WithInputKey(inputKey), memory.WithOutputKey(outputKey), memory.WithChatHistory(history))
	}
	if config.ConversionWindowSize > 0 {
		return memory.NewConversationWindowBuffer(config.ConversionWindowSize, memory.WithInputKey(inputKey), memory.WithOutputKey(outputKey), memory.WithChatHistory(history))
	}
	return memory.NewSimple()
}

func runTools(ctx context.Context, args map[string]any, tools []agent.Tool) map[string]any {
	if len(tools) == 0 {
		return args
	}
	input, ok := args["question"].(string)
	if !ok {
		return args
	}
	result := make([]string, len(tools))
	resultRef := make([][]retriever.Reference, len(tools))
	for i := range resultRef {
		resultRef[i] = make([]retriever.Reference, 0)
	}
	var wg sync.WaitGroup
	wg.Add(len(tools))
	for i, tool := range tools {
		i, tool := i, tool
		go func(int, agent.Tool) {
			defer wg.Done()
			switch tool.Name { // nolint:gocritic
			case "bing":
				klog.V(3).Infof("tools call bing search: %s", input)
				client, err := bingsearch.NewFromToolSpec(&tool)
				if err != nil {
					klog.Errorf("failed to create bing client: %w", err)
					return
				}
				data, _, err := client.SearchGetDetailData(ctx, input)
				if err != nil {
					klog.Errorf("failed to call bing search tool: %w", err)
					return
				}
				ref := make([]retriever.Reference, len(data))
				for j := range data {
					ref[j] = retriever.Reference{
						Title:   data[j].Title,
						Content: data[j].Description,
						URL:     data[j].URL,
					}
				}
				resultRef[i] = ref
				result[i] = bingsearch.FormatResults(data)
				klog.V(3).Infof("tools call bing search done: %s", input)
			}
		}(i, tool)
	}
	wg.Wait()
	res := make([]string, 0, len(result))
	for i := range result {
		if s := strings.TrimSpace(result[i]); s != "" {
			res = append(res, s)
		}
	}
	toolOut := strings.Join(res, "\n")
	old, exist := args["context"]
	if exist {
		toolOut = old.(string) + "\n" + toolOut
	}
	args["context"] = toolOut
	for i := range resultRef {
		args = retriever.AddReferencesToArgs(args, resultRef[i])
	}
	return args
}
