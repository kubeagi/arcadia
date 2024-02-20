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
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	pkgclient "github.com/kubeagi/arcadia/apiserver/pkg/client"
)

// MakeEndpoint provides a common way to handle endpoint input
// owner means the resource who owns this endpoint
func MakeEndpoint(ctx context.Context, c client.Client, owner client.Object, input generated.EndpointInput) (v1alpha1.Endpoint, error) {
	endpoint := v1alpha1.Endpoint{
		URL: input.URL,
	}

	// parse secure check policy
	endpoint.Insecure = pointer.BoolPtrDerefOr(input.Insecure, endpoint.Insecure)

	// parse auth secret
	if input.Auth != nil {
		// generate auth secret name
		kind := ""
		switch owner.(type) {
		case *v1alpha1.Datasource:
			kind = "Datasource"
		case *v1alpha1.Embedder:
			kind = "Embedder"
		case *v1alpha1.LLM:
			kind = "LLM"
		}
		secret := GenerateAuthSecretName(owner.GetName(), kind)
		// retrieve owner to metav1.Object
		// object is not nil if `Get` succeeded
		ownerNamespace := owner.GetNamespace()
		if err := c.Get(ctx, client.ObjectKeyFromObject(owner), owner); err != nil {
			owner = nil
		}
		// when object is not nil, a owner reference will be set to this auth secret
		err := MakeAuthSecret(ctx, c, ownerNamespace, secret, input.Auth, owner)
		if err != nil {
			return endpoint, err
		}
		endpoint.AuthSecret = &v1alpha1.TypedObjectReference{
			Kind:      "Secret",
			Name:      secret,
			Namespace: pointer.String(ownerNamespace),
		}
	}

	return endpoint, nil
}

// GenerateAuthSecretName returns a secret name based on its base name and owner's kind
func GenerateAuthSecretName(base string, ownerKind string) string {
	return strings.ToLower(fmt.Sprintf("%s-%s-auth", base, ownerKind))
}

// MakeAuthSecret will create or update a secret based on auth input
// When owner is not nil, owner reference will be set
func MakeAuthSecret(ctx context.Context, c client.Client, secretNamespace, secretName string, input map[string]interface{}, owner client.Object) error {
	authSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: secretNamespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, c, authSecret, func() error {
		// copy input map into data
		data := map[string][]byte{}
		for k, v := range input {
			data[k] = []byte(fmt.Sprintf(v.(string)))
		}
		authSecret.Data = data

		// set owner reference
		if owner != nil {
			if err := controllerutil.SetControllerReference(owner, authSecret, pkgclient.Scheme); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}
