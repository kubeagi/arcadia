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
	"encoding/json"

	"github.com/kubeagi/arcadia/pkg/llms"
)

var _ llms.Response = (*Response)(nil)

type Response struct {
	// https://help.aliyun.com/zh/dashscope/response-status-codes
	StatusCode int    `json:"status_code,omitempty"`
	Code       string `json:"code,omitempty"`
	Message    string `json:"message,omitempty"`
	Output     Output `json:"output"`
	Usage      Usage  `json:"usage"`
	RequestID  string `json:"request_id"`
}

type Output struct {
	Choices []Choice `json:"choices,omitempty"`
	Text    string   `json:"text,omitempty"`
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
	return string(response.Bytes())
}
