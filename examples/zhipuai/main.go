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
	"os"

	"github.com/kubeagi/arcadia/pkg/llms/zhipuai"
	"k8s.io/klog/v2"
)

func main() {
	if len(os.Args) == 0 {
		panic("api key is empty")
	}
	apiKey := os.Args[1]

	klog.V(0).Info("try `Invoke`")
	resp, err := sampleInvoke(apiKey)
	if err != nil {
		panic(err)
	}
	klog.V(0).Info("Response: \n %s\n", resp.String())

	klog.V(0).Info("try `AsyncInvoke`")
	resp, err = sampleInvokeAsync(apiKey)
	if err != nil {
		panic(err)
	}
	klog.V(0).Info("Response: \n %s\n", resp.String())

	var taskID string
	if resp.Data != nil {
		taskID = resp.Data.TaskID
	}
	if taskID == "" {
		panic("Failed to get task id from previous AsyncInvoke response")
	}

	klog.V(0).Info("try `getInvokeAsyncResult` with previous task id")
	resp, err = getInvokeAsyncResult(apiKey, taskID)
	if err != nil {
		panic(err)
	}
	klog.V(0).Info("Response: \n %s\n", resp.String())

	klog.V(0).Info("try `SSEInvoke` with default handler")
	err = sampleSSEInvoke(apiKey)
	if err != nil {
		panic(err)
	}
}

func sampleInvoke(apiKey string) (*zhipuai.Response, error) {
	client := zhipuai.NewZhiPuAI(apiKey)
	params := zhipuai.DefaultModelParams()
	params.Prompt = []zhipuai.Prompt{
		{Role: zhipuai.User, Content: "As a kubernetes expert,please answer the following questions."},
	}
	return client.Invoke(params)
}

func sampleInvokeAsync(apiKey string) (*zhipuai.Response, error) {
	client := zhipuai.NewZhiPuAI(apiKey)
	params := zhipuai.DefaultModelParams()
	params.Prompt = []zhipuai.Prompt{
		{Role: zhipuai.User, Content: "As a kubernetes expert,please answer the following questions."},
	}
	return client.AsyncInvoke(params)
}

func getInvokeAsyncResult(apiKey string, taskID string) (*zhipuai.Response, error) {
	client := zhipuai.NewZhiPuAI(apiKey)
	params := zhipuai.DefaultModelParams()
	params.TaskID = taskID
	return client.Get(params)
}

func sampleSSEInvoke(apiKey string) error {
	client := zhipuai.NewZhiPuAI(apiKey)
	params := zhipuai.DefaultModelParams()
	params.Prompt = []zhipuai.Prompt{
		{Role: zhipuai.User, Content: "As a kubernetes expert,please answer the following questions."},
	}
	// you can define a customized `handler` on `Event`
	err := client.SSEInvoke(params, nil)
	if err != nil {
		return err
	}
	return nil
}
