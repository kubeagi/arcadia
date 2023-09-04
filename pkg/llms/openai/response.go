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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kubeagi/arcadia/pkg/llms"
)

type Response struct {
	Code    int    `json:"code"`
	Data    string `json:"data"` // JSON format of the returned data
	Msg     string `json:"msg"`
	Success bool   `json:"success"`
}

func (response *Response) Type() llms.LLMType {
	return llms.OpenAI
}

func (response *Response) Bytes() []byte {
	bytes, err := json.Marshal(response)
	if err != nil {
		return []byte{}
	}
	return bytes
}

func (response *Response) String() string {
	return string(response.Bytes())
}

func (response *Response) Unmarshall(bytes []byte) error {
	return json.Unmarshal(bytes, response)
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
