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
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/klog/v2"

	"github.com/kubeagi/arcadia/apiserver/pkg/auth"
	"github.com/kubeagi/arcadia/apiserver/pkg/chat/storage"
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

	DefaultDocumentChunkSize    = 1024
	DefaultDocumentChunkOverlap = 100
)

// ReceiveConversationDocs receive and process docs for a conversation
func (cs *ChatServer) ReceiveConversationDoc(ctx context.Context, messageID string, req ConversationDocsReqBody, doc *multipart.FileHeader, respStream chan string) (*ConversationDocsRespBody, error) {
	if messageID == "" {
		messageID = string(uuid.NewUUID())
	}

	var conversation *storage.Conversation
	var err error
	currentUser, _ := ctx.Value(auth.UserNameContextKey).(string)
	if !req.NewChat {
		search := []storage.SearchOption{
			storage.WithAppName(req.APPName),
			storage.WithAppNamespace(req.AppNamespace),
			storage.WithDebug(req.Debug),
		}
		if currentUser != "" {
			search = append(search, storage.WithUser(currentUser))
		}
		conversation, err = cs.Storage().FindExistingConversation(req.ConversationID, search...)
		if err != nil {
			return nil, err
		}
	} else {
		conversation = &storage.Conversation{
			ID:           req.ConversationID,
			AppName:      req.APPName,
			AppNamespace: req.AppNamespace,
			StartedAt:    req.StartTime,
			Messages:     make([]storage.Message, 0),
			User:         currentUser,
			Debug:        req.Debug,
		}
	}

	// process document with map-reduce
	message := storage.Message{
		ID:        messageID,
		Query:     req.Query,
		Answer:    "",
		Documents: make([]storage.Document, 0),
	}

	// summarize conversation doc
	resp, err := cs.SummarizeConversationDoc(ctx, req, doc, respStream)
	if err != nil {
		return nil, err
	}
	message.Answer = resp.Summary
	message.Latency = int64(resp.TimecostForSummarization)
	message.Documents = append(message.Documents, storage.Document{
		ID:        string(uuid.NewUUID()),
		MessageID: messageID,
		Name:      doc.Filename,
		Summary:   resp.Summary,
	})

	// update conversat ion
	conversation.Messages = append(conversation.Messages, message)
	conversation.UpdatedAt = req.StartTime
	// update the conversation with new message
	if err := cs.Storage().UpdateConversation(conversation); err != nil {
		return nil, err
	}

	return &ConversationDocsRespBody{
		ChatRespBody: ChatRespBody{
			ConversationID: req.ConversationID,
			MessageID:      messageID,
			CreatedAt:      time.Now(),
			Message:        resp.Summary,
		},
		Doc: resp,
	}, nil
}

// SummarizeConversationDoc receive a single document,then generate embeddings and summary for this document
func (cs *ChatServer) SummarizeConversationDoc(ctx context.Context, req ConversationDocsReqBody, doc *multipart.FileHeader, respStream chan string) (*ConversatioSingleDocRespBody, error) {
	klog.V(5).Infof("Load and split the document %s size:%s for conversation %s", doc.Filename, utils.BytesToSizedStr(doc.Size), req.ConversationID)
	resp := &ConversatioSingleDocRespBody{
		FileName: doc.Filename,
	}
	var summarizationStart = time.Now()
	defer func() {
		resp.TotalTimecost = time.Since(summarizationStart).Seconds()
	}()

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

	// set document chunk parameters
	if req.ChunkSize == 0 {
		req.ChunkSize = DefaultDocumentChunkSize
	}
	if req.ChunkOverlap == 0 {
		req.ChunkOverlap = DefaultDocumentChunkOverlap
	}

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

	var errStr string
	// For embedding generation
	wg.Add(1)
	var errEmbedding error
	go func() {
		start := time.Now()
		defer wg.Done()
		semaphore <- struct{}{}
		defer func() {
			resp.TimecostForEmbedding = time.Since(start).Seconds()
			<-semaphore
		}()
		klog.V(5).Infof("Generate embeddings from file %s to vectorstore for conversation %s", doc.Filename, req.ConversationID)
		errEmbedding = cs.GenerateSingleDocEmbeddings(ctx, req, doc, documents)
		if errEmbedding != nil {
			errStr += fmt.Sprintf(" ErrEmbedding: %s", errEmbedding.Error())
			// break once error occurs
			ctx.Done()
		}
	}()

	// For summary generation
	wg.Add(1)
	var errSummary error
	var summary string
	go func() {
		start := time.Now()
		defer wg.Done()
		semaphore <- struct{}{}
		defer func() {
			resp.TimecostForSummarization = time.Since(start).Seconds()
			<-semaphore
		}()
		klog.V(5).Infof("Generate summarization from file %s for conversation %s", doc.Filename, req.ConversationID)
		summary, errSummary = cs.GenerateSingleDocSummary(ctx, req, doc, documents, respStream)
		if errSummary != nil {
			// break once error occurs
			errStr += fmt.Sprintf(" ErrSummary: %s", errEmbedding.Error())
			ctx.Done()
		}
	}()
	// wait until all finished
	wg.Wait()

	if errEmbedding != nil || errSummary != nil {
		return nil, errors.New(errStr)
	}

	resp.NumberOfDocuments = len(documents)
	resp.Summary = summary

	return resp, nil
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
func (cs *ChatServer) GenerateSingleDocSummary(ctx context.Context, req ConversationDocsReqBody, doc *multipart.FileHeader, documents []schema.Document, respStream chan string) (string, error) {
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

	var summary string
	if req.ResponseMode.IsStreaming() {
		summary, err = chains.Run(ctx, mpChain, documents, chains.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			respStream <- string(chunk)
			return nil
		}))
	} else {
		summary, err = chains.Run(ctx, mpChain, documents)
	}
	if err != nil {
		return "", fmt.Errorf("failed to generate summary for %s due to %s", doc.Filename, err.Error())
	}

	return summary, nil
}
