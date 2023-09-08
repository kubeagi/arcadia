/*
Copyright 2023 The KubeAGI Authors.

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
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	zhipuaiembeddings "github.com/kubeagi/arcadia/pkg/embeddings/zhipuai"
	"github.com/kubeagi/arcadia/pkg/llms/zhipuai"
	"github.com/kubeagi/arcadia/pkg/vectorstores/chromadb"
	"github.com/spf13/cobra"
	"github.com/tmc/langchaingo/schema"
)

const (
	_defaultNamespace = "arcadia"
)

var (
	apiKey    string
	addr      string
	url       string
	namespace string
)

func NewStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start [usage]",
		Short: "Start the server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run()
		},
	}

	cmd.Flags().StringVar(&apiKey, "api-key", "", "used to connect to ZhiPuAI platform")
	cmd.Flags().StringVar(&url, "vector-store", "", "the chromaDB vector database url")
	cmd.Flags().StringVar(&addr, "addr", ":8800", "used to listen and serve GET request, default :8800")
	cmd.Flags().StringVar(&namespace, "name-space", _defaultNamespace, "the vector database namespace")

	if err := cmd.MarkFlagRequired("api-key"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("vector-store"); err != nil {
		panic(err)
	}

	return cmd
}

func run() error {
	fmt.Println("Starting chat server example...")

	// check ZhiPuAI api key, build embedder
	if apiKey == "" {
		return fmt.Errorf("ZhiPuAI api key is empty")
	}
	if url == "" {
		return fmt.Errorf("chromaDB scheme is empty")
	}

	fmt.Println("Connecting platform...")
	z := zhipuai.NewZhiPuAI(apiKey)
	_, err := z.Validate()
	if err != nil {
		return fmt.Errorf("error validating ZhiPuAI api key: %s", err.Error())
	}

	embedder, err := zhipuaiembeddings.NewZhiPuAI(
		zhipuaiembeddings.WithClient(*zhipuai.NewZhiPuAI(apiKey)),
	)
	if err != nil {
		return fmt.Errorf("error creating ZhiPuAI embedder: %s", err.Error())
	}

	fmt.Println("Connecting vector database...")
	_, err = chromadb.New(
		chromadb.WithURL(url),
		chromadb.WithEmbedder(embedder),
		chromadb.WithNameSpace(namespace),
	)
	if err != nil {
		return fmt.Errorf("error connecting chroma db: %s", err.Error())
	}

	fmt.Println("Creating HTTP server...")
	app := fiber.New(fiber.Config{
		AppName:       "chat-server",
		CaseSensitive: true,
		StrictRouting: true,
		Immutable:     true,
	})

	app.Use(cors.New(cors.ConfigDefault))

	app.Get("/", HomePageGetHandler)
	app.Post("/load", LoadHandler)
	app.Post("/chat", QueryHandler)

	return app.Listen(addr)
}

func buildPrompt(question string, document []schema.Document) []zhipuai.Prompt {
	premise := zhipuai.Prompt{
		Role: zhipuai.User,
		Content: `
		我将要询问一些问题，希望你仅使用我提供的上下文信息回答。
		请不要在回答中添加其他信息。
		若我提供的上下文不足以回答问题,
		请回复"我不确定"，再做出适当的猜测。
		请将回答内容分割为适于阅读的段落。
		`,
	}
	reply := zhipuai.Prompt{
		Role: zhipuai.Assistant,
		Content: `
		好的，我将尝试仅使用你提供的上下文信息回答，并在信息不足时提供一些合理推测。
		`,
	}

	var info string
	for _, doc := range document {
		info += doc.PageContent
	}

	requirement := zhipuai.Prompt{
		Role:    zhipuai.User,
		Content: "问题内容如下：" + question + "\n  以下是我提供的上下文信息:\n" + info,
	}

	return []zhipuai.Prompt{premise, reply, requirement}
}
