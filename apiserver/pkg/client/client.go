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
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/utils/env"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/kubeagi/arcadia/apiserver/pkg/oidc"
)

func GetClient(idtoken *string) (dynamic.Interface, error) {
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

	cfg.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(50, 60)
	return dynamic.NewForConfig(cfg)
}
