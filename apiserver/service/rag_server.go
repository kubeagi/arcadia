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
	"strconv"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/evaluation/v1alpha1"
	gqlconfig "github.com/kubeagi/arcadia/apiserver/config"
	"github.com/kubeagi/arcadia/apiserver/pkg/auth"
	"github.com/kubeagi/arcadia/apiserver/pkg/oidc"
	"github.com/kubeagi/arcadia/apiserver/pkg/rag"
)

type RagAPI struct {
	c client.Client
}

const (
	ragNameQuery = "ragName"
	appNameQuery = "appName"
)

// @Summary	Get a summary of rag
// @Schemes
// @Description	Get a summary of rag
// @Tags			RAG
// @Accept			json
// @Produce		json
// @Param			ragName		query		string	true	"rag name"
// @Param			namespace	header		string	true	"Namespace of the rag"
// @Success		200			{object}	rag.Report
// @Failure		400			{object}	map[string]string
// @Failure		500			{object}	map[string]string
// @Router			/rags/report [get]
func (r *RagAPI) Summary(ctx *gin.Context) {
	ragName := ctx.Query(ragNameQuery)
	namespace := NamespaceInHeader(ctx)

	rr := v1alpha1.RAG{}
	if err := r.c.Get(ctx, types.NamespacedName{
		Namespace: namespace, Name: ragName,
	}, &rr); err != nil {
		klog.Error(fmt.Sprintf("can't get rag by name %s", ragName))
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("can't get rag by name %s", ragName),
		})
		return
	}
	thresholds := make(map[string]float64)
	for _, param := range rr.Spec.Metrics {
		thresholds[string(param.Kind)] = float64(param.ToleranceThreshbold) / 100.0
	}

	report, err := rag.ParseSummary(ctx.Request.Context(), r.c, rr.Spec.Application.Name, ragName, namespace, thresholds)
	if err != nil {
		klog.Errorf("an error occurred generating the report, error %s", err)
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, report)
}

// @Summary	Get detail data of a rag
// @Schemes
// @Description	Get detail data of a rag
// @Tags			RAG
// @Accept			json
// @Produce		json
// @Param			ragName		query		string	true	"rag name"
// @Param			namespace	header		string	true	"Namespace of the rag"
// @Param			page		query		int		false	"default is 1"
// @Param			size		query		string	false	"default is 10"
// @Param			sortBy		query		string	false	"rag metrcis"
// @Param			order		query		string	false	"desc or asc"
// @Success		200			{object}	rag.ReportDetail
// @Failure		400			{object}	map[string]string
// @Failure		500			{object}	map[string]string
// @Router			/rags/detail [get]
func (r *RagAPI) ReportDetail(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("size", "10"))
	sortBy := ctx.Query("sortBy")
	order := ctx.DefaultQuery("order", "desc")
	ragName := ctx.Query(ragNameQuery)
	namespace := NamespaceInHeader(ctx)

	rr := v1alpha1.RAG{}
	if err := r.c.Get(ctx, types.NamespacedName{
		Namespace: namespace, Name: ragName,
	}, &rr); err != nil {
		klog.Error(fmt.Sprintf("can't get rag by name %s", ragName))
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("can't get rag by name %s", ragName),
		})
		return
	}
	result, err := rag.ParseResult(ctx.Request.Context(), r.c, page, pageSize, rr.Spec.Application.Name, ragName, namespace, sortBy, order)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, result)
}

// @Summary	Get scatter data of a rag
// @Schemes
// @Description	Get scatter data of a rag
// @Tags			RAG
// @Accept			json
// @Produce		json
// @Param			ragName		query		string	true	"rag name"
// @Param			namespace	header		string	true	"Namespace of the rag"
// @Success		200			{object}	rag.ReportDetail
// @Failure		400			{object}	map[string]string
// @Failure		500			{object}	map[string]string
// @Router			/rags/scatter [get]
func (r *RagAPI) ReportScatter(ctx *gin.Context) {
	ragName := ctx.Query(ragNameQuery)
	namespace := NamespaceInHeader(ctx)

	rr := v1alpha1.RAG{}
	if err := r.c.Get(ctx, types.NamespacedName{
		Namespace: namespace, Name: ragName,
	}, &rr); err != nil {
		klog.Error(fmt.Sprintf("can't get rag by name %s", ragName))
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("can't get rag by name %s", ragName),
		})
		return
	}
	result, err := rag.PraseScatterChart(ctx.Request.Context(), r.c, rr.Spec.Application.Name, ragName, namespace)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"data": result,
	})
}

func registerRAG(g *gin.RouterGroup, conf gqlconfig.ServerConfig) {
	cfg := ctrl.GetConfigOrDie()
	c, err := client.New(cfg, client.Options{Scheme: conf.Scheme})
	if err != nil {
		panic(err)
	}
	api := RagAPI{c: c}

	g.GET("/report", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "rags"), api.Summary)
	g.GET("/detail", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "rags"), api.ReportDetail)
	g.GET("/scatter", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, v1alpha1.GroupVersion, "get", "rags"), api.ReportScatter)
}
