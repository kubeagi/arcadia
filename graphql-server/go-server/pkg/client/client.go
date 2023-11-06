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

package client

import (
	"sync"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/kubeagi/arcadia/graphql-server/go-server/pkg/oidc"
)

var (
	once sync.Once
)

var getClientByIDToken func(string) (dynamic.Interface, error)

func InitClient(enableOIDC bool) {
	once.Do(func() {
		getClientByIDToken = func(idtoken string) (dynamic.Interface, error) {
			var (
				cfg *rest.Config
				err error
			)
			if !enableOIDC {
				cfg, err = ctrl.GetConfig()
			} else {
				cfg, err = clientcmd.BuildConfigFromKubeconfigGetter("", oidc.OIDCKubeGetter(idtoken))
			}
			if err != nil {
				return nil, err
			}
			return dynamic.NewForConfig(cfg)
		}
	})
}

func GetClientByIDToken(idtoken string) (dynamic.Interface, error) {
	return getClientByIDToken(idtoken)
}
