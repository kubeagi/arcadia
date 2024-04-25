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
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/config"
	"github.com/kubeagi/arcadia/apiserver/pkg/auth"
	"github.com/kubeagi/arcadia/apiserver/pkg/chat"
	"github.com/kubeagi/arcadia/apiserver/pkg/client"
	"github.com/kubeagi/arcadia/apiserver/pkg/oidc"
	"github.com/kubeagi/arcadia/apiserver/pkg/requestid"
)

const (
	// Time interval to check if the chat stream should be closed if no more message arrives
	WaitTimeoutForChatStreaming = 120
	// default prompt starter
	PromptLimit = 4
)

type ChatService struct {
	server *chat.ChatServer
}

func NewChatService(cli runtimeclient.Client, isGpts bool) (*ChatService, error) {
	return &ChatService{chat.NewChatServer(cli, isGpts)}, nil
}

// @Summary	chat with application
// @Schemes
// @Description	chat with application
// @Tags			application
// @Accept			json
// @Produce		json
// @Param			namespace	header		string				true	"namespace this request is in"
// @Param			debug		query		bool				false	"Should the chat request be treated as debugging?"
// @Param			request		body		chat.ChatReqBody	true	"query params"
// @Success		200			{object}	chat.ChatRespBody	"blocking mode, will return all field; streaming mode, only conversation_id, message and created_at will be returned"
// @Failure		400			{object}	chat.ErrorResp
// @Failure		500			{object}	chat.ErrorResp
// @Router			/chat [post]
func (cs *ChatService) ChatHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		req := chat.ChatReqBody{StartTime: time.Now()}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, chat.ErrorResp{Err: err.Error()})
			return
		}
		req.AppNamespace = NamespaceInHeader(c)
		req.Debug = c.Query("debug") == "true"
		req.NewChat = len(req.ConversationID) == 0
		if req.NewChat {
			req.ConversationID = string(uuid.NewUUID())
		}
		messageID := string(uuid.NewUUID())
		var response *chat.ChatRespBody
		var err error
		logger := klog.FromContext(c.Request.Context())
		chatTimeoutSecond := pointer.Float64(WaitTimeoutForChatStreaming)

		if req.ResponseMode.IsStreaming() {
			buf := strings.Builder{}
			// handle chat streaming mode
			respStream := make(chan string, 1)
			manualStop := make(chan bool)
			go func() {
				defer func() {
					if e := recover(); e != nil {
						err, ok := e.(error)
						if ok {
							logger.Error(err, "A panic occurred when run chat.AppRun")
						} else {
							logger.Error(fmt.Errorf("get err:%#v", e), "A panic occurred when run chat.AppRun")
						}
					}
				}()
				response, err = cs.server.AppRun(c.Request.Context(), req, respStream, messageID, chatTimeoutSecond)
				if err != nil {
					c.SSEvent("error", chat.ChatRespBody{
						MessageID:      messageID,
						ConversationID: req.ConversationID,
						Message:        err.Error(),
						CreatedAt:      time.Now(),
						Latency:        time.Since(req.StartTime).Milliseconds(),
					})
					// c.JSON(http.StatusInternalServerError, chat.ErrorResp{Err: err.Error()})
					logger.Error(err, "error resp, stop the stream")
					manualStop <- true
					return
				}
				if response != nil {
					if str := buf.String(); response.Message == str || strings.TrimSpace(str) == strings.TrimSpace(response.Message) {
						logger.Info("blocking resp is same with streaming resp, no new message received, stop the stream")
						manualStop <- true
					}
				}
			}()

			LatestTimestampGetDataFromLLM := time.Now()
			// Use ticker to check if there is no data from llm for a long time and close the entire stream
			ticker := time.NewTicker(time.Microsecond * 500)
			defer ticker.Stop()
			go func() {
				for range ticker.C {
					timeout := time.Second * time.Duration(*chatTimeoutSecond)
					if time.Since(LatestTimestampGetDataFromLLM) > timeout {
						logger.Info("no data from LLM for a long time, stop the stream", "timeout", timeout)
						manualStop <- true
						return
					}
				}
			}()
			c.Writer.Header().Set("Content-Type", "text/event-stream")
			c.Writer.Header().Set("Cache-Control", "no-cache")
			c.Writer.Header().Set("Connection", "keep-alive")
			c.Writer.Header().Set("Transfer-Encoding", "chunked")
			logger.Info("start to receive messages...")
			clientDisconnected := c.Stream(func(w io.Writer) bool {
				for {
					select {
					case <-manualStop:
						return false
					case msg, ok := <-respStream:
						if !ok {
							return false
						}
						t := time.Now()
						c.SSEvent("", chat.ChatRespBody{
							MessageID:      messageID,
							ConversationID: req.ConversationID,
							Message:        msg,
							CreatedAt:      t,
							Latency:        time.Since(req.StartTime).Milliseconds(),
						})
						LatestTimestampGetDataFromLLM = t
						buf.WriteString(msg)
						return true
					}
				}
			})
			if clientDisconnected {
				manualStop <- true
				logger.Info("chatHandler: the client is disconnected")
			}
			logger.Info("end to receive messages")
		} else {
			// handle chat blocking mode
			response, err = cs.server.AppRun(c.Request.Context(), req, nil, messageID, chatTimeoutSecond)
			if err != nil {
				c.JSON(http.StatusInternalServerError, chat.ErrorResp{Err: err.Error()})
				logger.Error(err, "error resp")
				return
			}
			c.JSON(http.StatusOK, response)
		}
		switch {
		case logger.V(5).Enabled():
			logger.Info("chat done", "req", req, "resp", response)
		case logger.V(3).Enabled():
			logger.Info("chat done", "req", req)
		default:
			logger.Info("chat done")
		}
	}
}

// @Summary	receive conversational files for one conversation
// @Schemes
// @Description	receive conversational files for one conversation
// @Tags			application
// @Accept			multipart/form-data
// @Produce		json
//
// @Param			namespace		header		string	true	"namespace this request is in"
// @Param			app_name		formData	string	true	"The app name for this conversation"
// @Param			conversation_id	formData	string	false	"The conversation id for this file"
// @Param			file			formData	file	true	"This is the file for the conversation"
//
// @Success		200				{object}	chat.ChatRespBody
// @Failure		400				{object}	chat.ErrorResp
// @Failure		500				{object}	chat.ErrorResp
// @Router			/chat/conversations/file [post]
func (cs *ChatService) ChatFile() gin.HandlerFunc {
	return func(c *gin.Context) {
		req := chat.ConversationFilesReqBody{
			StartTime: time.Now(),
		}
		if err := c.ShouldBind(&req); err != nil {
			klog.FromContext(c.Request.Context()).Error(err, "conversationFileHandler: error binding json")
			c.JSON(http.StatusBadRequest, chat.ErrorResp{Err: err.Error()})
			return
		}
		req.AppNamespace = NamespaceInHeader(c)
		app, err := cs.server.GetApp(c.Request.Context(), req.APPName, req.AppNamespace)
		if err != nil {
			klog.FromContext(c.Request.Context()).Error(err, "conversationFileHandler: error get app")
			c.JSON(http.StatusBadRequest, chat.ErrorResp{Err: err.Error()})
			return
		}
		if !pointer.BoolDeref(app.Spec.EnableUploadFile, true) {
			c.JSON(http.StatusForbidden, chat.ErrorResp{Err: "file upload is not enabled"})
			return
		}
		req.Debug = c.Query("debug") == "true"
		// check if this is a new chat
		req.NewChat = len(req.ConversationID) == 0
		if req.NewChat {
			req.ConversationID = string(uuid.NewUUID())
		}

		file, err := c.FormFile("file")
		if err != nil {
			klog.FromContext(c.Request.Context()).Error(err, "error receive conversational file")
			c.JSON(http.StatusBadRequest, chat.ErrorResp{Err: err.Error()})
			return
		}

		messageID := string(uuid.NewUUID())
		// Upload the file to specific dst.
		resp, err := cs.server.ReceiveConversationFile(c.Request.Context(), messageID, req, file)
		if err != nil {
			klog.FromContext(c.Request.Context()).Error(err, "error receive conversational file")
			c.JSON(http.StatusInternalServerError, chat.ErrorResp{Err: err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
		klog.FromContext(c.Request.Context()).V(3).Info("receive conversational file done", "req", req)
	}
}

// @Summary	list all conversations
// @Schemes
// @Description	list all conversations
// @Tags			application
// @Accept			json
// @Produce		json
// @Param			namespace	header		string				true	"namespace this request is in"
// @Param			request		body		chat.APPMetadata	false	"query params, if not set will return all current user's conversations"
// @Success		200			{object}	[]storage.Conversation
// @Failure		400			{object}	chat.ErrorResp
// @Failure		500			{object}	chat.ErrorResp
// @Router			/chat/conversations [post]
func (cs *ChatService) ListConversationHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		req := chat.APPMetadata{}
		_ = c.ShouldBindJSON(&req)
		req.AppNamespace = NamespaceInHeader(c)
		resp, err := cs.server.ListConversations(c.Request.Context(), req)
		if err != nil {
			klog.FromContext(c.Request.Context()).Error(err, "error list conversation")
			c.JSON(http.StatusInternalServerError, chat.ErrorResp{Err: err.Error()})
			return
		}
		if err := cs.server.FillAppIconToConversations(c.Request.Context(), &resp); err != nil {
			// note: fill app icon is try our best, don't need to return error
			klog.FromContext(c.Request.Context()).Error(err, "error fill app icon to conversations")
		}
		klog.FromContext(c.Request.Context()).V(3).Info("list conversation done", "req", req)
		c.JSON(http.StatusOK, resp)
	}
}

// @Summary	delete one conversation
// @Schemes
// @Description	delete one conversation
// @Tags			application
// @Accept			json
// @Produce		json
// @Param			conversationID	path		string	true	"conversationID"
// @Success		200				{object}	chat.SimpleResp
// @Failure		400				{object}	chat.ErrorResp
// @Failure		500				{object}	chat.ErrorResp
// @Router			/chat/conversations/{conversationID} [delete]
func (cs *ChatService) DeleteConversationHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		conversationID := c.Param("conversationID")
		if conversationID == "" {
			err := errors.New("conversationID is required")
			klog.FromContext(c.Request.Context()).Error(err, "conversationID is required")
			c.JSON(http.StatusBadRequest, chat.ErrorResp{Err: err.Error()})
			return
		}
		err := cs.server.DeleteConversation(c.Request.Context(), conversationID)
		if err != nil {
			klog.FromContext(c.Request.Context()).Error(err, "error delete conversation")
			c.JSON(http.StatusInternalServerError, chat.ErrorResp{Err: err.Error()})
			return
		}
		klog.FromContext(c.Request.Context()).V(3).Info("delete conversation done", "conversationID", conversationID)
		c.JSON(http.StatusOK, chat.SimpleResp{Message: "ok"})
	}
}

// @Summary	get all messages history for one conversation
// @Schemes
// @Description	get all messages history for one conversation
// @Tags			application
// @Accept			json
// @Produce		json
// @Param			namespace	header		string						true	"namespace this request is in"
// @Param			request		body		chat.ConversationReqBody	true	"query params"
// @Success		200			{object}	storage.Conversation
// @Failure		400			{object}	chat.ErrorResp
// @Failure		500			{object}	chat.ErrorResp
// @Router			/chat/messages [post]
func (cs *ChatService) HistoryHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		req := chat.ConversationReqBody{}
		if err := c.ShouldBindJSON(&req); err != nil {
			klog.FromContext(c.Request.Context()).Error(err, "historyHandler: error binding json")
			c.JSON(http.StatusBadRequest, chat.ErrorResp{Err: err.Error()})
			return
		}
		req.AppNamespace = NamespaceInHeader(c)
		resp, err := cs.server.ListMessages(c.Request.Context(), req)
		if err != nil {
			klog.FromContext(c.Request.Context()).Error(err, "error list messages")
			c.JSON(http.StatusInternalServerError, chat.ErrorResp{Err: err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
		klog.FromContext(c.Request.Context()).V(3).Info("get message history done", "req", req)
	}
}

// @Summary	get one message references
// @Schemes
// @Description	get one message's references
// @Tags			application
// @Accept			json
// @Produce		json
// @Param			namespace	header		string				true	"namespace this request is in"
// @Param			messageID	path		string				true	"messageID"
// @Param			request		body		chat.MessageReqBody	true	"query params"
// @Success		200			{object}	[]retriever.Reference
// @Failure		400			{object}	chat.ErrorResp
// @Failure		500			{object}	chat.ErrorResp
// @Router			/chat/messages/{messageID}/references [post]
func (cs *ChatService) ReferenceHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		messageID := c.Param("messageID")
		if messageID == "" {
			err := errors.New("messageID is required")
			klog.FromContext(c.Request.Context()).Error(err, "messageID is required")
			c.JSON(http.StatusBadRequest, chat.ErrorResp{Err: err.Error()})
			return
		}
		req := chat.MessageReqBody{
			MessageID: messageID,
		}
		req.AppNamespace = NamespaceInHeader(c)
		if err := c.ShouldBindJSON(&req); err != nil {
			klog.FromContext(c.Request.Context()).Error(err, "referenceHandler: error binding json")
			c.JSON(http.StatusBadRequest, chat.ErrorResp{Err: err.Error()})
			return
		}
		resp, err := cs.server.GetMessageReferences(c.Request.Context(), req)
		if err != nil {
			klog.FromContext(c.Request.Context()).Error(err, "error get message references")
			c.JSON(http.StatusInternalServerError, chat.ErrorResp{Err: err.Error()})
			return
		}
		klog.FromContext(c.Request.Context()).V(3).Info("get message references done", "req", req)
		c.JSON(http.StatusOK, resp)
	}
}

// @Summary	get app's prompt starters
// @Schemes
// @Description	get app's prompt starters
// @Tags			application
// @Accept			json
// @Produce		json
// @Param			namespace	header		string				true	"namespace this request is in"
// @Param			limit		query		int					false	"how many prompts you need should > 0 and < 10"
// @Param			request		body		chat.APPMetadata	true	"query params"
// @Success		200			{object}	[]string
// @Failure		400			{object}	chat.ErrorResp
// @Failure		500			{object}	chat.ErrorResp
// @Router			/chat/prompt-starter [post]
func (cs *ChatService) PromptStartersHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		req := chat.APPMetadata{}
		if err := c.ShouldBindJSON(&req); err != nil {
			klog.FromContext(c.Request.Context()).Error(err, "PromptStartersHandler: error binding json")
			c.JSON(http.StatusBadRequest, chat.ErrorResp{Err: err.Error()})
			return
		}
		req.AppNamespace = NamespaceInHeader(c)
		limit := c.Query("limit")
		limitVal := PromptLimit
		if limit != "" {
			var err error
			limitVal, err = strconv.Atoi(limit)
			if err != nil {
				c.JSON(http.StatusBadRequest, chat.ErrorResp{Err: err.Error()})
				return
			}
			if limitVal > 10 || limitVal < 1 {
				limitVal = PromptLimit
			}
		}
		resp, err := cs.server.ListPromptStarters(c.Request.Context(), req, limitVal)
		if err != nil {
			klog.FromContext(c.Request.Context()).Error(err, "error get Prompt Starters")
			c.JSON(http.StatusInternalServerError, chat.ErrorResp{Err: err.Error()})
			return
		}
		klog.FromContext(c.Request.Context()).V(3).Info("get Prompt Starters done", "req", req)
		c.JSON(http.StatusOK, resp)
	}
}

func NamespaceInHeader(c *gin.Context) string {
	return c.GetHeader(namespaceHeader)
}

func registerChat(g *gin.RouterGroup, conf config.ServerConfig) {
	c, err := client.GetClient(nil)
	if err != nil {
		panic(err)
	}

	chatService, err := NewChatService(c, false)
	if err != nil {
		panic(err)
	}

	g.POST("", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "applications"), requestid.RequestIDInterceptor(), chatService.ChatHandler()) // chat with bot

	g.POST("/conversations/file", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "applications"), requestid.RequestIDInterceptor(), chatService.ChatFile())                               // upload fles for conversation
	g.POST("/conversations", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "applications"), requestid.RequestIDInterceptor(), chatService.ListConversationHandler())                     // list conversations
	g.DELETE("/conversations/:conversationID", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "applications"), requestid.RequestIDInterceptor(), chatService.DeleteConversationHandler()) // delete conversation

	g.POST("/messages", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "applications"), requestid.RequestIDInterceptor(), chatService.HistoryHandler())                         // messages history
	g.POST("/messages/:messageID/references", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "applications"), requestid.RequestIDInterceptor(), chatService.ReferenceHandler()) // messages reference

	g.POST("/prompt-starter", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "applications"), requestid.RequestIDInterceptor(), chatService.PromptStartersHandler())
}
