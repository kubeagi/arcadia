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

package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	av1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	pkgclient "github.com/kubeagi/arcadia/apiserver/pkg/client"
)

var NeedAuth bool

type contextKey string

const (
	idTokenContextKey  contextKey = "idToken"
	UserNameContextKey contextKey = "userName"
)

type User struct {
	Name        string            `json:"name"`
	Password    string            `json:"password,omitempty"`
	Email       string            `json:"email"`
	Phone       string            `json:"phone"`
	Description string            `json:"description"`
	Groups      []string          `json:"groups"`
	Role        string            `json:"role,omitempty"`
	CreateTime  string            `json:"creationTimestamp,omitempty"`
	Type        string            `json:"type"`
	Subject     string            `json:"sub,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

func isBearerToken(token string) (string, bool) {
	if len(token) < 6 {
		return "", false
	}
	head := strings.ToLower(token[:6])
	payload := strings.TrimSpace(token[6:])
	return payload, head == "bearer" && len(payload) > 0
}

// TODO: We could consider abstracting a validation function, that way we could have multiple validation methods.
func cani(c runtimeclient.Client, oidcToken *oidc.IDToken, resourceAttributes *av1.ResourceAttributes) (bool, string, error) {
	u := &User{}
	if err := oidcToken.Claims(u); err != nil {
		klog.Errorf("parse user info from idToken, error %v", err)
		return false, "", fmt.Errorf("can't parse user info")
	}

	av := &av1.SubjectAccessReview{
		Spec: av1.SubjectAccessReviewSpec{
			ResourceAttributes: resourceAttributes,
			Groups:             u.Groups,
			User:               u.Name,
		},
	}
	err := c.Create(context.TODO(), av)
	if err != nil {
		err = fmt.Errorf("auth can-i failed, error %w", err)
		klog.Error(err)
		return false, "", err
	}

	return av.Status.Allowed, u.Name, nil
}

func AuthInterceptor(needAuth bool, oidcVerifier *oidc.IDTokenVerifier, groupVersion schema.GroupVersion, verb, resources string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !needAuth {
			ctx.Next()
			return
		}
		rawToken := ctx.GetHeader("Authorization")
		namespace := ctx.GetHeader("namespace")
		rawToken, ok := isBearerToken(rawToken)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized, not bearer token",
			})
			return
		}

		oidcIDtoken, err := oidcVerifier.Verify(ctx.Request.Context(), rawToken)
		if err != nil {
			klog.Errorf("auth error: illegal token, rawtoken %s, error %s", rawToken, err)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": fmt.Sprintf("illegal token. error %s", err),
			})
			return
		}

		// Use operator permissions to determine if a user has permission to perform an operation.
		client, err := pkgclient.GetClient(nil)
		if err != nil {
			klog.Errorf("auth error: failed to connect cluster error %s", err)
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message": fmt.Sprintf("can't connect to cluster. error %s", err),
			})
			return
		}
		if verb != "" {
			ra := &av1.ResourceAttributes{Group: groupVersion.Group, Version: groupVersion.Version, Verb: verb, Resource: resources, Namespace: namespace}
			allowed, userName, err := cani(client, oidcIDtoken, ra)
			if err != nil {
				klog.Errorf("auth error: failed to checkout permission. error %s", err)
				ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"message": fmt.Sprintf("some error occurred in checking the permissions. error %s", err),
				})
				return
			}
			if !allowed {
				klog.Warningf("auth failed: you don't have permission to perform this operations. resource: %s, verb: %s, namespace: %s", resources, verb, namespace)
				ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"message": "you do not have permission to perform this operation. Please check the permissions.",
				})
				return
			}
			ctx.Request = ctx.Request.WithContext(context.WithValue(ctx.Request.Context(), UserNameContextKey, userName))
		} else {
			u := &User{}
			if err := oidcIDtoken.Claims(u); err == nil {
				ctx.Request = ctx.Request.WithContext(context.WithValue(ctx.Request.Context(), UserNameContextKey, u.Name))
			}
		}

		// for graphql query
		ctx.Request = ctx.Request.WithContext(context.WithValue(ctx.Request.Context(), idTokenContextKey, rawToken))
		ctx.Next()
	}
}

func AuthInterceptorInGraphql(needAuth bool, oidcVerifier *oidc.IDTokenVerifier) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !needAuth {
			NeedAuth = false
			ctx.Next()
			return
		}
		NeedAuth = true
		rawToken, ok := isBearerToken(ctx.GetHeader("Authorization"))
		if ok {
			oidcIDtoken, err := oidcVerifier.Verify(ctx.Request.Context(), rawToken)
			if err == nil {
				u := &User{}
				if err := oidcIDtoken.Claims(u); err == nil {
					ctx.Request = ctx.Request.WithContext(context.WithValue(ctx.Request.Context(), UserNameContextKey, u.Name))
				}
				ctx.Request = ctx.Request.WithContext(context.WithValue(ctx.Request.Context(), idTokenContextKey, rawToken))
			}
		}

		ctx.Next()
	}
}

func AuthTokenIsValid(needAuth bool, oidcVerifier *oidc.IDTokenVerifier) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !needAuth {
			ctx.Next()
			return
		}
		bearerToken := ctx.GetHeader("Authorization")
		rawToken, ok := isBearerToken(bearerToken)
		if !ok {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": "unauthorized, not bearer token",
			})
			return
		}

		oidcIDtoken, err := oidcVerifier.Verify(ctx.Request.Context(), rawToken)
		if err != nil {
			klog.Errorf("auth error: illegal token, rawtoken %s, error %s", rawToken, err)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"message": fmt.Sprintf("illegal token. error %s", err),
			})
			return
		}
		u := &User{}
		if err := oidcIDtoken.Claims(u); err == nil {
			ctx.Request = ctx.Request.WithContext(context.WithValue(ctx.Request.Context(), UserNameContextKey, u.Name))
		}
		ctx.Request = ctx.Request.WithContext(context.WithValue(ctx.Request.Context(), idTokenContextKey, rawToken))
		ctx.Next()
	}
}

func ForOIDCToken(ctx context.Context) *string {
	v, _ := ctx.Value(idTokenContextKey).(string)
	if v == "" {
		return nil
	}
	return &v
}
