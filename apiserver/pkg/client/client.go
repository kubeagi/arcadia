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
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/utils/env"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	agentv1alpha1 "github.com/kubeagi/arcadia/api/app-node/agent/v1alpha1"
	apichain "github.com/kubeagi/arcadia/api/app-node/chain/v1alpha1"
	apiprompt "github.com/kubeagi/arcadia/api/app-node/prompt/v1alpha1"
	apiretriever "github.com/kubeagi/arcadia/api/app-node/retriever/v1alpha1"
	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	evaluationarcadiav1alpha1 "github.com/kubeagi/arcadia/api/evaluation/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/pkg/oidc"
)

func GetClient(idtoken *string) (client.Client, error) {
	var (
		cfg *rest.Config
		err error
	)
	if idtoken == nil {
		cfg, err = ctrl.GetConfig()
	} else {
		cfg, err = clientcmd.BuildConfigFromKubeconfigGetter("", oidc.OIDCKubeGetter(*idtoken))
	}
	if err != nil {
		return nil, err
	}
	// RateLimiter: default qps is 5，default burst is 10, so increase them
	qps, _ := env.GetInt("K8S_REST_CONFIG_QPS", 50)
	burst, _ := env.GetInt("K8S_REST_CONFIG_BURST", 60)
	// both should be configured if need to customize
	if qps > 0 && burst > 0 {
		cfg.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(float32(qps), burst)
	}
	cli, err := client.New(cfg, client.Options{
		Scheme: Scheme,
	})
	if err != nil {
		return nil, err
	}
	return cli, nil
}

var (
	Scheme = runtime.NewScheme()
)

// Note: like `init()` function in `main.go` for controllers, this `init()` function is for client in apiserver.
// for example, If you want to use pod in apiserver client, you should add `v1.AddToScheme(Scheme)` here.
func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(Scheme))
	utilruntime.Must(arcadiav1alpha1.AddToScheme(Scheme))
	utilruntime.Must(v1.AddToScheme(Scheme))
	utilruntime.Must(apichain.AddToScheme(Scheme))
	utilruntime.Must(apiprompt.AddToScheme(Scheme))
	utilruntime.Must(apiretriever.AddToScheme(Scheme))
	utilruntime.Must(evaluationarcadiav1alpha1.AddToScheme(Scheme))
	utilruntime.Must(batchv1.AddToScheme(Scheme))
	utilruntime.Must(agentv1alpha1.AddToScheme(Scheme))
}
