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
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"github.com/kubeagi/arcadia/apiserver/config"
	"github.com/kubeagi/arcadia/apiserver/pkg/chat"
)

func chatHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		req := chat.ChatReqBody{}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		stream := req.ResponseMode == chat.Streaming
		var response *chat.ChatRespBody
		var err error

		if stream {
			// handle chat streaming mode
			respStream := make(chan string, 1)
			go func() {
				defer func() {
					if err := recover(); err != nil {
						klog.Errorln("An error occurred when run chat.AppRun: %s", err)
					}
				}()
				response, err = chat.AppRun(c, req, respStream)
			}()

			// Use a ticker to check if there is no data arrived and close the stream
			// TODO: check if any better solution for this?
			hasData := true
			ticker := time.NewTicker(5 * time.Second)
			quit := make(chan struct{})
			defer close(quit)
			go func() {
				for {
					select {
					case <-ticker.C:
						// If there is no generated data within 5 seconds, just close it
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
			klog.Infoln("start to receive message")
			clientDisconnected := c.Stream(func(w io.Writer) bool {
				if msg, ok := <-respStream; ok {
					c.SSEvent("", chat.ChatRespBody{
						ConversionID: req.ConversionID,
						Message:      msg,
						CreatedAt:    time.Now(),
					})
					hasData = true
					return true
				}
				return false
			})
			if clientDisconnected {
				klog.Infoln("chatHandler: client is disconnected")
			}
			klog.Infoln("end to receive message")
		} else {
			// handle chat blocking mode
			response, err = chat.AppRun(c, req, nil)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				klog.Infof("error resp: %v", err)
				return
			}
			c.JSON(http.StatusOK, response)
			return
		}
	}
}

func RegisterChat(g *gin.Engine, conf config.ServerConfig) {
	g.POST("/chat", chatHandler())
}
