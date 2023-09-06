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
	"log"
	"os"

	embedding "github.com/kubeagi/arcadia/pkg/embeddings/zhipuai"
	"github.com/kubeagi/arcadia/pkg/llms/zhipuai"
	"github.com/kubeagi/arcadia/pkg/vectorstores/chromadb"
	"github.com/tmc/langchaingo/schema"
)

func main() {
	if len(os.Args) == 1 {
		panic("api key is empty")
	}
	apiKey := os.Args[1]

	// init embedder
	embedder, err := embedding.NewZhiPuAI(
		embedding.WithClient(*zhipuai.NewZhiPuAI(apiKey)),
	)
	if err != nil {
		panic(fmt.Errorf("error create embedder: %s", err.Error()))
	}
	// init vector store
	chroma, err := chromadb.New(context.TODO(), chromadb.WithBasePath("http://localhost:8000"), chromadb.WithEmbedder(embedder))
	if err != nil {
		panic(fmt.Errorf("error create chroma db: %s", err.Error()))
	}

	// add documents
	err = chroma.AddDocuments(context.TODO(), []schema.Document{
		{PageContent: "This is a document about cats. Cats are great."},
		{PageContent: "this is a document about dogs. Dogs are great."},
	})
	if err != nil {
		panic(fmt.Errorf("error add documents to chroma db: %s", err.Error()))
	}

	docs, err := chroma.SimilaritySearch(context.TODO(), "cats", 1)
	if err != nil {
		log.Fatalf("Error similarity search: %s \n", err.Error())
	}

	fmt.Printf("SimilaritySearch: %v \n", docs)
}
