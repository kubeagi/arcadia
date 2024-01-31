/*
Copyright 2024 KubeAGI.

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

package chat

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"sync"
	"time"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/documentloaders"
	langchainllms "github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"k8s.io/klog/v2"

	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	runtimebase "github.com/kubeagi/arcadia/pkg/appruntime/base"
	runtimellm "github.com/kubeagi/arcadia/pkg/appruntime/llm"
	"github.com/kubeagi/arcadia/pkg/langchainwrap"
	"github.com/kubeagi/arcadia/pkg/utils"
	"github.com/kubeagi/arcadia/pkg/vectorstore"
)

var (
	ErrNoLLMProvidedInApplication = errors.New("llm not provided in application")
)

const (
	DefaultPromptTemplateForMap = `
		{{.context}}

		With above content, please summarize it with only half content size of it.
		`
	DefaultPromptTemplatForReduce       = `"{{.context}}"`
	DefaultSummaryMaxNumberOfConcurrent = 2

	DefaultDocumentChunkSize    = 2048
	DefaultDocumentChunkOverlap = 200
)

// ReceiveConversationDocs receive and process docs for a conversation
func (cs *ChatServer) ReceiveConversationDocs(ctx context.Context, req ConversationDocsReqBody, docs ...*multipart.FileHeader) (*ConversationDocsRespBody, error) {
	if req.ChunkSize == 0 {
		req.ChunkSize = DefaultDocumentChunkSize
	}
	if req.ChunkOverlap == 0 {
		req.ChunkOverlap = DefaultDocumentChunkOverlap
	}
	resps := make([]*ConversatioSingleDocRespBody, 0)
	for _, doc := range docs {
		// Upload the file to specific dst
		resp, err := cs.ReceiveConversationSingleDoc(ctx, req, doc)
		if err != nil {
			return nil, err
		}
		resps = append(resps, resp)
	}

	return &ConversationDocsRespBody{
		Docs: resps,
	}, nil
}

// ReceiveConversationSingleDoc receive a single document,then generate embeddings and summary for this document
func (cs *ChatServer) ReceiveConversationSingleDoc(ctx context.Context, req ConversationDocsReqBody, doc *multipart.FileHeader) (*ConversatioSingleDocRespBody, error) {
	klog.V(5).Infof("Load and split the document %s size:%s for conversation %s", doc.Filename, utils.BytesToSizedStr(doc.Size), req.ConversationID)
	src, err := doc.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}
	dataReader := bytes.NewReader(data)

	var documents []schema.Document
	var loader documentloaders.Loader
	switch ext := filepath.Ext(doc.Filename); ext {
	case ".pdf":
		loader = documentloaders.NewPDF(dataReader, doc.Size)
	case ".txt":
		loader = documentloaders.NewText(dataReader)
	case ".html", ".htm":
		loader = documentloaders.NewHTML(dataReader)
	default:
		return nil, fmt.Errorf("file with extension %s not supported yet", ext)
	}

	// TODO: expose the chunk parameters
	split := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(req.ChunkSize),
		textsplitter.WithChunkOverlap(req.ChunkOverlap),
	)
	documents, err = loader.LoadAndSplit(ctx, split)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 2)

	// For embedding generation
	wg.Add(1)
	var errEmbedding error
	var timecostForEmbedding float64
	go func() {
		start := time.Now()
		defer wg.Done()
		semaphore <- struct{}{} // 获取一个信号量
		defer func() {
			timecostForEmbedding = time.Since(start).Seconds()
			<-semaphore // 释放信号量
		}()
		klog.V(5).Infof("Generate embeddings from file %s to vectorstore for conversation %s", doc.Filename, req.ConversationID)
		errEmbedding = cs.GenerateSingleDocEmbeddings(ctx, req, doc, documents)
		if errEmbedding != nil {
			// break once error occurs
			ctx.Done()
		}
	}()

	// For summary generation
	wg.Add(1)
	var timecostForSummarization float64
	var errSummary error
	var summary string
	go func() {
		start := time.Now()
		defer wg.Done()
		semaphore <- struct{}{}
		defer func() {
			timecostForSummarization = time.Since(start).Seconds()
			<-semaphore
		}()
		klog.V(5).Infof("Generate summarization from file %s for conversation %s", doc.Filename, req.ConversationID)
		summary, errSummary = cs.GenerateSingleDocSummary(ctx, req, doc, documents)
		if errSummary != nil {
			// break once error occurs
			ctx.Done()
		}
	}()
	// wait until all finished
	wg.Wait()

	if errEmbedding != nil {
		return nil, errEmbedding
	}
	if errSummary != nil {
		return nil, errSummary
	}

	return &ConversatioSingleDocRespBody{
		FileName:                 doc.Filename,
		NumberOfDocuments:        len(documents),
		TimecostForEmbedding:     timecostForEmbedding,
		Summary:                  summary,
		TimecostForSummarization: timecostForSummarization,
	}, nil
}

// GenerateSingleDocEmbeddings
func (cs *ChatServer) GenerateSingleDocEmbeddings(ctx context.Context, req ConversationDocsReqBody, doc *multipart.FileHeader, documents []schema.Document) error {
	// get the built-in system embedder and vectorstore
	embedder, vectorStore, err := common.SystemEmbeddingSuite(ctx, cs.cli)
	if err != nil {
		return err
	}
	langchainEmbedder, err := langchainwrap.GetLangchainEmbedder(ctx, embedder, nil, cs.cli, "")
	if err != nil {
		return err
	}
	err = vectorstore.AddDocuments(ctx, klog.FromContext(ctx), vectorStore, langchainEmbedder, req.ConversationID, nil, cs.cli, documents)
	if err != nil {
		return err
	}
	return nil
}

// GenerateSingleDocSummary generate the summary of sinle document
func (cs *ChatServer) GenerateSingleDocSummary(ctx context.Context, req ConversationDocsReqBody, doc *multipart.FileHeader, documents []schema.Document) (string, error) {
	app, c, err := cs.getApp(ctx, req.APPName, req.AppNamespace)
	if err != nil {
		return "", fmt.Errorf("failed to get app due to %s", err.Error())
	}

	var llm langchainllms.LLM
	for _, n := range app.Spec.Nodes {
		baseNode := runtimebase.NewBaseNode(app.Namespace, n.Name, *n.Ref)
		if baseNode.Kind() == "llm" {
			l := runtimellm.NewLLM(baseNode)
			if err := l.Init(ctx, c, nil); err != nil {
				return "", fmt.Errorf("failed init llm due to %s", err.Error())
			}
			llm = l.LLM
		}
	}
	// If no LLM provided,we can't generate the summary
	if llm == nil {
		return "", ErrNoLLMProvidedInApplication
	}

	mpChain := chains.NewMapReduceDocuments(
		chains.NewLLMChain(llm, prompts.NewPromptTemplate(DefaultPromptTemplateForMap, []string{"context"})),
		chains.NewStuffDocuments(
			chains.NewLLMChain(
				llm,
				prompts.NewPromptTemplate(DefaultPromptTemplatForReduce, []string{"context"}),
			),
		),
	)

	// concurrent api call
	mpChain.MaxNumberOfConcurrent = DefaultSummaryMaxNumberOfConcurrent

	summary, err := chains.Run(ctx, mpChain, documents)
	if err != nil {
		return "", fmt.Errorf("failed to generate summary for %s due to %s", doc.Filename, err.Error())
	}

	return summary, nil
}
