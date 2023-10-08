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

package main

import (
	"context"
	"os"

	"github.com/kubeagi/arcadia/pkg/llms"
	"github.com/kubeagi/arcadia/pkg/llms/dashscope"
	"k8s.io/klog/v2"
)

const (
	samplePrompt = "how to change a deployment's image?"
)

func main() {
	if len(os.Args) == 1 {
		panic("api key is empty")
	}
	apiKey := os.Args[1]
	klog.Infof("sample chat start...\nwe use same prompt: %s to test\n", samplePrompt)
	for _, model := range []dashscope.Model{dashscope.QWEN14BChat, dashscope.QWEN7BChat} {
		klog.V(0).Infof("\nChat with %s\n", model)
		resp, err := sampleChat(apiKey, model)
		if err != nil {
			panic(err)
		}
		klog.V(0).Infof("Response: \n %s\n", resp)
		klog.V(0).Infoln("\nChat again with sse enable")
		err = sampleSSEChat(apiKey, model)
		if err != nil {
			panic(err)
		}
	}
	klog.Infoln("sample chat done")
}

func sampleChat(apiKey string, model dashscope.Model) (llms.Response, error) {
	client := dashscope.NewDashScope(apiKey, false)
	params := dashscope.DefaultModelParams()
	params.Model = model
	params.Input.Messages = []dashscope.Message{
		{Role: dashscope.System, Content: "You are a kubernetes expert."},
		{Role: dashscope.User, Content: samplePrompt},
	}
	return client.Call(params.Marshal())
}

func sampleSSEChat(apiKey string, model dashscope.Model) error {
	client := dashscope.NewDashScope(apiKey, true)
	params := dashscope.DefaultModelParams()
	params.Model = model
	params.Input.Messages = []dashscope.Message{
		{Role: dashscope.System, Content: "You are a kubernetes expert."},
		{Role: dashscope.User, Content: samplePrompt},
	}
	// you can define a customized `handler` on `Event`
	err := client.StreamCall(context.TODO(), params.Marshal(), nil)
	if err != nil {
		return err
	}
	return nil
}
