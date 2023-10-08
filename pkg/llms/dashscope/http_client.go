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
)

func setHeaders(req *http.Request, token string, sse bool) {
	if sse {
		// req.Header.Set("Content-Type", "text/event-stream") // Although the documentation says we should do this, but will return a 400 error and the python sdk doesn't do this.
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("X-DashScope-SSE", "enable")
	} else {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "*/*")
	}
	req.Header.Set("Authorization", "Bearer "+token)
}

func parseHTTPResponse(resp *http.Response) (data *Response, err error) {
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func req(ctx context.Context, apiURL, token string, data []byte, sse bool) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	setHeaders(req, token, sse)

	return http.DefaultClient.Do(req)
}
func do(ctx context.Context, apiURL, token string, data []byte, sse bool) (*Response, error) {
	resp, err := req(ctx, apiURL, token, data, sse)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return parseHTTPResponse(resp)
}
