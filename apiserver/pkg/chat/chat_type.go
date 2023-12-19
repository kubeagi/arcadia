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
)

type ResponseMode string

const (
	Blocking  ResponseMode = "blocking"
	Streaming ResponseMode = "streaming"
	// todo isFlowValidForStream only some node(llm chain) support streaming
)

type ChatReqBody struct {
	Query        string       `json:"query" binding:"required"`
	ResponseMode ResponseMode `json:"response_mode" binding:"required"`
	ConversionID string       `json:"conversion_id"`
	APPName      string       `json:"app_name" binding:"required"`
	AppNamespace string       `json:"app_namespace" binding:"required"`
}

type ChatRespBody struct {
	ConversionID string    `json:"conversion_id"`
	MessageID    string    `json:"message_id"`
	Message      string    `json:"message"`
	CreatedAt    time.Time `json:"created_at"`
}

type Conversion struct {
	ID          string    `json:"id"`
	AppName     string    `json:"app_name"`
	AppNamespce string    `json:"app_namespace"`
	StartedAt   time.Time `json:"started_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Messages    []Message `json:"messages"`
	History     *memory.ChatMessageHistory
}

type Message struct {
	ID     string `json:"id"`
	Query  string `json:"query"`
	Answer string `json:"answer"`
}
