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

package log

import (
	"context"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
	"k8s.io/klog/v2"
)

// KLogHandler is a callback handler that prints to klog v3
type KLogHandler struct {
	LogLevel int
}

var _ callbacks.Handler = KLogHandler{}

func (l KLogHandler) HandleLLMGenerateContentStart(ctx context.Context, ms []llms.MessageContent) {
	logger := klog.FromContext(ctx)
	logger.V(l.LogLevel).Info("Entering LLM with messages:")
	for _, m := range ms {
		// TODO: Implement logging of other content types
		var buf strings.Builder
		for _, t := range m.Parts {
			if t, ok := t.(llms.TextContent); ok {
				buf.WriteString(t.Text)
			}
		}
		logger.V(l.LogLevel).Info("Role:", m.Role)
		logger.V(l.LogLevel).Info("Text:", buf.String())
	}
}

func (l KLogHandler) HandleLLMGenerateContentEnd(ctx context.Context, res *llms.ContentResponse) {
	logger := klog.FromContext(ctx)
	logger.V(l.LogLevel).Info("Exiting LLM with response:")
	for _, c := range res.Choices {
		if c.Content != "" {
			logger.V(l.LogLevel).Info("Content:", c.Content)
		}
		if c.StopReason != "" {
			logger.V(l.LogLevel).Info("StopReason:", c.StopReason)
		}
		if len(c.GenerationInfo) > 0 {
			logger.V(l.LogLevel).Info("GenerationInfo:")
			for k, v := range c.GenerationInfo {
				fmt.Printf("%20s: %v\n", k, v)
			}
		}
		// if c.FuncCall != nil {
		//	logger.V(l.LogLevel).Info("FuncCall: ", c.FuncCall.Name, c.FuncCall.Arguments)
		//}
	}
}

func (l KLogHandler) HandleStreamingFunc(ctx context.Context, chunk []byte) {
	logger := klog.FromContext(ctx)
	logger.V(l.LogLevel).Info(string(chunk))
}

func (l KLogHandler) HandleText(ctx context.Context, text string) {
	logger := klog.FromContext(ctx)
	logger.V(l.LogLevel).Info(text)
}

func (l KLogHandler) HandleLLMStart(ctx context.Context, prompts []string) {
	logger := klog.FromContext(ctx)
	logger.V(l.LogLevel).Info("Entering LLM with prompts:", prompts)
}

func (l KLogHandler) HandleLLMEnd(ctx context.Context, output llms.LLMResult) {
	logger := klog.FromContext(ctx)
	logger.V(l.LogLevel).Info("Exiting LLM with results:", formatLLMResult(output))
}

func (l KLogHandler) HandleLLMError(ctx context.Context, err error) {
	logger := klog.FromContext(ctx)
	logger.V(l.LogLevel).Info("Exiting LLM with error:", err)
}

func (l KLogHandler) HandleChainStart(ctx context.Context, inputs map[string]any) {
	logger := klog.FromContext(ctx)
	logger.V(l.LogLevel).Info("Entering chain with inputs:", formatChainValues(inputs))
}

func (l KLogHandler) HandleChainEnd(ctx context.Context, outputs map[string]any) {
	logger := klog.FromContext(ctx)
	logger.V(l.LogLevel).Info("Exiting chain with outputs:", formatChainValues(outputs))
}

func (l KLogHandler) HandleChainError(ctx context.Context, err error) {
	logger := klog.FromContext(ctx)
	logger.V(l.LogLevel).Info("Exiting chain with error:", err)
}

func (l KLogHandler) HandleToolStart(ctx context.Context, input string) {
	logger := klog.FromContext(ctx)
	logger.V(l.LogLevel).Info("Entering tool with input:", removeNewLines(input))
}

func (l KLogHandler) HandleToolEnd(ctx context.Context, output string) {
	logger := klog.FromContext(ctx)
	logger.V(l.LogLevel).Info("Exiting tool with output:", removeNewLines(output))
}

func (l KLogHandler) HandleToolError(ctx context.Context, err error) {
	logger := klog.FromContext(ctx)
	logger.V(l.LogLevel).Info("Exiting tool with error:", err)
}

func (l KLogHandler) HandleAgentAction(ctx context.Context, action schema.AgentAction) {
	logger := klog.FromContext(ctx)
	logger.V(l.LogLevel).Info("Agent selected action:", formatAgentAction(action))
}

func (l KLogHandler) HandleAgentFinish(ctx context.Context, finish schema.AgentFinish) {
	logger := klog.FromContext(ctx)
	logger.V(l.LogLevel).Info(fmt.Sprintf("Agent finish: %v", finish))
}

func (l KLogHandler) HandleRetrieverStart(ctx context.Context, query string) {
	logger := klog.FromContext(ctx)
	logger.V(l.LogLevel).Info("Entering retriever with query:", removeNewLines(query))
}

func (l KLogHandler) HandleRetrieverEnd(ctx context.Context, query string, documents []schema.Document) {
	logger := klog.FromContext(ctx)
	logger.V(l.LogLevel).Info(fmt.Sprintf("Exiting retriever with documents for query:%s documents: %v", query, documents))
}

func formatChainValues(values map[string]any) string {
	output := ""
	for key, value := range values {
		output += fmt.Sprintf("\"%s\" : \"%s\", ", removeNewLines(key), removeNewLines(value))
	}

	return output
}

func formatLLMResult(output llms.LLMResult) string {
	results := "[ "
	for i := 0; i < len(output.Generations); i++ {
		for j := 0; j < len(output.Generations[i]); j++ {
			results += output.Generations[i][j].Text
		}
	}

	return results + " ]"
}

func formatAgentAction(action schema.AgentAction) string {
	return fmt.Sprintf("\"%s\" with input \"%s\"", removeNewLines(action.Tool), removeNewLines(action.ToolInput))
}

func removeNewLines(s any) string {
	return strings.ReplaceAll(fmt.Sprint(s), "\n", " ")
}
