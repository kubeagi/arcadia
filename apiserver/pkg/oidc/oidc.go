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

package oidc

import (
	"context"
	"crypto/tls"
	"net/http"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
)

var (
	once     sync.Once
	Verifier *oidc.IDTokenVerifier

	issuerURL1, masterURL1, clientSecret1, clientID1 string
)

func InitOIDCArgs(issuerURL, masterURL, clientSecret, clientID string) {
	once.Do(func() {
		issuerURL1 = issuerURL
		masterURL1 = masterURL
		clientSecret1 = clientSecret
		clientID1 = clientID

		ctx := oidc.ClientContext(context.TODO(), &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		})
		provider, err := oidc.NewProvider(ctx, issuerURL)
		if err != nil {
			panic(err)
		}
		Verifier = provider.Verifier(&oidc.Config{ClientID: clientID})
		klog.V(5).Infof("oidc token validation was successful. issuerurl: %s, masterurl: %s, clientid: %s, clientsecret: %s", issuerURL1, masterURL1, clientID1, clientSecret1)
	})
}

func OIDCKubeGetter(idtoken string) func() (*clientcmdapi.Config, error) {
	return func() (*clientcmdapi.Config, error) {
		return &clientcmdapi.Config{
			Kind:       "Config",
			APIVersion: "v1",
			Clusters: map[string]*clientcmdapi.Cluster{
				"kube-oidc-proxy": {
					Server:                masterURL1,
					InsecureSkipTLSVerify: true,
				},
			},
			Contexts: map[string]*clientcmdapi.Context{
				"oidc@kube-oidc-proxy": {
					Cluster:  "kube-oidc-proxy",
					AuthInfo: "oidc",
				},
			},
			CurrentContext: "oidc@kube-oidc-proxy",
			AuthInfos: map[string]*clientcmdapi.AuthInfo{
				"oidc": {
					AuthProvider: &clientcmdapi.AuthProviderConfig{
						Name: "oidc",
						Config: map[string]string{
							"client-id":      clientID1,
							"client-secret":  clientSecret1,
							"id-token":       idtoken,
							"idp-issuer-url": issuerURL1,
						},
					},
				},
			},
		}, nil
	}
}
