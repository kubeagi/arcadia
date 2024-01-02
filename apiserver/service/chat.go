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

package service

import (
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"github.com/kubeagi/arcadia/apiserver/config"
	"github.com/kubeagi/arcadia/apiserver/pkg/auth"
	"github.com/kubeagi/arcadia/apiserver/pkg/chat"
	"github.com/kubeagi/arcadia/apiserver/pkg/oidc"
)

const (
	// Time interval to check if the chat stream should be closed if no more message arrives
	WaitTimeoutForChatStreaming = 5
)

// @BasePath /chat

// @Summary chat with application
// @Schemes
// @Description chat with application
// @Tags application
// @Accept json
// @Produce json
// @Param debug    query  bool              false  "Should the chat request be treated as debugging?"
// @Param request  body   chat.ChatReqBody  true   "query params"
// @Success 200 {object} chat.ChatRespBody "blocking mode, will return all field; streaming mode, only conversation_id, message and created_at will be returned"
// @Failure 400 {object} chat.ErrorResp
// @Failure 500 {object} chat.ErrorResp
// @Router / [post]
func chatHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		req := chat.ChatReqBody{}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, chat.ErrorResp{Err: err.Error()})
			return
		}
		req.Debug = c.Query("debug") == "true"
		stream := req.ResponseMode == chat.Streaming
		var response *chat.ChatRespBody
		var err error

		if stream {
			buf := strings.Builder{}
			// handle chat streaming mode
			respStream := make(chan string, 1)
			go func() {
				defer func() {
					if err := recover(); err != nil {
						klog.Errorln("An error occurred when run chat.AppRun: %s", err)
					}
				}()
				response, err = chat.AppRun(c, req, respStream)
				if response.Message == buf.String() {
					close(respStream)
				}
			}()

			// Use a ticker to check if there is no data arrived and close the stream
			// TODO: check if any better solution for this?
			hasData := true
			ticker := time.NewTicker(WaitTimeoutForChatStreaming * time.Second)
			quit := make(chan struct{})
			defer close(quit)
			go func() {
				for {
					select {
					case <-ticker.C:
						// If there is no generated data within WaitTimeoutForChatStreaming seconds, just close it
						if !hasData {
							close(respStream)
						}
						hasData = false
					case <-quit:
						ticker.Stop()
						return
					}
				}
			}()
			c.Writer.Header().Set("Content-Type", "text/event-stream")
			c.Writer.Header().Set("Cache-Control", "no-cache")
			c.Writer.Header().Set("Connection", "keep-alive")
			c.Writer.Header().Set("Transfer-Encoding", "chunked")
			klog.Infoln("start to receive messages...")
			clientDisconnected := c.Stream(func(w io.Writer) bool {
				if msg, ok := <-respStream; ok {
					c.SSEvent("", chat.ChatRespBody{
						ConversationID: req.ConversationID,
						Message:        msg,
						CreatedAt:      time.Now(),
					})
					hasData = true
					buf.WriteString(msg)
					return true
				}
				return false
			})
			if clientDisconnected {
				klog.Infoln("chatHandler: client is disconnected")
			}
			klog.Infoln("end to receive messages.")
		} else {
			// handle chat blocking mode
			response, err = chat.AppRun(c, req, nil)
			if err != nil {
				c.JSON(http.StatusInternalServerError, chat.ErrorResp{Err: err.Error()})
				klog.Infof("error resp: %v", err)
				return
			}
			c.JSON(http.StatusOK, response)
		}
	}
}

// @Summary list all conversations
// @Schemes
// @Description list all conversations
// @Tags application
// @Accept json
// @Produce json
// @Param request  body   chat.APPMetadata  true   "query params"
// @Success 200 {object} []chat.Conversation
// @Failure 400 {object} chat.ErrorResp
// @Failure 500 {object} chat.ErrorResp
// @Router /conversations [post]
func listConversationHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		req := chat.APPMetadata{}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, chat.ErrorResp{Err: err.Error()})
			return
		}
		resp, err := chat.ListConversations(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, chat.ErrorResp{Err: err.Error()})
			klog.Infof("error resp: %v", err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

// @Summary delete one conversation
// @Schemes
// @Description delete one conversation
// @Tags application
// @Accept json
// @Produce json
// @Param conversationID path string true "conversationID"
// @Success 200 {object} chat.SimpleResp
// @Failure 400 {object} chat.ErrorResp
// @Failure 500 {object} chat.ErrorResp
// @Router /conversations/:conversationID [delete]
func deleteConversationHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		conversationID := c.Param("conversationID")
		if conversationID == "" {
			c.JSON(http.StatusBadRequest, chat.ErrorResp{Err: "conversationID is required"})
			return
		}
		err := chat.DeleteConversation(c, conversationID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, chat.ErrorResp{Err: err.Error()})
			klog.Infof("error resp: %v", err)
			return
		}
		c.JSON(http.StatusOK, chat.SimpleResp{Message: "ok"})
	}
}

// @Summary get all messages history for one conversation
// @Schemes
// @Description get all messages history for one conversation
// @Tags application
// @Accept json
// @Produce json
// @Param request  body   chat.ConversationReqBody  true   "query params"
// @Success 200 {object} chat.Conversation
// @Failure 400 {object} chat.ErrorResp
// @Failure 500 {object} chat.ErrorResp
// @Router /messages [post]
func historyHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		req := chat.ConversationReqBody{}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, chat.ErrorResp{Err: err.Error()})
			return
		}
		resp, err := chat.ListMessages(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, chat.ErrorResp{Err: err.Error()})
			klog.Infof("error resp: %v", err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

// @Summary get one message references
// @Schemes
// @Description get one message's references
// @Tags application
// @Accept json
// @Produce json
// @Param messageID path  string               true   "messageID"
// @Param request   body  chat.MessageReqBody  true   "query params"
// @Success 200 {object} []retriever.Reference
// @Failure 400 {object} chat.ErrorResp
// @Failure 500 {object} chat.ErrorResp
// @Router /messages/:messageID/references [post]
func referenceHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		messageID := c.Param("messageID")
		if messageID == "" {
			c.JSON(http.StatusBadRequest, chat.ErrorResp{Err: "messageID is required"})
			return
		}
		req := chat.MessageReqBody{
			MessageID: messageID,
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, chat.ErrorResp{Err: err.Error()})
			return
		}
		resp, err := chat.GetMessageReferences(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, chat.ErrorResp{Err: err.Error()})
			klog.Infof("error resp: %v", err)
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

func registerChat(g *gin.RouterGroup, conf config.ServerConfig) {
	g.POST("", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "applications"), chatHandler()) // chat with bot

	g.POST("/conversations", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "applications"), listConversationHandler())                     // list conversations
	g.DELETE("/conversations/:conversationID", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "applications"), deleteConversationHandler()) // delete conversation

	g.POST("/messages", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "applications"), historyHandler())                         // messages history
	g.POST("/messages/:messageID/references", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "get", "applications"), referenceHandler()) // messages reference
}
