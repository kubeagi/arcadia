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

package agent

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/callbacks"
	"k8s.io/klog/v2"

	"github.com/kubeagi/arcadia/pkg/appruntime/base"
)

// StreamHandler is a callback handler that prints to the standard output streaming.
type StreamHandler struct {
	callbacks.SimpleHandler
	args map[string]any
}

var _ callbacks.Handler = StreamHandler{}

func (handler StreamHandler) HandleStreamingFunc(ctx context.Context, chunk []byte) {
	logger := klog.FromContext(ctx)
	if _, ok := handler.args[base.OutputAnserStreamChanKeyInArg]; !ok {
		logger.Info("no _answer_stream found, create a new one")
		handler.args[base.OutputAnserStreamChanKeyInArg] = make(chan string)
	}
	streamChan, ok := handler.args[base.OutputAnserStreamChanKeyInArg].(chan string)
	if !ok {
		err := fmt.Errorf("answer_stream is not chan string, but %T", handler.args[base.OutputAnserStreamChanKeyInArg])
		logger.Error(err, "answer_stream is not chan string")
		return
	}
	logger.V(5).Info("stream out:" + string(chunk))
	streamChan <- string(chunk)
}
