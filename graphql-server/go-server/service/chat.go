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

	"github.com/gin-gonic/gin"

	"github.com/kubeagi/arcadia/graphql-server/go-server/config"
	"github.com/kubeagi/arcadia/graphql-server/go-server/pkg/chat"
)

func chatHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		req := chat.ChatReqBody{}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		stream := req.ResponseMode == chat.Streaming
		resp, respStreamChain, err := chat.AppRun(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !stream {
			c.JSON(http.StatusOK, resp)
			return
		}
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("Transfer-Encoding", "chunked")
		c.Stream(func(w io.Writer) bool {
			if msg, ok := <-respStreamChain; ok {
				c.SSEvent("", msg)
				return true
			}
			return false
		})
	}
}
func RegisteryChat(g *gin.Engine, conf config.ServerConfig) {
	g.POST("/chat", chatHandler())
}
