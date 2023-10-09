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

package llms

import "errors"

type LLMType string

const (
	OpenAI    LLMType = "openai"
	ZhiPuAI   LLMType = "zhipuai"
	DashScope LLMType = "dashscope"
	Unknown   LLMType = "unknown"
)

type LLM interface {
	Type() LLMType
	Call([]byte) (Response, error)
	Validate() (Response, error)
}

type ModelParams interface {
	Marshal() []byte
	Unmarshal([]byte) error
}

type Response interface {
	Type() LLMType
	String() string
	Bytes() []byte
	Unmarshal([]byte) error
}

type UnknowLLM struct{}

func NewUnknowLLM() UnknowLLM {
	return UnknowLLM{}
}
func (unknown UnknowLLM) Type() LLMType {
	return Unknown
}

func (unknown UnknowLLM) Call(data []byte) (Response, error) {
	return nil, errors.New("unknown llm type")
}

func (unknown UnknowLLM) Validate() (Response, error) {
	return nil, errors.New("unknown llm type")
}
