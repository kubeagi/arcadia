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
	"fmt"
	"os"
	"time"

	"github.com/kubeagi/arcadia/pkg/llms"
	"github.com/kubeagi/arcadia/pkg/llms/dashscope"
	"k8s.io/klog/v2"
)

const (
	samplePrompt           = "how to change a deployment's image?"
	sampleEmbeddingTextURL = "https://gist.githubusercontent.com/Abirdcfly/e66c1fbd48dbdd89398123362660828b/raw/de30715a99f32b66959f3c4c96b53db82554fa40/demo.txt"
)

var sampleEmbeddingText = []string{"离离原上草", "一岁一枯荣", "野火烧不尽", "春风吹又生"}

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
	for _, model := range []dashscope.Model{dashscope.LLAMA27BCHATV2, dashscope.LLAMA213BCHATV2, dashscope.BAICHUAN7BV1, dashscope.CHATGLM6BV2} {
		klog.V(0).Infof("\nChat with %s\n", model)
		resp, err := sampleChatWithOthers(apiKey, model)
		if err != nil {
			panic(err)
		}
		klog.V(0).Infof("Response: \n %s\n", resp)
	}
	klog.Infoln("sample chat done")
	klog.Infof("\nsample embedding start...\nwe use same embedding: %s to test\n", sampleEmbeddingText)
	resp, err := sampleEmbedding(apiKey)
	if err != nil {
		panic(err)
	}
	klog.V(0).Infof("embedding sync call return: \n %+v\n", resp)

	taskID, err := sampleEmbeddingAsync(apiKey, sampleEmbeddingTextURL)
	if err != nil {
		panic(err)
	}
	klog.V(0).Infof("embedding async call will return taskID: %s", taskID)
	klog.V(0).Infoln("wait 3s to make the task done")
	time.Sleep(3 * time.Second)
	downloadURL, err := sampleEmbeddingAsyncGetTaskDetail(apiKey, taskID)
	if err != nil {
		panic(err)
	}
	klog.V(0).Infof("get download url: %s\n", downloadURL)
	localfile := "/tmp/embedding.txt"
	klog.V(0).Infof("download and extract the embedding file to %s...\n", localfile)
	err = dashscope.DownloadAndExtract(downloadURL, localfile)
	if err != nil {
		panic(err)
	}
	content, err := os.ReadFile(localfile)
	if err != nil {
		panic(err)
	}
	klog.V(0).Infoln("show the embedding file content:")
	fmt.Println(string(content))
	klog.Infoln("sample embedding done")
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

func sampleChatWithOthers(apiKey string, model dashscope.Model) (llms.Response, error) {
	client := dashscope.NewDashScope(apiKey, false)
	params := dashscope.DefaultModelParamsSimpleChat()
	if model == dashscope.CHATGLM6BV2 {
		params.Input.History = &[]string{}
	}
	params.Model = model
	params.Input.Prompt = samplePrompt
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

func sampleEmbedding(apiKey string) ([]dashscope.Embeddings, error) {
	client := dashscope.NewDashScope(apiKey, false)
	return client.CreateEmbedding(context.TODO(), sampleEmbeddingText, false)
}

func sampleEmbeddingAsync(apiKey string, url string) (string, error) {
	client := dashscope.NewDashScope(apiKey, false)
	return client.CreateEmbeddingAsync(context.TODO(), url, false)
}

func sampleEmbeddingAsyncGetTaskDetail(apiKey string, taskID string) (string, error) {
	client := dashscope.NewDashScope(apiKey, false)
	return client.GetTaskDetail(context.TODO(), taskID)
}
