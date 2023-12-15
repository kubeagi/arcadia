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

package arctl

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/chroma"

	zhipuaiembeddings "github.com/kubeagi/arcadia/pkg/embeddings/zhipuai"
	"github.com/kubeagi/arcadia/pkg/llms"
	"github.com/kubeagi/arcadia/pkg/llms/zhipuai"
)

var (
	question string

	// chat with LLM
	model       string
	method      string
	temperature float32
	topP        float32

	// similarity search
	scoreThreshold float32
	numDocs        int

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

func NewChatCmd(homePath string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chat [usage]",
		Short: "Do LLM chat with similarity search(optional)",
		RunE: func(cmd *cobra.Command, args []string) error {
			var docs []schema.Document
			var err error

			if dataset != "" {
				docs, err = SimilaritySearch(context.Background(), homePath)
				if err != nil {
					return err
				}
				fmt.Printf("similarDocs: %v \n", docs)
			}

			return Chat(context.Background(), docs)
		},
	}

	// Similarity search params
	cmd.Flags().StringVar(&dataset, "dataset", "", "dataset(namespace/collection) to query from")
	cmd.Flags().Float32Var(&scoreThreshold, "score-threshold", 0, "score threshold for similarity search(Higher is better)")
	cmd.Flags().IntVar(&numDocs, "num-docs", 5, "number of documents to be returned with SimilarSearch")

	// For LLM chat
	cmd.Flags().StringVar(&llmType, "llm-type", string(llms.ZhiPuAI), "llm type to use for embedding & chat(Only zhipuai,openai supported now)")
	cmd.Flags().StringVar(&apiKey, "llm-apikey", "", "apiKey to access llm service.Must required when embedding similarity search is enabled")
	cmd.Flags().StringVar(&question, "question", "", "question text to be asked")
	if err := cmd.MarkFlagRequired("llm-apikey"); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired("question"); err != nil {
		panic(err)
	}

	// LLM Chat params
	cmd.PersistentFlags().StringVar(&model, "model", string(llms.ZhiPuAILite), "which model to use: chatglm_lite/chatglm_std/chatglm_pro")
	cmd.PersistentFlags().StringVar(&method, "method", "sse-invoke", "Invoke method used when access LLM service(invoke/sse-invoke)")
	cmd.PersistentFlags().Float32Var(&temperature, "temperature", 0.95, "temperature for chat")
	cmd.PersistentFlags().Float32Var(&topP, "top-p", 0.7, "top-p for chat")

	return cmd
}

func SimilaritySearch(ctx context.Context, homePath string) ([]schema.Document, error) {
	var embedder embeddings.Embedder
	var err error

	ds, err := loadCachedDataset(filepath.Join(homePath, "dataset", dataset))
	if err != nil {
		return nil, err
	}
	if ds.Name == "" {
		return nil, fmt.Errorf("dataset %s does not exist", dataset)
	}

	switch ds.LLMType {
	case "zhipuai":
		embedder, err = zhipuaiembeddings.NewZhiPuAI(
			zhipuaiembeddings.WithClient(*zhipuai.NewZhiPuAI(ds.LLMApiKey)),
		)
		if err != nil {
			return nil, err
		}
	case "openai":
		llm, err := openai.New()
		if err != nil {
			return nil, err
		}
		embedder, err = embeddings.NewEmbedder(llm)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("unsupported embedding type")
	}

	chroma, err := chroma.New(
		chroma.WithChromaURL(ds.VectorStore),
		chroma.WithEmbedder(embedder),
		chroma.WithNameSpace(dataset),
	)
	if err != nil {
		return nil, err
	}

	return chroma.SimilaritySearch(ctx, question, numDocs, vectorstores.WithScoreThreshold(scoreThreshold))
}

func Chat(ctx context.Context, similarDocs []schema.Document) error {
	// Only for zhipuai
	client := zhipuai.NewZhiPuAI(apiKey)

	params := zhipuai.DefaultModelParams()
	params.Model = model
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
