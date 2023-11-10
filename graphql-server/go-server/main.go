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

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"k8s.io/klog/v2"

	"github.com/kubeagi/arcadia/graphql-server/go-server/graph"
	"github.com/kubeagi/arcadia/graphql-server/go-server/pkg/auth"
	"github.com/kubeagi/arcadia/graphql-server/go-server/pkg/oidc"

	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

var (
	// We should define a structure to store these configurations
	host = flag.String("host", "", "bind to the host, default is 0.0.0.0")
	port = flag.Int("port", 8081, "service listening port")

	enablePlayground         = flag.Bool("enable-playground", false, "enable the graphql playground")
	enableOIDC               = flag.Bool("enable-oidc", false, "enable oidc authorization")
	playgroundEndpointPrefix = flag.String("playground-endpoint-prefix", "", "this parameter should also be configured when the service is forwarded via ingress and a path prefix is configured to avoid not finding the service, such as /apis")

	// Flags fro oidc client
	issuerURL    = flag.String("issuer-url", "", "oidc issuer url(required when enable odic)")
	masterURL    = flag.String("master-url", "", "k8s master url(required when enable odic)")
	clientID     = flag.String("client-id", "", "oidc client id(required when enable odic)")
	clientSecret = flag.String("client-secret", "", "oidc client secret(required when enable odic)")
)

func main() {
	flag.Parse()

	if *enableOIDC {
		oidc.InitOIDCArgs(*issuerURL, *masterURL, *clientSecret, *clientID)
	}

	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}}))
	srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
		rc := graphql.GetFieldContext(ctx)
		klog.Infoln("Entered", rc.Object, rc.Field.Name)
		res, err = next(ctx)
		klog.Infoln("Left", rc.Object, rc.Field.Name, "=>", res, err)
		return res, err
	})

	if *enablePlayground {
		endpoint := "/bff"
		if *playgroundEndpointPrefix != "" {
			if prefix := strings.TrimPrefix(strings.TrimSuffix(*playgroundEndpointPrefix, "/"), "/"); prefix != "" {
				endpoint = fmt.Sprintf("/%s%s", prefix, endpoint)
			}
		}
		http.Handle("/", playground.Handler("Arcadia-Graphql-Server", endpoint))
	}

	if *enableOIDC {
		http.Handle("/bff", auth.AuthInterceptor(oidc.Verifier, srv.ServeHTTP))
	} else {
		http.Handle("/bff", srv)
	}

	klog.Infof("listening server on port: %d", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", *host, *port), nil))
}
