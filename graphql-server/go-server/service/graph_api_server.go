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
	"context"
	"fmt"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"github.com/kubeagi/arcadia/graphql-server/go-server/config"
	"github.com/kubeagi/arcadia/graphql-server/go-server/graph/generated"
	"github.com/kubeagi/arcadia/graphql-server/go-server/graph/impl"
	"github.com/kubeagi/arcadia/graphql-server/go-server/pkg/auth"
	"github.com/kubeagi/arcadia/graphql-server/go-server/pkg/oidc"
)

func graphqlHandler() gin.HandlerFunc {
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &impl.Resolver{}}))
	srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
		rc := graphql.GetFieldContext(ctx)
		klog.Infoln("Entered", rc.Object, rc.Field.Name)
		res, err = next(ctx)
		klog.Infoln("Left", rc.Object, rc.Field.Name, "=>", res, err)
		return res, err
	})
	return func(c *gin.Context) {
		srv.ServeHTTP(c.Writer, c.Request)
	}
}

func RegisterGraphQL(g *gin.Engine, conf config.ServerConfig) {
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &impl.Resolver{}}))
	srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
		rc := graphql.GetFieldContext(ctx)
		klog.Infoln("Entered", rc.Object, rc.Field.Name)
		res, err = next(ctx)
		klog.Infoln("Left", rc.Object, rc.Field.Name, "=>", res, err)
		return res, err
	})

	if conf.EnablePlayground {
		endpoint := "/bff"
		if conf.PlaygroundEndpointPrefix != "" {
			if prefix := strings.TrimPrefix(strings.TrimSuffix(conf.PlaygroundEndpointPrefix, "/"), "/"); prefix != "" {
				endpoint = fmt.Sprintf("/%s%s", prefix, endpoint)
			}
		}
		g.GET("/", gin.WrapH(playground.Handler("Arcadia-Graphql-Server", endpoint)))
	}

	g.POST("/bff", auth.AuthInterceptor(conf.EnableOIDC, oidc.Verifier, "", ""), graphqlHandler())
}
