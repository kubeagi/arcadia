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
	llmzhipuai "github.com/kubeagi/arcadia/pkg/llms/zhipuai"
)

const (
	_defaultBatchSize     = 512
	_defaultStripNewLines = true
)

// Option is a function type that can be used to modify the client.
type Option func(p *ZhiPuAI)

// WithClient is an option for providing the LLM client.
func WithClient(client llmzhipuai.ZhiPuAI) Option {
	return func(p *ZhiPuAI) {
		p.client = &client
	}
}

// WithStripNewLines is an option for specifying the should it strip new lines.
func WithStripNewLines(stripNewLines bool) Option {
	return func(p *ZhiPuAI) {
		p.StripNewLines = stripNewLines
	}
}

// WithBatchSize is an option for specifying the batch size.
func WithBatchSize(batchSize int) Option {
	return func(p *ZhiPuAI) {
		p.BatchSize = batchSize
	}
}

func applyClientOptions(opts ...Option) (ZhiPuAI, error) {
	o := &ZhiPuAI{
		StripNewLines: _defaultStripNewLines,
		BatchSize:     _defaultBatchSize,
	}

	for _, opt := range opts {
		opt(o)
	}

	if o.client == nil {
		o.client = llmzhipuai.NewZhiPuAI("")
	}

	return *o, nil
}
