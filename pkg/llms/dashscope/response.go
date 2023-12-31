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
	"bytes"
	"encoding/json"

	"github.com/kubeagi/arcadia/pkg/llms"
)

var _ llms.Response = (*Response)(nil)
var _ llms.Response = (*ResponseChatGLB6B)(nil)

type CommonResponse struct {
	// https://help.aliyun.com/zh/dashscope/response-status-codes
	StatusCode int    `json:"status_code,omitempty"`
	Code       string `json:"code,omitempty"`
	Message    string `json:"message,omitempty"`
	RequestID  string `json:"request_id"`
}
type Response struct {
	CommonResponse
	Output Output `json:"output"`
	Usage  Usage  `json:"usage"`
}

type Output struct {
	Choices []Choice `json:"choices,omitempty"`
	Text    string   `json:"text,omitempty"`
	History []string `json:"history,omitempty"`
}

type FinishReason string

const (
	Finish     FinishReason = "stop"
	Generating FinishReason = "null"
	ToLoogin   FinishReason = "length"
)

type Choice struct {
	FinishReason FinishReason `json:"finish_reason"`
	Message      Message      `json:"message"`
}

type Usage struct {
	OutputTokens int `json:"output_tokens"`
	InputTokens  int `json:"input_tokens"`
}

func (response *Response) Unmarshal(bytes []byte) error {
	return json.Unmarshal(bytes, response)
}

func (response *Response) Type() llms.LLMType {
	return llms.DashScope
}

func (response *Response) Bytes() []byte {
	bytes, err := json.Marshal(response)
	if err != nil {
		return []byte{}
	}
	return bytes
}

func (response *Response) String() string {
	if response.Output.Text != "" {
		return response.Output.Text
	}
	buf := &bytes.Buffer{}
	for _, c := range response.Output.Choices {
		buf.WriteString(c.Message.Content)
	}
	return buf.String()
}

type ResponseChatGLB6B struct {
	CommonResponse
	Output struct {
		Text struct {
			Response string `json:"response,omitempty"`
		} `json:"text,omitempty"`
		History []string `json:"history,omitempty"`
	} `json:"output"`
	Usage Usage `json:"usage"`
}

func (r *ResponseChatGLB6B) Type() llms.LLMType {
	return llms.DashScope
}

func (r *ResponseChatGLB6B) String() string {
	return r.Output.Text.Response
}

func (r *ResponseChatGLB6B) Bytes() []byte {
	bytes, err := json.Marshal(r)
	if err != nil {
		return []byte{}
	}
	return bytes
}

func (r *ResponseChatGLB6B) Unmarshal(bytes []byte) error {
	return json.Unmarshal(bytes, r)
}
