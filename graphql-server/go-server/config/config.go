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

	"k8s.io/klog/v2"
)

type ServerConfig struct {
	Host             string
	Port             int
	EnablePlayground bool
	EnableOIDC       bool

	PlaygroundEndpointPrefix                     string
	IssuerURL, MasterURL, ClientID, ClientSecret string

	DataProcessURL string
}

func NewServerFlags() ServerConfig {
	s := ServerConfig{}
	flag.StringVar(&s.Host, "host", "", "bind to the host, default is 0.0.0.0")
	flag.IntVar(&s.Port, "port", 8081, "service listening port")
	flag.BoolVar(&s.EnablePlayground, "enable-playground", false, "enable the graphql playground")
	flag.BoolVar(&s.EnableOIDC, "enable-oidc", false, "enable oidc authorization")
	flag.StringVar(&s.PlaygroundEndpointPrefix, "playground-endpoint-prefix", "", "this parameter should also be configured when the service is forwarded via ingress and a path prefix is configured to avoid not finding the service, such as /apis")
	flag.StringVar(&s.IssuerURL, "issuer-url", "", "oidc issuer url(required when enable odic)")
	flag.StringVar(&s.MasterURL, "master-url", "", "k8s master url(required when enable odic)")
	flag.StringVar(&s.ClientID, "client-id", "", "oidc client id(required when enable odic)")
	flag.StringVar(&s.ClientSecret, "client-secret", "", "oidc client secret(required when enable odic)")
	flag.StringVar(&s.DataProcessURL, "data-processing-url", "http://127.0.0.1:28888", "url to access data processing server")

	klog.InitFlags(nil)
	flag.Parse()
	return s
}
