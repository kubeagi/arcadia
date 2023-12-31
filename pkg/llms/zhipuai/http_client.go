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

// NOTE: Reference zhipuai's python sdk: utils/http_client.py

package zhipuai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func setHeadersWithToken(req *http.Request, token string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", token)
}

func parseHTTPResponse(resp *http.Response) (*Response, error) {
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("exception: %s", resp.Status)
	}

	var data = new(Response)
	err := json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func Post(apiURL, token string, params ModelParams, timeout time.Duration) (*Response, error) {
	jsonParams, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonParams))
	if err != nil {
		return nil, err
	}

	setHeadersWithToken(req, token)

	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseHTTPResponse(resp)
}

func Get(apiURL, token string, timeout time.Duration) (*Response, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	setHeadersWithToken(req, token)

	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseHTTPResponse(resp)
}

// EmbeddingPost posts embedding request to model
// FIXME: should this be made an interface?
func EmbeddingPost(apiURL, token string, text EmbeddingText, timeout time.Duration) (*EmbeddingResponse, error) {
	jsonParams, err := json.Marshal(text)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonParams))
	if err != nil {
		return nil, err
	}

	setHeadersWithToken(req, token)

	client := http.Client{
		Timeout: timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("exception: %s", resp.Status)
	}

	var data = new(EmbeddingResponse)
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, err
	}

	return data, nil
}
