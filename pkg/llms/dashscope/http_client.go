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

package dashscope

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/kubeagi/arcadia/pkg/llms"
)

func setHeaders(req *http.Request, token string, sse, async bool) {
	if sse {
		// req.Header.Set("Content-Type", "text/event-stream") // Although the documentation says we should do this, but will return a 400 error and the python sdk doesn't do this.
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("X-DashScope-SSE", "enable")
	} else {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "*/*")
	}
	if async {
		req.Header.Set("X-DashScope-Async", "enable")
	}
	req.Header.Set("Authorization", "Bearer "+token)
}

func parseHTTPResponse(resp *http.Response, data llms.Response) (llms.Response, error) {
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func req(ctx context.Context, apiURL, token string, data []byte, sse, async bool) (resp *http.Response, err error) {
	var req *http.Request
	if len(data) == 0 {
		req, err = http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	} else {
		req, err = http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(data))
	}
	if err != nil {
		return nil, err
	}

	setHeaders(req, token, sse, async)

	return http.DefaultClient.Do(req)
}
func do(ctx context.Context, apiURL, token string, data []byte, sse, async bool, model Model) (llms.Response, error) {
	resp, err := req(ctx, apiURL, token, data, sse, async)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var respData llms.Response
	if model == CHATGLM6BV2 {
		respData = &ResponseChatGLB6B{}
	} else {
		respData = &Response{}
	}
	return parseHTTPResponse(resp, respData)
}
