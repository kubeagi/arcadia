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
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tmc/langchaingo/embeddings"
	openaiEmbeddings "github.com/tmc/langchaingo/embeddings/openai"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"

	zhipuaiembeddings "github.com/kubeagi/arcadia/pkg/embeddings/zhipuai"
	"github.com/kubeagi/arcadia/pkg/llms"
	"github.com/kubeagi/arcadia/pkg/llms/zhipuai"
	"github.com/kubeagi/arcadia/pkg/vectorstores/chromadb"
)

var (
	question string
	// chat with LLM
	chatLLMType string
	chatAPIKey  string
	model       string
	method      string
	temperature float32
	topP        float32

	// similarity search
	enableSimilaritySearch bool
	scoreThreshold         float64
	numDocs                int

	// promptsWithSimilaritySearch = []zhipuai.Prompt{
	// 	{
	// 		Role: zhipuai.User,
	// 		Content: `Hi there, I am going to ask you a question, which I would like you to answer
	// 	based only on the provided context, and not any other information.
	//     If there is not enough information in the context to answer the question,'say \"I am not sure\", then try to make a guess.'
	//     Break your answer up into nicely readable paragraphs.`,
	// 	},
	// 	{
	// 		Role:    zhipuai.Assistant,
	// 		Content: "Sure.Please provide your documents.",
	// 	},
	// }
	promptsWithSimilaritySearchCN = []zhipuai.Prompt{
		{
			Role: zhipuai.User,
			Content: `
      我将要询问一些问题，希望你仅使用我提供的上下文信息回答。
      请不要在回答中添加其他信息。
      若我提供的上下文不足以回答问题,
      请回复"我不确定"，再做出适当的猜测。
      请将回答内容分割为适于阅读的段落。
      `,
		},
		{
			Role: zhipuai.Assistant,
			Content: `
      	好的，我将尝试仅使用你提供的上下文信息回答，并在信息不足时提供一些合理推测。
      `,
		},
	}
)

func NewChatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chat [usage]",
		Short: "Do LLM chat with similarity search(optional)",
		RunE: func(cmd *cobra.Command, args []string) error {
			var docs []schema.Document
			var err error

			if enableSimilaritySearch {
				docs, err = SimilaritySearch(context.Background())
				if err != nil {
					return err
				}
				fmt.Printf("similarDocs: %v \n", docs)
			}

			return Chat(context.Background(), docs)
		},
	}

	// For similarity search
	cmd.Flags().BoolVar(&enableSimilaritySearch, "enable-embedding-search", false, "enable embedding similarity search")
	cmd.Flags().StringVar(&embeddingLLMType, "embedding-llm-type", string(llms.ZhiPuAI), "llm type to use(Only zhipuai,openai supported now)")
	cmd.Flags().StringVar(&embeddingLLMApiKey, "embedding-llm-apikey", "", "apiKey to access embedding service.Must required when embedding similarity search is enabled")
	cmd.Flags().StringVar(&vectorStore, "vector-store", "http://localhost:8000", "vector stores to use(Only chroma supported now)")
	cmd.Flags().StringVar(&nameSpace, "namespace", "arcadia", "namespace/collection to query from")
	cmd.Flags().Float64Var(&scoreThreshold, "score-threshold", 0, "score threshold for similarity search(Higher is better)")
	cmd.Flags().IntVar(&numDocs, "num-docs", 5, "number of documents to be returned with SimilarSearch")

	cmd.Flags().StringVar(&question, "question", "", "question text to be asked")

	// For LLM chat
	cmd.Flags().StringVar(&chatLLMType, "chat-llm-type", string(llms.ZhiPuAI), "llm type to use(Only zhipuai,openai supported now)")
	cmd.Flags().StringVar(&chatAPIKey, "chat-llm-apikey", "", "apiKey to access embedding service")
	cmd.PersistentFlags().StringVar(&model, "model", string(zhipuai.ZhiPuAILite), "which model to use: chatglm_lite/chatglm_std/chatglm_pro")
	cmd.PersistentFlags().StringVar(&method, "method", "sse-invoke", "Invoke method used when access LLM service(invoke/sse-invoke)")
	cmd.PersistentFlags().Float32Var(&temperature, "temperature", 0.95, "temperature for chat")
	cmd.PersistentFlags().Float32Var(&topP, "top-p", 0.7, "top-p for chat")

	if err = cmd.MarkFlagRequired("chat-llm-apikey"); err != nil {
		panic(err)
	}
	if err = cmd.MarkFlagRequired("question"); err != nil {
		panic(err)
	}

	return cmd
}

func SimilaritySearch(ctx context.Context) ([]schema.Document, error) {
	var embedder embeddings.Embedder
	var err error

	if embeddingLLMApiKey == "" {
		return nil, errors.New("embedding-llm-apikey is required when embedding similarity search is enabled")
	}

	switch embeddingLLMType {
	case "zhipuai":
		embedder, err = zhipuaiembeddings.NewZhiPuAI(
			zhipuaiembeddings.WithClient(*zhipuai.NewZhiPuAI(embeddingLLMApiKey)),
		)
		if err != nil {
			return nil, err
		}
	case "openai":
		embedder, err = openaiEmbeddings.NewOpenAI()
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unsupported embedding type")
	}

	chroma, err := chromadb.New(
		chromadb.WithURL(vectorStore),
		chromadb.WithEmbedder(embedder),
		chromadb.WithNameSpace(nameSpace),
	)
	if err != nil {
		return nil, err
	}

	return chroma.SimilaritySearch(ctx, question, numDocs, vectorstores.WithScoreThreshold(scoreThreshold))
}

func Chat(ctx context.Context, similarDocs []schema.Document) error {
	// Only for zhipuai
	client := zhipuai.NewZhiPuAI(chatAPIKey)

	params := zhipuai.DefaultModelParams()
	params.Model = zhipuai.Model(model)
	params.Method = zhipuai.Method(method)
	params.Temperature = temperature
	params.TopP = topP

	var prompts []zhipuai.Prompt
	if len(similarDocs) != 0 {
		var docString string
		for _, doc := range similarDocs {
			docString += doc.PageContent
		}
		prompts = append(prompts, promptsWithSimilaritySearchCN...)
		prompts = append(prompts, zhipuai.Prompt{
			Role:    zhipuai.User,
			Content: fmt.Sprintf("我的问题是: %s. 以下是我提供的上下文:%s", question, docString),
		})
	} else {
		prompts = append(prompts, zhipuai.Prompt{
			Role:    zhipuai.User,
			Content: question,
		})
	}

	fmt.Printf("Prompts: %v \n", prompts)
	params.Prompt = prompts
	if params.Method == zhipuai.ZhiPuAIInvoke {
		resp, err := client.Invoke(params)
		if err != nil {
			return err
		}
		if resp.Code != 200 {
			return fmt.Errorf("chat failed: %s", resp.String())
		}
		fmt.Println(resp.Data.Choices[0].Content)
		return nil
	}

	err := client.SSEInvoke(params, nil)
	if err != nil {
		return err
	}

	return nil
}
