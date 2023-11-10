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
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

type idtokenKey struct{}

func isBearerToken(token string) (bool, string) {
	if len(token) < 6 {
		return false, ""
	}
	head := strings.ToLower(token[:6])
	payload := strings.TrimSpace(token[6:])
	return head == "bearer" && len(payload) > 0, payload
}

func AuthInterceptor(oidcVerifier *oidc.IDTokenVerifier, hf http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idtoken := r.Header.Get("Authorization")
		ok, rawToken := isBearerToken(idtoken)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("Unauthorized. Please provide an oidc token"))
			return
		}
		_, err := oidcVerifier.Verify(context.TODO(), rawToken)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		ctx := context.WithValue(r.Context(), idtokenKey{}, rawToken)
		r = r.WithContext(ctx)
		hf(w, r)
	}
}

func ForOIDCToken(ctx context.Context) *string {
	v, _ := ctx.Value(idtokenKey{}).(string)
	if v == "" {
		return nil
	}
	return &v
}
