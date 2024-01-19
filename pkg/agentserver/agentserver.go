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

package agentserver

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kubeagi/arcadia/apiserver/docs"
	"github.com/kubeagi/arcadia/apiserver/pkg/oidc"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"k8s.io/klog/v2"
)

var server = &Server{}

// Config for agentserver
type Server struct {
	// Basic info
	Host string
	Port int

	// OIDC auth info
	OIDC `json:"inline,"`

	// EnableSwagger enables the swagger docs when it is true
	EnableSwagger bool

	// Namespace(KubeAGI managed) which this agentserver works on
	Namespace string

	//Postgresql which this agentserver uses
	Postgresql string
}

type OIDC struct {
	Enable                                       bool
	IssuerURL, MasterURL, ClientID, ClientSecret string
}

func GetServer() Server {
	return *server
}

func initFlags() {
	// basic configurations
	flag.StringVar(&server.Host, "host", "", "bind to the host, default is 0.0.0.0")
	flag.IntVar(&server.Port, "port", 8081, "service listening port")
	// OIDC configurations
	flag.BoolVar(&server.OIDC.Enable, "enable-oidc", false, "enable oidc authorization")
	flag.StringVar(&server.OIDC.IssuerURL, "issuer-url", "", "oidc issuer url(required when enable odic)")
	flag.StringVar(&server.OIDC.MasterURL, "master-url", "", "k8s master url(required when enable odic)")
	flag.StringVar(&server.OIDC.ClientID, "client-id", "", "oidc client id(required when enable odic)")
	flag.StringVar(&server.OIDC.ClientSecret, "client-secret", "", "oidc client secret(required when enable odic)")

	// swagger doc
	flag.BoolVar(&server.EnableSwagger, "enable-swagger", true, "enable the swagger doc")

	klog.InitFlags(nil)
	flag.Parse()
}

func Run() {
	initFlags()

	r := gin.Default()
	r.Use(cors())
	r.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	if server.OIDC.Enable {
		oidc.InitOIDCArgs(server.OIDC.IssuerURL, server.OIDC.MasterURL, server.OIDC.ClientSecret, server.OIDC.ClientID)
	}

	if server.EnableSwagger {
		docs.SwaggerInfo.BasePath = "/"
		r.GET("swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	}

	_ = r.Run(fmt.Sprintf("%s:%d", server.Host, server.Port))
}

func cors() gin.HandlerFunc {
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
