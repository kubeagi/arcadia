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
	"os"

	"github.com/spf13/cobra"
	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/embeddings"
	openaiEmbeddings "github.com/tmc/langchaingo/embeddings/openai"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"

	zhipuaiembeddings "github.com/kubeagi/arcadia/pkg/embeddings/zhipuai"
	"github.com/kubeagi/arcadia/pkg/llms"
	"github.com/kubeagi/arcadia/pkg/llms/zhipuai"
	"github.com/kubeagi/arcadia/pkg/vectorstores/chromadb"
)

var (
	llmType string
	apiKey  string

	document string
	language string

	nameSpace    string
	chunkSize    int
	chunkOverlap int

	vectorStore string
)

func NewLoadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "load [usage]",
		Short: "Load documents into VectorStore",
		RunE: func(cmd *cobra.Command, args []string) error {
			documents, err := LoadAndSplitDocument(context.Background())
			if err != nil {
				return err
			}
			return EmbedDocuments(context.Background(), documents)
		},
	}

	cmd.Flags().StringVar(&llmType, "llm-type", string(llms.ZhiPuAI), "llm type to use(Only zhipuai,openai supported now)")
	cmd.Flags().StringVar(&apiKey, "llm-apikey", "", "apiKey to access embedding service")

	cmd.Flags().StringVar(&vectorStore, "vector-store", "http://localhost:8000", "vector stores to use(Only chroma supported now)")

	cmd.Flags().StringVar(&document, "document", "", "path of the document to load")
	cmd.Flags().StringVar(&language, "document-language", "text", "language of the document(Only text,html,csv supported now)")

	cmd.Flags().StringVar(&nameSpace, "namespace", "arcadia", "namespace/collection of the document to load into")
	cmd.Flags().IntVar(&chunkSize, "chunk-size", 300, "chunk size for embedding")
	cmd.Flags().IntVar(&chunkOverlap, "chunk-overlap", 30, "chunk overlap for embedding")

	if err = cmd.MarkFlagRequired("llm-apikey"); err != nil {
		panic(err)
	}
	if err = cmd.MarkFlagRequired("document"); err != nil {
		panic(err)
	}

	return cmd
}

func LoadAndSplitDocument(ctx context.Context) ([]schema.Document, error) {
	file, err := os.Open(document)
	if err != nil {
		return nil, err
	}

	var loader documentloaders.Loader
	switch language {
	case "text":
		loader = documentloaders.NewText(file)
	case "csv":
		loader = documentloaders.NewCSV(file)
	case "html":
		loader = documentloaders.NewHTML(file)
	default:
		return nil, errors.New("unsupported document language")
	}

	split := textsplitter.NewRecursiveCharacter()
	split.ChunkSize = chunkSize
	split.ChunkOverlap = chunkOverlap

	return loader.LoadAndSplit(ctx, split)
}

func EmbedDocuments(ctx context.Context, documents []schema.Document) error {
	var embedder embeddings.Embedder
	var err error

	switch llmType {
	case "zhipuai":
		embedder, err = zhipuaiembeddings.NewZhiPuAI(
			zhipuaiembeddings.WithClient(*zhipuai.NewZhiPuAI(apiKey)),
		)
		if err != nil {
			return err
		}
	case "openai":
		embedder, err = openaiEmbeddings.NewOpenAI()
		if err != nil {
			return err
		}
	default:
		return errors.New("unsupported embedding type")
	}

	chroma, err := chromadb.New(
		chromadb.WithURL(vectorStore),
		chromadb.WithEmbedder(embedder),
		chromadb.WithNameSpace(nameSpace),
	)
	if err != nil {
		return err
	}

	return chroma.AddDocuments(ctx, documents)
}
