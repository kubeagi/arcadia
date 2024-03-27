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
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeagi/arcadia/apiserver/config"
	"github.com/kubeagi/arcadia/apiserver/pkg/auth"
	"github.com/kubeagi/arcadia/apiserver/pkg/forwardrepo"
	"github.com/kubeagi/arcadia/apiserver/pkg/oidc"
)

const (
	queryModelID = "modelid"

	// huggingface or modelscope
	pathParamRepo = "repo"

	queryParamRevision = "revision"

	repoToken = "REPOTOKEN"

	proxyEnv = "APISERVER_PROXY"
)

type (
	// FrowarAPI is the forward api handler which forward requests to other services
	FrowarAPI   struct{}
	SummaryResp struct {
		Summary string `json:"summary"`
	}
)

func (f *FrowarAPI) Summary(ctx *gin.Context) {
	modelID := ctx.Query(queryModelID)
	revision := ctx.DefaultQuery(queryParamRevision, "main")
	repo := ctx.Param(pathParamRepo)
	token := ctx.GetHeader(repoToken)
	if modelID == "" {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "The modelid parameter is required",
		})
		return
	}

	tp := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	opts := []forwardrepo.Option{
		forwardrepo.WithHFToken(token), forwardrepo.WithModelID(modelID), forwardrepo.WithRevision(revision),
	}
	if repo == forwardrepo.HuggingFaceForward {
		if r := os.Getenv(proxyEnv); r != "" {
			u, _ := url.Parse(r)
			tp.Proxy = http.ProxyURL(u)
		}
		opts = append(opts, forwardrepo.WithTransport(tp))
	}
	instance, err := forwardrepo.NewForward(repo, opts...)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("Unsupported repository type %s", repo),
		})
		return
	}
	summary, err := instance.Summary(ctx.Request.Context())
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("failed to get model summary error: %s", err),
		})
		return
	}
	ctx.JSON(http.StatusOK, SummaryResp{
		Summary: summary,
	})
}

func (f *FrowarAPI) Revisions(ctx *gin.Context) {
	modelID := ctx.Query(queryModelID)
	repo := ctx.Param(pathParamRepo)
	token := ctx.GetHeader(repoToken)
	if modelID == "" {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": "The modelid parameter is required",
		})
		return
	}

	tp := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	opts := []forwardrepo.Option{
		forwardrepo.WithHFToken(token), forwardrepo.WithModelID(modelID),
	}
	if repo == forwardrepo.HuggingFaceForward {
		if r := os.Getenv(proxyEnv); r != "" {
			u, _ := url.Parse(r)
			tp.Proxy = http.ProxyURL(u)
		}
		opts = append(opts, forwardrepo.WithTransport(tp))
	}
	instance, err := forwardrepo.NewForward(repo, opts...)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("Unsupported repository type %s", repo),
		})
		return
	}
	revisions, err := instance.Revisions(ctx.Request.Context())
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("failed to get model revisions error: %s", err),
		})
		return
	}
	ctx.JSON(http.StatusOK, revisions)
}

func registerForward(g *gin.RouterGroup, conf config.ServerConfig) {
	api := FrowarAPI{}

	g.GET("/:repo/summary", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, schema.GroupVersion{}, "", ""), api.Summary)
	g.GET("/:repo/revisions", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, schema.GroupVersion{}, "", ""), api.Revisions)
}
