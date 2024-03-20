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
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/kubeagi/arcadia/apiserver/config"
	"github.com/kubeagi/arcadia/apiserver/docs"
	"github.com/kubeagi/arcadia/apiserver/pkg/oidc"
)

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
		c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization, namespace, Referer, User-Agent")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		c.Next()
	}
}

func NewServerAndRun(conf config.ServerConfig) {
	r := gin.Default()
	r.Use(Cors())
	r.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// enable oidc authentication
	if conf.EnableOIDC {
		oidc.InitOIDCArgs(conf.IssuerURL, conf.MasterURL, conf.ClientSecret, conf.ClientID)
	}

	bffGroup := r.Group("/bff")
	// for file operations
	registerMinIOAPI(bffGroup, conf)
	// for ops apis with graphql
	registerGraphQL(r, bffGroup, conf)

	ragGroup := r.Group("/rags")
	registerRAG(ragGroup, conf)

	// for chat server with Restful apis
	chatGroup := r.Group("/chat")
	registerChat(chatGroup, conf)

	fg := r.Group("/forward")
	registerForward(fg, conf)

	//  for swagger
	if conf.EnableSwagger {
		docs.SwaggerInfo.Host = fmt.Sprintf("%s:%d", conf.Host, conf.Port)
		r.GET("swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	}

	_ = r.Run(fmt.Sprintf("%s:%d", conf.Host, conf.Port))
}
