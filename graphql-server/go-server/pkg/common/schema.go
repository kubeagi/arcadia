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

package common

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	apichain "github.com/kubeagi/arcadia/api/app-node/chain/v1alpha1"
	apiprompt "github.com/kubeagi/arcadia/api/app-node/prompt/v1alpha1"
	apiretriever "github.com/kubeagi/arcadia/api/app-node/retriever/v1alpha1"
	"github.com/kubeagi/arcadia/api/base/v1alpha1"
)

var (
	scheme = runtime.NewScheme()

	// CoreV1
	CoreV1APIGroup = corev1.SchemeGroupVersion.String()
	corev1Schemas  = map[string]schema.GroupVersionResource{
		"secret": {
			Group:    corev1.SchemeGroupVersion.Group,
			Version:  corev1.SchemeGroupVersion.Version,
			Resource: "secrets",
		},
	}

	// Arcadia
	ArcadiaAPIGroup = v1alpha1.GroupVersion.String()
	arcadiaSchemas  = map[string]schema.GroupVersionResource{
		"datasource": {
			Group:    v1alpha1.GroupVersion.Group,
			Version:  v1alpha1.GroupVersion.Version,
			Resource: "datasources",
		},
		"dataset": {
			Group:    v1alpha1.GroupVersion.Group,
			Version:  v1alpha1.GroupVersion.Version,
			Resource: "datasets",
		},
		"application": {
			Group:    v1alpha1.GroupVersion.Group,
			Version:  v1alpha1.GroupVersion.Version,
			Resource: "applications",
		},
		"prompt": {
			Group:    apiprompt.GroupVersion.Group,
			Version:  apiprompt.GroupVersion.Version,
			Resource: "prompts",
		},
		"chain": {
			Group:    apichain.GroupVersion.Group,
			Version:  apichain.GroupVersion.Version,
			Resource: "chains",
		},
		"retriever": {
			Group:    apiretriever.GroupVersion.Group,
			Version:  apiretriever.GroupVersion.Version,
			Resource: "retrievers",
		},
	}
)

// SchemaOf returns the GroupVersionResource by resource kind
func SchemaOf(apiGroup *string, kind string) schema.GroupVersionResource {
	// corev1
	if apiGroup == nil || *apiGroup == CoreV1APIGroup {
		return corev1Schemas[strings.ToLower(kind)]
	}

	// arcadia
	if strings.Contains(*apiGroup, "arcadia") {
		return arcadiaSchemas[strings.ToLower(kind)]
	}
	return schema.GroupVersionResource{}
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(apichain.AddToScheme(scheme))
	utilruntime.Must(apiprompt.AddToScheme(scheme))
	utilruntime.Must(apiretriever.AddToScheme(scheme))
}
