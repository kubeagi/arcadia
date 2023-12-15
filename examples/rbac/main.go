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
	"github.com/kubeagi/arcadia/pkg/llms"
	"github.com/spf13/cobra"
)

var (
	apiKey string
	model  string
	method string
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rbac [usage]",
		Short: "A Command line tool for rbac checks(Only ZhiPuAI/ChatGLM supported now)",
	}

	cmd.PersistentFlags().StringVar(&apiKey, "apiKey", "", "apiKey to access LLM service")
	cmd.PersistentFlags().StringVar(&model, "model", string(llms.ZhiPuAILite), "which model to use: chatglm_lite/chatglm_std/chatglm_pro")
	cmd.PersistentFlags().StringVar(&method, "method", "sse-invoke", "Invoke method used when access LLM service(invoke/sse-invoke)")

	cmd.MarkPersistentFlagRequired("apiKey")

	cmd.AddCommand(Inquiry())

	return cmd
}

func main() {
	if err := NewCmd().Execute(); err != nil {
		panic(err)
	}

}
