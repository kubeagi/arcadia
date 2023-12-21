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
	Blocking  ResponseMode = "blocking"
	Streaming ResponseMode = "streaming"
	// todo isFlowValidForStream only some node(llm chain) support streaming
)

type APPMetadata struct {
	APPName      string `json:"app_name" binding:"required"`
	AppNamespace string `json:"app_namespace" binding:"required"`
}

type ConversionReqBody struct {
	APPMetadata  `json:",inline"`
	ConversionID string `json:"conversion_id"`
}

type MessageReqBody struct {
	ConversionReqBody `json:",inline"`
	MessageID         string `json:"message_id"`
}

type ChatReqBody struct {
	Query             string       `json:"query" binding:"required"`
	ResponseMode      ResponseMode `json:"response_mode" binding:"required"`
	ConversionReqBody `json:",inline"`
	Debug             bool `json:"-"`
}

type ChatRespBody struct {
	ConversionID string                `json:"conversion_id"`
	MessageID    string                `json:"message_id"`
	Message      string                `json:"message"`
	CreatedAt    time.Time             `json:"created_at"`
	References   []retriever.Reference `json:"references,omitempty"`
}

type Conversion struct {
	ID          string                     `json:"id"`
	AppName     string                     `json:"app_name"`
	AppNamespce string                     `json:"app_namespace"`
	StartedAt   time.Time                  `json:"started_at"`
	UpdatedAt   time.Time                  `json:"updated_at"`
	Messages    []Message                  `json:"messages"`
	History     *memory.ChatMessageHistory `json:"-"`
	User        string                     `json:"-"`
	Debug       bool                       `json:"-"`
}

type Message struct {
	ID         string                `json:"id"`
	Query      string                `json:"query"`
	Answer     string                `json:"answer"`
	References []retriever.Reference `json:"references,omitempty"`
}
