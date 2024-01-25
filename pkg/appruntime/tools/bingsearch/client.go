/*
Copyright 2024 KubeAGI.

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

package bingsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"k8s.io/klog/v2"
)

const (
	Endpoint      = "https://api.bing.microsoft.com/v7.0/search?mkt=zh-CN&q="
	AuthHeaderKey = "Ocp-Apim-Subscription-Key"
)

type BingClient struct {
	apiKey string
}

func NewBingClient(apiKey string) *BingClient {
	if apiKey == "" {
		apiKey = os.Getenv("BING_KEY")
	}
	return &BingClient{
		apiKey: apiKey,
	}
}

func (client *BingClient) Search(ctx context.Context, query string) (string, error) {
	p, data, err := client.GetWebPages(ctx, query)
	if len(p) > 0 {
		return FormatResults(p), nil
	}
	return data, err
}
func (client *BingClient) GetWebPages(ctx context.Context, query string) (p []WebPage, data string, err error) {
	queryURL := Endpoint + url.QueryEscape(query)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("creating bingSearch request failed: %w", err)
	}
	request.Header.Add(AuthHeaderKey, client.apiKey)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, "", fmt.Errorf("bingSearch[%s] get error: %w", queryURL, err)
	}

	defer response.Body.Close()
	code := response.StatusCode
	resp := &RespData{}
	if err := json.NewDecoder(response.Body).Decode(&resp); err != nil {
		return nil, "", fmt.Errorf("bingSearch parse json resp get err:%w, http status code:%d", err, code)
	}
	if resp.ErrorResp != nil {
		return nil, "", fmt.Errorf("bingSearch get error resp from bing server: http status code:%d message:%s, code:%s", code, resp.ErrorResp.Message, resp.ErrorResp.Code)
	}
	if len(resp.WebPages.Value) > 0 {
		p = make([]WebPage, len(resp.WebPages.Value))
		for i, v := range resp.WebPages.Value {
			v := v
			p[i] = WebPage{
				Title:       v.Name,
				Description: v.Snippet,
				URL:         v.URL,
			}
		}
	}
	bytes, err := json.Marshal(resp)
	if err != nil {
		return nil, "", fmt.Errorf("bingSearch json marshal resp, get err:%w", err)
	}
	klog.V(3).Infof("bingSearch get webpages: %#v", p)
	klog.V(5).Infof("bingSearch get resp: %s", string(bytes))
	return p, string(bytes), nil
}

func FormatResults(vals []WebPage) (res string) {
	for _, val := range vals {
		res += fmt.Sprintf("Title: %s\nDescription: %s\nURL: %s\n\n", val.Title, val.Description, val.URL)
	}
	return res
}
