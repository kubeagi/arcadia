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

package service

import (
	"github.com/gin-gonic/gin"

	"github.com/kubeagi/arcadia/apiserver/config"
	"github.com/kubeagi/arcadia/apiserver/pkg/auth"
	"github.com/kubeagi/arcadia/apiserver/pkg/client"
	"github.com/kubeagi/arcadia/apiserver/pkg/oidc"
	"github.com/kubeagi/arcadia/apiserver/pkg/requestid"
)

func registerGptsChat(g *gin.RouterGroup, conf config.ServerConfig) {
	c, err := client.GetClient(nil)
	if err != nil {
		panic(err)
	}

	chatService, err := NewChatService(c, true)
	if err != nil {
		panic(err)
	}

	g.POST("", auth.AuthTokenIsValid(conf.EnableOIDC, oidc.Verifier), requestid.RequestIDInterceptor(), chatService.ChatHandler()) // chat with bot

	g.POST("/conversations/file", auth.AuthTokenIsValid(conf.EnableOIDC, oidc.Verifier), requestid.RequestIDInterceptor(), chatService.ChatFile())                               // upload fles for conversation
	g.POST("/conversations", auth.AuthTokenIsValid(conf.EnableOIDC, oidc.Verifier), requestid.RequestIDInterceptor(), chatService.ListConversationHandler())                     // list conversations
	g.DELETE("/conversations/:conversationID", auth.AuthTokenIsValid(conf.EnableOIDC, oidc.Verifier), requestid.RequestIDInterceptor(), chatService.DeleteConversationHandler()) // delete conversation

	g.POST("/messages", auth.AuthTokenIsValid(conf.EnableOIDC, oidc.Verifier), requestid.RequestIDInterceptor(), chatService.HistoryHandler())                         // messages history
	g.POST("/messages/:messageID/references", auth.AuthTokenIsValid(conf.EnableOIDC, oidc.Verifier), requestid.RequestIDInterceptor(), chatService.ReferenceHandler()) // messages reference

	g.POST("/prompt-starter", auth.AuthTokenIsValid(conf.EnableOIDC, oidc.Verifier), requestid.RequestIDInterceptor(), chatService.PromptStartersHandler())
}
