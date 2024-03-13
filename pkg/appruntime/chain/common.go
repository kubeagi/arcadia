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

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
	langchaingoschema "github.com/tmc/langchaingo/schema"
	"k8s.io/klog/v2"

	appnode "github.com/kubeagi/arcadia/api/app-node"
	"github.com/kubeagi/arcadia/api/app-node/chain/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/appruntime/base"
)

func stream(res map[string]any) func(ctx context.Context, chunk []byte) error {
	return func(ctx context.Context, chunk []byte) error {
		logger := klog.FromContext(ctx)
		if _, ok := res[base.OutputAnserStreamChanKeyInArg]; !ok {
			logger.Info("no _answer_stream found, create a new one")
			res[base.OutputAnserStreamChanKeyInArg] = make(chan string)
		}
		streamChan, ok := res[base.OutputAnserStreamChanKeyInArg].(chan string)
		if !ok {
			err := fmt.Errorf("answer_stream is not chan string, but %T", res[base.OutputAnserStreamChanKeyInArg])
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

func GetMemory(llm llms.Model, config appnode.Memory, history langchaingoschema.ChatMessageHistory, inputKey, outputKey string) langchaingoschema.Memory {
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

/*
When using **stream** mode and an error occurs, **fastchat** returns http code 200 and returns an error message in json,
but this json is inconsistent with the default format of openai, so it is silently **ignored** by langchaingo,
so in this case, we need to re-request the blocking mode to find the correct error message.

fastchat will return like this:

data: {"id": "chatcmpl-3Lo28HviQo8949WkJyfFzJ", "model": "7597c2a3-b186-4bac-b2e4-21ff56413f7f", "choices": [{"index": 0, "delta": {"role": "assistant"}, "finish_reason": null}]}
data: {"text": "**NETWORK ERROR DUE TO HIGH TRAFFIC. PLEASE REGENERATE OR REFRESH THIS PAGE.**\n\n(\"addmm_impl_cpu_\" not implemented for 'Half')", "error_code": 50001}
data: [DONE]
*/
func handleNoErrNoOut(ctx context.Context, needStream bool, oldOut string, oldErr error, chain chains.Chain, args map[string]any, options []chains.ChainCallOption) (out string, err error) {
	if !needStream {
		return oldOut, oldErr
	}
	if len(strings.TrimSpace(oldOut)) == 0 && oldErr == nil {
		// Only the stream mode will encounter this problem. When in stream mode and the option length is 1, the option is the configuration parameter of the stream mode, and we can directly ignore it.
		if len(options) <= 1 {
			out, err = chains.Predict(ctx, chain, args)
		} else {
			out, err = chains.Predict(ctx, chain, args, options[:len(options)-1]...)
		}
		return out, err
	}
	return oldOut, oldErr
}
