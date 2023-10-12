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
	"errors"

	"github.com/kubeagi/arcadia/pkg/llms"
)

type Role string

const (
	System    Role = "system"
	User      Role = "user"
	Assistant Role = "assistant"
)

var _ llms.ModelParams = (*ModelParams)(nil)

// +kubebuilder:object:generate=true

// ModelParams
// ref: https://help.aliyun.com/zh/dashscope/developer-reference/tongyi-qianwen-7b-14b-api-detailes#25745d61fbx49
// do not use 'input.history', according to the above document, this parameter will be deprecated soon.
// use 'message' in 'parameters.result_format' to keep better compatibility.
type ModelParams struct {
	Model      Model      `json:"model"`
	Input      Input      `json:"input"`
	Parameters Parameters `json:"parameters,omitempty"`
}

// +kubebuilder:object:generate=true

type Input struct {
	Messages []Message `json:"messages,omitempty"`
	Prompt   string    `json:"prompt,omitempty"`
	History  *[]string `json:"history,omitempty"`
}

type Parameters struct {
	TopP         float32 `json:"top_p,omitempty"`
	TopK         int     `json:"top_k,omitempty"`
	Seed         int     `json:"seed,omitempty"`
	ResultFormat string  `json:"result_format,omitempty"`
}

// +kubebuilder:object:generate=true

type Message struct {
	Role    Role   `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

func DefaultModelParams() ModelParams {
	return ModelParams{
		Model: QWEN14BChat,
		Input: Input{
			Messages: []Message{},
		},
		Parameters: Parameters{
			TopP:         0.5,
			TopK:         0,
			Seed:         1234,
			ResultFormat: "message",
		},
	}
}
func DefaultModelParamsSimpleChat() ModelParams {
	return ModelParams{
		Model: QWEN14BChat,
		Input: Input{
			Prompt: "",
		},
	}
}

func (params *ModelParams) Marshal() []byte {
	data, err := json.Marshal(params)
	if err != nil {
		return []byte{}
	}
	return data
}

func (params *ModelParams) Unmarshal(bytes []byte) error {
	return json.Unmarshal(bytes, params)
}

func ValidateModelParams(params ModelParams) error {
	if params.Parameters.TopP < 0 || params.Parameters.TopP > 1 {
		return errors.New("top_p must be in (0, 1)")
	}

	return nil
}
