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

// NOTE: Reference zhipuai's python sdk: utils/jwt_token.py
package zhipuai

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
)

const (
	API_TOKEN_TTL_SECONDS = 3 * 60
	// FIXME: impl TLL Cache
	CACHE_TTL_SECONDS = (API_TOKEN_TTL_SECONDS - 30)
)

func GenerateToken(apikey string, expSeconds int64) (string, error) {
	parts := strings.Split(apikey, ".")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid apikey")
	}

	id := parts[0]
	secret := parts[1]

	currentTime := time.Now().UnixMilli()
	claims := jwt.MapClaims{
		"api_key":   id,
		"exp":       currentTime + expSeconds*1000,
		"timestamp": currentTime,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["sign_type"] = "SIGN"
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
