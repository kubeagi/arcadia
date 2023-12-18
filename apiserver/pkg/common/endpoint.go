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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
)

// MakeEndpoint provides a common way to handle endpoint input
// owner means the resource who owns this endpoint
func MakeEndpoint(ctx context.Context, c dynamic.Interface, owner generated.TypedObjectReferenceInput, input generated.EndpointInput) (v1alpha1.Endpoint, error) {
	endpoint := v1alpha1.Endpoint{
		URL: input.URL,
	}

	// parse secure check policy
	if input.Insecure != nil {
		endpoint.Insecure = *input.Insecure
	}

	// parse auth secret
	if input.Auth != nil {
		// create auth secret
		secret := MakeAuthSecretName(owner.Name, owner.Kind)
		// retrieve owner to metav1.Object
		// object is not nil if `Get` succeeded
		var ownerObject metav1.Object
		resource, err := ResouceGet(ctx, c, owner, metav1.GetOptions{})
		if err == nil {
			ownerObject = resource
		}
		// when object is not nil, a owner reference will be set to this auth secret
		err = MakeAuthSecret(ctx, c, generated.TypedObjectReferenceInput{
			APIGroup:  &CoreV1APIGroup,
			Kind:      "Secret",
			Name:      secret,
			Namespace: owner.Namespace,
		}, *input.Auth, ownerObject)
		if err != nil {
			return endpoint, err
		}
		endpoint.AuthSecret = &v1alpha1.TypedObjectReference{
			Kind:      "Secret",
			Name:      secret,
			Namespace: owner.Namespace,
		}
	}

	return endpoint, nil
}

// MakeAuthSecretName returns a secret name based on its base name and owner's kind
func MakeAuthSecretName(base string, ownerKind string) string {
	return strings.ToLower(fmt.Sprintf("%s-%s-auth", base, ownerKind))
}

// MakeAuthSecret will create or update a secret based on auth input
// When owner is not nil, owner reference will be set
func MakeAuthSecret(ctx context.Context, c dynamic.Interface, secret generated.TypedObjectReferenceInput, input generated.AuthInput, owner metav1.Object) error {
	// initialize a auth secret
	authSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: CoreV1APIGroup,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.Name,
			Namespace: *secret.Namespace,
		},
		Data: map[string][]byte{
			"rootUser":     []byte(input.Username),
			"rootPassword": []byte(input.Password),
		},
	}

	// set owner reference
	if owner != nil {
		if err := controllerutil.SetControllerReference(owner, authSecret, scheme); err != nil {
			return err
		}
	}

	unstructuredDatasource, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&authSecret)
	if err != nil {
		return err
	}

	_, err = ResouceGet(ctx, c, secret, metav1.GetOptions{})
	if err != nil {
		// Create is not fount
		if apierrors.IsNotFound(err) {
			_, err = c.Resource(schema.GroupVersionResource{Group: corev1.SchemeGroupVersion.Group, Version: corev1.SchemeGroupVersion.Version, Resource: "secrets"}).
				Namespace(*secret.Namespace).Create(ctx, &unstructured.Unstructured{Object: unstructuredDatasource}, metav1.CreateOptions{})
			if err != nil {
				return err
			}
			return nil
		}
		return err
	}
	// update if found
	_, err = c.Resource(schema.GroupVersionResource{Group: corev1.SchemeGroupVersion.Group, Version: corev1.SchemeGroupVersion.Version, Resource: "secrets"}).
		Namespace(*secret.Namespace).Update(ctx, &unstructured.Unstructured{Object: unstructuredDatasource}, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}
