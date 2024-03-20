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

package chat

import (
	"time"

	"github.com/kubeagi/arcadia/pkg/appruntime/retriever"
)

type ResponseMode string

func (r ResponseMode) IsStreaming() bool {
	return r == Streaming
}

const (
	// Blocking means the response is returned in a blocking manner
	Blocking ResponseMode = "blocking"
	// Streaming means the response will use Server-Sent Events
	Streaming ResponseMode = "streaming"
)

type APPMetadata struct {
	// AppName, the name of the application
	APPName string `json:"app_name" form:"app_name" binding:"required" example:"chat-with-llm"`
	// AppNamespace, the namespace of the application, will be forced to use the value of the namespace in the request header for security reasons and is placed here only for compatibility with older versions
	AppNamespace string `json:"app_namespace" form:"app_namespace" example:"kubeagi-system"`
}

type ConversationReqBody struct {
	APPMetadata `json:",inline" form:",inline"`
	// ConversationID, if it is empty, a new conversation will be created
	ConversationID string `json:"conversation_id" form:"conversation_id" example:"5a41f3ca-763b-41ec-91c3-4bbbb00736d0"`
}

type ConversationFilesReqBody struct {
	ConversationReqBody `json:",inline"`
	Debug               bool      `json:"-"`
	NewChat             bool      `json:"-"`
	StartTime           time.Time `json:"-"`
}

type MessageReqBody struct {
	ConversationReqBody `json:",inline"`
	// MessageID, single message id
	MessageID string `json:"message_id" example:"4f3546dd-5404-4bf8-a3bc-4fa3f9a7ba24"`
}

type ChatReqBody struct {
	// Query user query string
	Query string `json:"query" form:"query" binding:"required" example:"旷工最小计算单位为多少天？"`
	// Files this conversation will use in the context
	Files []string `json:"files" form:"files" example:"test.pdf,song.mp3"`
	// ResponseMode:
	// * Blocking - means the response is returned in a blocking manner
	// * Streaming - means the response will use Server-Sent Events
	ResponseMode        ResponseMode `json:"response_mode" form:"response_mode" binding:"required" example:"blocking"`
	ConversationReqBody `json:",inline"`
	Debug               bool      `json:"-"`
	NewChat             bool      `json:"-"`
	StartTime           time.Time `json:"-"`
}

type ChatRespBody struct {
	ConversationID string `json:"conversation_id" example:"5a41f3ca-763b-41ec-91c3-4bbbb00736d0"`
	MessageID      string `json:"message_id" example:"4f3546dd-5404-4bf8-a3bc-4fa3f9a7ba24"`
	// Action indicates what is this chat for
	Action string `json:"action,omitempty" example:"CHAT"`
	// Message is what AI say
	Message string `json:"message" example:"旷工最小计算单位为0.5天。"`
	// CreatedAt is the time when the message is created
	CreatedAt time.Time `json:"created_at" example:"2023-12-21T10:21:06.389359092+08:00"`
	// References is the list of references
	References []retriever.Reference `json:"references,omitempty"`
	// Latency(ms) is how much time the server cost to process a certain request.
	Latency int64 `json:"latency,omitempty" example:"1000"`
	// Documents in this chat
	Document DocumentRespBody `json:"document,omitempty"`
}

type DocumentRespBody struct {
	ID     string `json:"id,omitempty" example:"8b833028-5d8d-418c-9f28-8aaa23c972b0"`
	Name   string `json:"name,omitempty" example:"example.pdf"`
	Object string `json:"object,omitempty" example:"application/base-chat-document-assistant/conversation/f54f5122-28fb-474e-8593-39f5b3760eaa/90fe100fb9ee6e6cb9ccb091fd91b6264f5f5444dea6d93b3c8ed7418e20c37d.pdf"`
}

type ErrorResp struct {
	Err string `json:"error" example:"conversation is not found"`
}

type SimpleResp struct {
	Message string `json:"message" example:"ok"`
}

// ConversationDocsRespBody is the response body for a ConversationDocsReq
type ConversationDocsRespBody struct {
	ChatRespBody `json:",inline"`
	// Docs are the responbody for each document
	Doc *ConversatioSingleDocRespBody `json:"doc,omitempty"`
}

// ConversatioSingleDocRespBody is the response body for a single conversation doc
type ConversatioSingleDocRespBody struct {
	FileName          string  `json:"file_name,omitempty"`
	NumberOfDocuments int     `json:"number_of_documents,omitempty"`
	TotalTimecost     float64 `json:"total_time_cost,omitempty"`
	// Embedding info
	TimecostForEmbedding float64 `json:"timecost_for_embedding,omitempty"`
	// Summary info
	Summary                  string  `json:"summary,omitempty"`
	TimecostForSummarization float64 `json:"timecost_for_summarization,omitempty"`
}
