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
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	zhipuaiembeddings "github.com/kubeagi/arcadia/pkg/embeddings/zhipuai"
	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/textsplitter"

	"github.com/kubeagi/arcadia/pkg/llms/zhipuai"
	"github.com/kubeagi/arcadia/pkg/vectorstores/chromadb"
)

const (
	_defaultChunkSize    = 2048
	_defaultChunkOverlap = 128
)

type Workload struct {
	Document     string `json:"document"`
	ChunkSize    int    `json:"chunk-size"`
	ChunkOverlap int    `json:"chunk-overlap"`
}

type Chat struct {
	Content string `json:"content"`
}

func HomePageGetHandler(c *fiber.Ctx) error {
	return c.SendString("This is the home page of chat server sample. Send POST request to /chat with your question to chat with me!")
}

func LoadHandler(c *fiber.Ctx) error {
	// Convert body to json workload
	var workload Workload
	err := c.BodyParser(&workload)
	if err != nil {
		return errors.New("error parsing body to workload type" + err.Error())
	}

	if workload.Document == "" {
		return errors.New("document cannot be empty")
	}

	err = workload.EmbedAndStoreDocument(context.Background())
	if err != nil {
		return errors.New("Error embedding documents:" + err.Error())
	}

	return c.SendString("OK")
}

func QueryHandler(c *fiber.Ctx) error {
	var chat Chat
	err := c.BodyParser(&chat)
	if chat.Content == "" {
		return errors.New("content cannot be empty")
	}

	embedder, err := zhipuaiembeddings.NewZhiPuAI(
		zhipuaiembeddings.WithClient(*zhipuai.NewZhiPuAI(apiKey)),
	)
	if err != nil {
		return err
	}

	fmt.Println("Connecting vector database...")
	db, err := chromadb.New(
		chromadb.WithURL(url),
		chromadb.WithEmbedder(embedder),
		chromadb.WithNameSpace(namespace),
	)
	if err != nil {
		return fmt.Errorf("error creating chroma db: %s", err.Error())
	}

	res, sErr := db.SimilaritySearch(context.Background(), chat.Content, 5)
	if sErr != nil {
		return fmt.Errorf("error performing similarity search: %s", sErr.Error())
	}

	prompt := buildPrompt(chat.Content, res)

	params := zhipuai.ModelParams{
		Method:      zhipuai.ZhiPuAIInvoke,
		Model:       zhipuai.ZhiPuAIPro,
		Temperature: 0.5,
		TopP:        0.7,
		Prompt:      prompt,
	}

	z := zhipuai.NewZhiPuAI(apiKey)
	resp, iErr := z.Invoke(params)
	if iErr != nil {
		return fmt.Errorf("error invoking ZhiPuAI: %s", iErr.Error())
	}

	return c.SendString(resp.String())
}

func (w Workload) EmbedAndStoreDocument(ctx context.Context) error {
	var embedder embeddings.Embedder
	var err error

	if w.Document == "" {
		return errors.New("document cannot be empty")
	}

	docReader := bytes.NewBufferString(w.Document)

	if err != nil {
		return errors.New("Error reading document:" + err.Error())
	}

	loader := documentloaders.NewText(docReader)

	split := textsplitter.NewRecursiveCharacter()
	if w.ChunkSize > 0 {
		split.ChunkSize = w.ChunkSize
	} else {
		split.ChunkSize = _defaultChunkSize
	}
	if w.ChunkOverlap > 0 {
		split.ChunkOverlap = w.ChunkOverlap
	} else {
		split.ChunkOverlap = _defaultChunkOverlap
	}

	documents, err := loader.LoadAndSplit(context.Background(), split)
	if err != nil {
		return errors.New("Error loading documents:" + err.Error())
	}

	embedder, err = zhipuaiembeddings.NewZhiPuAI(
		zhipuaiembeddings.WithClient(*zhipuai.NewZhiPuAI(apiKey)),
	)
	if err != nil {
		return err
	}

	chroma, err := chromadb.New(
		chromadb.WithURL(url),
		chromadb.WithEmbedder(embedder),
		chromadb.WithNameSpace(namespace),
	)
	if err != nil {
		return err
	}

	return chroma.AddDocuments(ctx, documents)
}
