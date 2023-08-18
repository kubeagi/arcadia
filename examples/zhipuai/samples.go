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

import "github.com/kubeagi/arcadia/pkg/llms/zhipuai"

func sampleInvoke(apiKey string) (map[string]interface{}, error) {
	client := zhipuai.NewZhiPuAI(apiKey)
	params := zhipuai.DefaultModelParams()
	params.Prompt = []zhipuai.Prompt{
		{Role: zhipuai.User, Content: "As a kubernetes expert,please answer the following questions."},
	}
	return client.Invoke(params)
}

func sampleInvokeAsync(apiKey string) (map[string]interface{}, error) {
	client := zhipuai.NewZhiPuAI(apiKey)
	params := zhipuai.DefaultModelParams()
	params.Method = zhipuai.ZhiPuAIAsyncInvoke
	params.Prompt = []zhipuai.Prompt{
		{Role: zhipuai.User, Content: "As a kubernetes expert,please answer the following questions."},
	}
	return client.AsyncInvoke(params)
}

func getInvokeAsyncResult(apiKey string, taskID string) (map[string]interface{}, error) {
	client := zhipuai.NewZhiPuAI(apiKey)
	params := zhipuai.DefaultModelParams()
	params.Method = zhipuai.ZhiPuAIAsyncGet
	params.TaskID = taskID
	return client.Get(params)
}
