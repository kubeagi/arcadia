/*
Copyright 2024 KubeAGI.

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
	buf := strings.Builder{}
	buf.WriteString("Entering LLM with messages: ")
	for _, m := range ms {
		// TODO: Implement logging of other content types
		buf.WriteString("text: ")
		for _, t := range m.Parts {
			if t, ok := t.(llms.TextContent); ok {
				buf.WriteString(t.Text)
			}
		}
		buf.WriteString("Role: ")
		buf.WriteString(string(m.Role))
	}
	logger.WithValues("logger", "arcadia")
	logger.V(l.LogLevel).Info(buf.String())
}

func (l KLogHandler) HandleLLMGenerateContentEnd(ctx context.Context, res *llms.ContentResponse) {
	logger := klog.FromContext(ctx)
	buf := strings.Builder{}
	buf.WriteString("Exiting LLM with response: ")
	for _, c := range res.Choices {
		if c.Content != "" {
			buf.WriteString("Content: " + c.Content)
		}
		if c.StopReason != "" {
			buf.WriteString("StopReason: " + c.StopReason)
		}
		if len(c.GenerationInfo) > 0 {
			buf.WriteString("GenerationInfo: ")
			for k, v := range c.GenerationInfo {
				buf.WriteString(fmt.Sprintf("%20s: %#v\n", k, v))
			}
		}
		if c.FuncCall != nil {
			buf.WriteString("FuncCall: " + c.FuncCall.Name + " " + c.FuncCall.Arguments)
		}
	}
	logger.WithValues("logger", "arcadia")
	logger.V(l.LogLevel).Info(buf.String())
}

func (l KLogHandler) HandleStreamingFunc(ctx context.Context, chunk []byte) {
	logger := klog.FromContext(ctx)
	logger.WithValues("logger", "arcadia")
	logger.V(l.LogLevel).Info("log streaming: " + string(chunk))
}

func (l KLogHandler) HandleText(ctx context.Context, text string) {
	logger := klog.FromContext(ctx)
	logger.WithValues("logger", "arcadia")
	logger.V(l.LogLevel).Info("log text: " + text)
}

func (l KLogHandler) HandleLLMStart(ctx context.Context, prompts []string) {
	logger := klog.FromContext(ctx)
	buf := strings.Builder{}
	buf.WriteString("Entering LLM with prompts: ")
	for _, p := range prompts {
		buf.WriteString(p)
		buf.WriteString(" ")
	}
	logger.WithValues("logger", "arcadia")
	logger.V(l.LogLevel).Info(buf.String())
}

func (l KLogHandler) HandleLLMError(ctx context.Context, err error) {
	logger := klog.FromContext(ctx)
	logger.WithValues("logger", "arcadia")
	logger.V(l.LogLevel).Error(err, "Exiting LLM with error")
}

func (l KLogHandler) HandleChainStart(ctx context.Context, inputs map[string]any) {
	logger := klog.FromContext(ctx)
	logger.WithValues("logger", "arcadia")
	logger.V(l.LogLevel).Info(fmt.Sprintf("Entering chain with inputs: %#v", inputs))
}

func (l KLogHandler) HandleChainEnd(ctx context.Context, outputs map[string]any) {
	logger := klog.FromContext(ctx)
	logger.WithValues("logger", "arcadia")
	logger.V(l.LogLevel).Info(fmt.Sprintf("Exiting chain with outputs: %#v", outputs))
}

func (l KLogHandler) HandleChainError(ctx context.Context, err error) {
	logger := klog.FromContext(ctx)
	logger.WithValues("logger", "arcadia")
	logger.V(l.LogLevel).Error(err, "Exiting chain with error")
}

func (l KLogHandler) HandleToolStart(ctx context.Context, input string) {
	logger := klog.FromContext(ctx)
	logger.WithValues("logger", "arcadia")
	logger.V(l.LogLevel).Info("Entering tool with input: " + removeNewLines(input))
}

func (l KLogHandler) HandleToolEnd(ctx context.Context, output string) {
	logger := klog.FromContext(ctx)
	logger.WithValues("logger", "arcadia")
	logger.V(l.LogLevel).Info("Exiting tool with output: " + removeNewLines(output))
}

func (l KLogHandler) HandleToolError(ctx context.Context, err error) {
	logger := klog.FromContext(ctx)
	logger.WithValues("logger", "arcadia")
	logger.V(l.LogLevel).Error(err, "Exiting tool with error")
}

func (l KLogHandler) HandleAgentAction(ctx context.Context, action schema.AgentAction) {
	logger := klog.FromContext(ctx)
	logger.WithValues("logger", "arcadia")
	logger.V(l.LogLevel).Info("Agent selected action: " + formatAgentAction(action))
}

func (l KLogHandler) HandleAgentFinish(ctx context.Context, finish schema.AgentFinish) {
	logger := klog.FromContext(ctx)
	logger.WithValues("logger", "arcadia")
	logger.V(l.LogLevel).Info("Agent finish: " + formatAgentFinish(finish))
}

func (l KLogHandler) HandleRetrieverStart(ctx context.Context, query string) {
	logger := klog.FromContext(ctx)
	logger.WithValues("logger", "arcadia")
	logger.V(l.LogLevel).Info("Entering retriever with query: " + removeNewLines(query))
}

func (l KLogHandler) HandleRetrieverEnd(ctx context.Context, query string, documents []schema.Document) {
	logger := klog.FromContext(ctx)
	logger.WithValues("logger", "arcadia")
	// TODO need format
	// logger.V(l.LogLevel).Info(fmt.Sprintf("Exiting retriever with documents for query:%s documents: %#v", query, documents))
	logger.V(l.LogLevel).Info(fmt.Sprintf("Exiting retriever with documents for query: %s", query))
}

func formatAgentAction(action schema.AgentAction) string {
	return fmt.Sprintf("\"%s\" with input \"%s\"", removeNewLines(action.Tool), removeNewLines(action.ToolInput))
}
func formatAgentFinish(finish schema.AgentFinish) string {
	return fmt.Sprintf("ReturnValues: %#v Log: %s", removeNewLines(finish.ReturnValues), removeNewLines(finish.Log))
}
func removeNewLines(s any) string {
	return strings.ReplaceAll(fmt.Sprint(s), "\n", " ")
}
