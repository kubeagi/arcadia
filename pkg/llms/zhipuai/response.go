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
	"encoding/json"

	"github.com/kubeagi/arcadia/pkg/llms"
)

type Response struct {
	Code    int    `json:"code"`
	Data    *Data  `json:"data"`
	Msg     string `json:"msg"`
	Success bool   `json:"success"`
}

func (response *Response) Type() llms.LLMType {
	return llms.ZhiPuAI
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

type Data struct {
	RequestID  string `json:"request_id,omitempty"`
	TaskID     string `json:"task_id,omitempty"`
	TaskStatus string `json:"task_status,omitempty"`
	Usage      Usage  `json:"usage,omitempty"`

	Choices []Choice `json:"choices,omitempty"`
}

type Usage struct {
	TotalTokens int `json:"total_tokens,omitempty"`
}

type Choice struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}
