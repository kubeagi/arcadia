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

	"github.com/tmc/langchaingo/memory"

	"github.com/kubeagi/arcadia/pkg/application/retriever"
)

type ResponseMode string

const (
	// Blocking means the response is returned in a blocking manner
	Blocking ResponseMode = "blocking"
	// Streaming means the response will use Server-Sent Events
	Streaming ResponseMode = "streaming"
	// todo isFlowValidForStream only some node(llm chain) support streaming
)

type APPMetadata struct {
	// AppName, the name of the application
	APPName string `json:"app_name" binding:"required" example:"chat-with-llm"`
	// AppNamespace, the namespace of the application
	AppNamespace string `json:"app_namespace" binding:"required" example:"arcadia"`
}

type ConversationReqBody struct {
	APPMetadata `json:",inline"`
	// ConversationID, if it is empty, a new conversation will be created
	ConversationID string `json:"conversation_id" example:"5a41f3ca-763b-41ec-91c3-4bbbb00736d0"`
}

type MessageReqBody struct {
	ConversationReqBody `json:",inline"`
	// MessageID, single message id
	MessageID string `json:"message_id" example:"4f3546dd-5404-4bf8-a3bc-4fa3f9a7ba24"`
}

type ChatReqBody struct {
	// Query user query string
	Query string `json:"query" binding:"required" example:"旷工最小计算单位为多少天？"`
	// ResponseMode:
	// * Blocking - means the response is returned in a blocking manner
	// * Streaming - means the response will use Server-Sent Events
	ResponseMode        ResponseMode `json:"response_mode" binding:"required" example:"blocking"`
	ConversationReqBody `json:",inline"`
	Debug               bool `json:"-"`
}

type ChatRespBody struct {
	ConversationID string `json:"conversation_id" example:"5a41f3ca-763b-41ec-91c3-4bbbb00736d0"`
	MessageID      string `json:"message_id" example:"4f3546dd-5404-4bf8-a3bc-4fa3f9a7ba24"`
	// Message is what AI say
	Message string `json:"message" example:"旷工最小计算单位为0.5天。"`
	// CreatedAt is the time when the message is created
	CreatedAt time.Time `json:"created_at" example:"2023-12-21T10:21:06.389359092+08:00"`
	// References is the list of references
	References []retriever.Reference `json:"references,omitempty"`
}

type Conversation struct {
	ID          string                     `json:"id" example:"5a41f3ca-763b-41ec-91c3-4bbbb00736d0"`
	AppName     string                     `json:"app_name" example:"chat-with-llm"`
	AppNamespce string                     `json:"app_namespace" example:"arcadia"`
	StartedAt   time.Time                  `json:"started_at" example:"2023-12-21T10:21:06.389359092+08:00"`
	UpdatedAt   time.Time                  `json:"updated_at" example:"2023-12-22T10:21:06.389359092+08:00"`
	Messages    []Message                  `json:"messages"`
	History     *memory.ChatMessageHistory `json:"-"`
	User        string                     `json:"-"`
	Debug       bool                       `json:"-"`
}

type Message struct {
	ID         string                `json:"id" example:"4f3546dd-5404-4bf8-a3bc-4fa3f9a7ba24"`
	Query      string                `json:"query" example:"旷工最小计算单位为多少天？"`
	Answer     string                `json:"answer" example:"旷工最小计算单位为0.5天。"`
	References []retriever.Reference `json:"references,omitempty"`
}

type ErrorResp struct {
	Err string `json:"error" example:"conversation is not found"`
}

type SimpleResp struct {
	Message string `json:"message" example:"ok"`
}
