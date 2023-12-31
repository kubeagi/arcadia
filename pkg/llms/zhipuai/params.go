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

// NOTE: Reference zhipuai's python sdk: model_api/params.py

package zhipuai

import (
	"encoding/json"
	"errors"

	"github.com/kubeagi/arcadia/pkg/llms"
)

type Role string

const (
	User      Role = "user"
	Assistant Role = "assistant"
)

var _ llms.ModelParams = (*ModelParams)(nil)

// +kubebuilder:object:generate=true
// ZhiPuAIParams defines the params of ZhiPuAI Prompt Call
type ModelParams struct {
	// Method used for this prompt call
	Method Method `json:"method,omitempty"`

	// Model used for this prompt call
	Model string `json:"model,omitempty"`

	// Temperature is float in zhipuai
	Temperature float32 `json:"temperature,omitempty"`
	// TopP is float in zhipuai
	TopP float32 `json:"top_p,omitempty"`
	// Contents
	Prompt []Prompt `json:"prompt"`

	// TaskID is used for getting result of AsyncInvoke
	TaskID string `json:"task_id,omitempty"`

	// Incremental is only Used for SSE Invoke
	Incremental bool `json:"incremental,omitempty"`
}

// +kubebuilder:object:generate=true
// Prompt defines the content of ZhiPuAI Prompt Call
type Prompt struct {
	Role    Role   `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

func DefaultModelParams() ModelParams {
	// TODO: should allow user to configure the temperature and top_p of inference
	// use 0.8 and 0.7 for now
	return ModelParams{
		Model:       llms.ZhiPuAILite,
		Method:      ZhiPuAIInvoke,
		Temperature: 0.8, // more accurate?
		TopP:        0.7,
		Prompt:      []Prompt{},
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

// MergeZhiPuAI merges b to a  with this rule
// - if a.x is empty and b.x is not, then a.x = b.x
func MergeParams(a, b ModelParams) ModelParams {
	if a.Model == "" && b.Model != "" {
		a.Model = b.Model
	}
	if a.Method == "" && b.Method != "" {
		a.Method = b.Method
	}
	if a.Temperature == 0 && b.Temperature != 0 {
		a.Temperature = b.Temperature
	}
	if a.TopP == 0 && b.TopP != 0 {
		a.TopP = b.TopP
	}
	if !a.Incremental && b.Incremental {
		a.Incremental = b.Incremental
	}
	if len(a.Prompt) == 0 && len(b.Prompt) > 0 {
		a.Prompt = b.Prompt
	}
	return a
}

func ValidateModelParams(params ModelParams) error {
	if params.Model == "" || params.Method == "" {
		return errors.New("model or method is required")
	}

	if params.Temperature < 0 || params.Temperature > 1 {
		return errors.New("temperature must be in [0, 1]")
	}

	if params.TopP < 0 || params.TopP > 1 {
		return errors.New("top_p must be in [0, 1]")
	}

	switch params.Method {
	case ZhiPuAIInvoke, ZhiPuAIAsyncInvoke, ZhiPuAISSEInvoke:
	case ZhiPuAIAsyncGet:
		if params.TaskID == "" {
			return errors.New("task_id is required")
		}
	default:
		return errors.New("method must be one of [invoke, async-invoke, sse-invoke,get]")
	}

	return nil
}
