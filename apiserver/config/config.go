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
package config

import (
	"flag"
	"os"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	evaluationarcadiav1alpha1 "github.com/kubeagi/arcadia/api/evaluation/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/pkg/dataprocessing"
)

var s = &ServerConfig{}

type ServerConfig struct {
	Scheme *runtime.Scheme

	// Debug mode which only have graphql server running
	Debug bool
	// SystemNamespace which hosts system resources
	SystemNamespace string

	Host string
	Port int

	PlaygroundEndpointPrefix string

	// EnablePlayground is true when graphql playground is going to be utilized
	EnablePlayground bool
	// EnableSwagger is true when swagger is going to be utilized
	EnableSwagger bool

	// OIDC configurations
	EnableOIDC                                   bool
	IssuerURL, MasterURL, ClientID, ClientSecret string

	// DataProcessURL is the URL of the data process service
	DataProcessURL string
}

func NewServerFlags() ServerConfig {
	flag.StringVar(&s.SystemNamespace, "system-namespace", os.Getenv("POD_NAMESPACE"), "system namespace where kubeagi has been installed")
	flag.StringVar(&s.Host, "host", "", "bind to the host, default is 0.0.0.0")
	flag.IntVar(&s.Port, "port", 8081, "service listening port")
	flag.BoolVar(&s.EnablePlayground, "enable-playground", false, "enable the graphql playground")
	flag.BoolVar(&s.EnableSwagger, "enable-swagger", true, "enable the swagger doc")
	flag.BoolVar(&s.EnableOIDC, "enable-oidc", false, "enable oidc authorization")
	flag.StringVar(&s.PlaygroundEndpointPrefix, "playground-endpoint-prefix", "", "this parameter should also be configured when the service is forwarded via ingress and a path prefix is configured to avoid not finding the service, such as /apis")
	flag.StringVar(&s.IssuerURL, "issuer-url", "", "oidc issuer url(required when enable odic)")
	flag.StringVar(&s.MasterURL, "master-url", "", "k8s master url(required when enable odic)")
	flag.StringVar(&s.ClientID, "client-id", "", "oidc client id(required when enable odic)")
	flag.StringVar(&s.ClientSecret, "client-secret", "", "oidc client secret(required when enable odic)")
	flag.StringVar(&s.DataProcessURL, "data-processing-url", "http://127.0.0.1:28888", "url to access data processing server")
	flag.BoolVar(&s.Debug, "debug", false, "debug model for apiserver")

	klog.InitFlags(nil)
	flag.Parse()

	s.Scheme = runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(s.Scheme))
	utilruntime.Must(v1.AddToScheme(s.Scheme))
	utilruntime.Must(evaluationarcadiav1alpha1.AddToScheme(s.Scheme))
	utilruntime.Must(v1alpha1.AddToScheme(s.Scheme))

	dataprocessing.Init(s.DataProcessURL)
	return *s
}

func GetConfig() ServerConfig {
	return *s
}
