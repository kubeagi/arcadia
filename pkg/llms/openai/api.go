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

package openai

import (
	"fmt"
	"net/http"
	"time"

	"github.com/kubeagi/arcadia/pkg/llms"
)

const (
	OpenaiModelAPIURL    = "https://api.openai.com/v1"
	OpenaiDefaultTimeout = 300 * time.Second
)

var _ llms.LLM = (*OpenAI)(nil)

type OpenAI struct {
	apiKey string
}

func NewOpenAI(auth string) *OpenAI {
	return &OpenAI{
		apiKey: auth,
	}
}

func (o OpenAI) Type() llms.LLMType {
	return llms.OpenAI
}

func (o *OpenAI) Validate() (llms.Response, error) {
	// Validate OpenAI type CRD LLM Instance
	// instance.Spec.URL should be like "https://api.openai.com/"

	if o.apiKey == "" {
		// TODO: maybe we should consider local pseudo-openAI LLM worker that doesn't require an apiKey?
		return nil, fmt.Errorf("auth is empty")
	}

	testURL := OpenaiModelAPIURL + "/models"
	testAuth := "Bearer " + o.apiKey // openAI official requirement

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", testAuth)
	req.Header.Set("Content-Type", "application/json")

	cli := &http.Client{}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("returns unexpected status code: %d", resp.StatusCode)
	}

	// FIXME: response object
	response, err := parseHTTPResponse(resp)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// TODO: Openai Model Object & Other definition
