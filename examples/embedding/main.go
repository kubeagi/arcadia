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
	llmszhipuai "github.com/kubeagi/arcadia/pkg/llms/zhipuai"
	"k8s.io/klog/v2"
	"os"
)

func main() {
	// usage: go run main.go <apiKey> <testWord>
	// apiKey 	== The apiKey of user, available on ZhiPuAI dashboard.
	// testWord == The text to be embedded.
	if len(os.Args) == 1 {
		panic("api key is empty")
	}
	apiKey := os.Args[1]

	if len(os.Args) == 2 {
		panic("test word is empty")
	}
	testWord := os.Args[2]

	klog.V(0).Info("try `Embedding`")
	resp, err := sampleEmbedding(apiKey, testWord)

	if err != nil {
		panic(err)
	}
	klog.V(0).Info("Response: \n", resp.Success)
	klog.V(0).Info("Usage: \n", resp.Data.Usage)
	klog.V(0).Info("First 5 number of the Result: \n", resp.Data.Embedding[:5])
	// resp.Data.Embedding is available.
}

func sampleEmbedding(apiKey, testWord string) (*llmszhipuai.EmbeddingResponse, error) {
	ai := llmszhipuai.NewZhiPuAI(apiKey)
	text := llmszhipuai.EmbeddingText{
		Prompt:    testWord,
		RequestID: "1",
	}

	resp, err := ai.Embedding(text)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
