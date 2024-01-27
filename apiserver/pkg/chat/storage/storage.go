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

package storage

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/kubeagi/arcadia/pkg/appruntime/retriever"
)

var (
	ErrConversationNotFound = errors.New("conversation is not found")
)

// Conversation represent a conversation in storage
type Conversation struct {
	ID           string         `gorm:"column:id;primaryKey;type:uuid;comment:conversation id" json:"id" example:"5a41f3ca-763b-41ec-91c3-4bbbb00736d0"`
	AppName      string         `gorm:"column:app_name;type:string;comment:app name" json:"app_name" example:"chat-with-llm"`
	AppNamespace string         `gorm:"column:app_namespace;type:string;comment:app namespace" json:"app_namespace" example:"arcadia"`
	StartedAt    time.Time      `gorm:"column:started_at;type:time;autoCreateTime;comment:the time the conversation started at" json:"started_at" example:"2023-12-21T10:21:06.389359092+08:00"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;type:time;autoUpdateTime;comment:the time the conversation updated at" json:"updated_at" example:"2023-12-22T10:21:06.389359092+08:00"`
	Messages     []Message      `gorm:"foreignKey:ConversationID" json:"messages"`
	User         string         `gorm:"column:user;type:string;comment:the conversation chat user" json:"-"`
	Debug        bool           `gorm:"column:debug;type:bool;comment:debug mode" json:"-"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;type:time;comment:the time the conversation deleted at" json:"-"`
}

// Message represent a message in storage
type Message struct {
	ID             string     `gorm:"column:id;primaryKey;type:uuid;comment:message id" json:"id" example:"4f3546dd-5404-4bf8-a3bc-4fa3f9a7ba24"`
	Query          string     `gorm:"column:query;type:string;comment:user input" json:"query" example:"旷工最小计算单位为多少天？"`
	Answer         string     `gorm:"column:answer;type:string;comment:ai response" json:"answer" example:"旷工最小计算单位为0.5天。"`
	References     References `gorm:"column:references;type:json;comment:references" json:"references,omitempty"`
	ConversationID string     `gorm:"column:conversation_id;type:uuid;comment:conversation id" json:"-"`
	Latency        int64      `gorm:"column:latency;type:int;comment:request latency, in ms" json:"latency" example:"1000"`
}

type References []retriever.Reference

func (Conversation) TableName() string {
	return "app_chat_conversation"
}

func (Message) TableName() string {
	return "app_chat_message"
}

type Storage interface {
	ConversationStorage
	MessageStorage
}

// ConversationStorage interface
type ConversationStorage interface {
	// FindExistingConversation searches for an existing conversation by ConversationID.
	//
	// ConversationID string, opts ...SearchOption
	// *Conversation, error
	FindExistingConversation(ID string, opts ...SearchOption) (*Conversation, error)
	// Delete deletes a conversation with the given options.
	//
	// It takes variadic SearchOption parameter(s) and returns an error.
	// **not** return error if the conversation is not found
	Delete(opts ...SearchOption) error
	// UpdateConversation updates the Conversation.
	//
	// It takes a pointer to a Conversation and returns an error.
	UpdateConversation(*Conversation) error
	// ListConversations returns a list of conversations based on the provided options.
	//
	// It accepts SearchOption(s) and returns a slice of Conversation and an error.
	ListConversations(opts ...SearchOption) ([]Conversation, error)
}

type MessageStorage interface {
	// FindExistingMessage finds a message in the conversation.
	//
	// It takes ConversationID, messageID string parameters and returns *Message, error.
	FindExistingMessage(ConversationID, messageID string, opts ...SearchOption) (*Message, error)
	// CountMessages count how many messages is about this app
	CountMessages(appName, appNamespace string) (int64, error)
}
