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
	"strconv"
	"sync"

	"github.com/tmc/langchaingo/tools/scraper"
	"k8s.io/klog/v2"
)

const (
	Endpoint      = "https://api.bing.microsoft.com/v7.0/search"
	AuthHeaderKey = "Ocp-Apim-Subscription-Key"
)

type BingClient struct {
	options options
}

type options struct {
	apiKey         string
	count          int
	responseFilter string
	promote        string
	mkt            string
	answerCount    int
	scraperPage    bool
}

func defaultOptions() options {
	return options{
		apiKey:         os.Getenv("BING_KEY"),
		count:          5,
		responseFilter: "News,Webpages",
		promote:        "News,Webpages",
		mkt:            "zh-CN",
		answerCount:    2,
	}
}

type Option func(*options)

func WithAPIKey(apiKey string) Option {
	return func(opts *options) {
		if len(apiKey) != 0 {
			opts.apiKey = apiKey
		}
	}
}

func WithCount(count int) Option {
	return func(opts *options) {
		if count > 0 {
			opts.count = count
		}
	}
}

func WithScraperPage(scraperPage bool) Option {
	return func(opts *options) {
		opts.scraperPage = scraperPage
	}
}

func NewBingClient(opts ...Option) *BingClient {
	clientOptions := defaultOptions()
	for _, opt := range opts {
		opt(&clientOptions)
	}
	return &BingClient{clientOptions}
}

func (client *BingClient) Search(ctx context.Context, query string) (string, error) {
	p, data, err := client.SearchGetDetailData(ctx, query)
	if len(p) > 0 {
		return FormatResults(p), nil
	}
	return data, err
}

// SearchGetDetailData will try to parse bing search list type webpages and news.
// Unlike the Search method, it returns a more detailed list of structures, not just a string.
// Note: only parse search list, not single source page.
func (client *BingClient) SearchGetDetailData(ctx context.Context, query string) (resp []WebPage, data string, err error) {
	want := client.options.count
	remains := want
	// count max value is 50, ref: https://learn.microsoft.com/en-us/rest/api/cognitiveservices-bingsearch/bing-web-api-v7-reference#query-parameters
	// offset default value is 0, same ref with above
	count, offset := 50, 0
	resp = make([]WebPage, 0)
	for remains > 0 {
		if want < count {
			count = want
		}
		data, err := client.getOnePage(ctx, query, count, offset)
		if err != nil {
			return nil, "", err
		}
		if len(data) == 0 {
			break
		}
		resp = append(resp, data...)
		offset += len(data)
		remains = want - len(resp)
	}
	if len(resp) > want {
		resp = resp[:want]
	}
	bytes, err := json.Marshal(resp)
	if err != nil {
		return nil, "", fmt.Errorf("bingSearch json marshal resp, get err:%w", err)
	}
	logger := klog.FromContext(ctx)
	logger.V(5).Info(fmt.Sprintf("bingSearch get resp: %s", string(bytes)))
	if client.options.scraperPage {
		var wg sync.WaitGroup
		wg.Add(len(resp))
		for i, data := range resp {
			u := data.URL
			go func(i int, URL string) {
				defer wg.Done()
				s, err := scraper.New()
				if err != nil {
					logger.V(3).Error(err, "failed to create a new scraper")
					return
				}
				resp[i].Content, err = s.Call(ctx, URL)
				if err != nil {
					logger.V(3).Error(err, fmt.Sprintf("failed to scraper page: %s", URL))
					return
				}
			}(i, u)
		}
		wg.Wait()
	}
	return resp, string(bytes), nil
}

func (client *BingClient) getOnePage(ctx context.Context, query string, count, offset int) (p []WebPage, err error) {
	queryURL, err := url.Parse(Endpoint)
	if err != nil {
		return nil, err
	}
	q := queryURL.Query()
	q.Set("q", query)
	q.Set("count", strconv.Itoa(count))
	q.Set("mkt", client.options.mkt)
	q.Set("promote", client.options.promote)
	q.Set("answerCount", strconv.Itoa(client.options.answerCount))
	q.Set("offset", strconv.Itoa(offset))
	queryURL.RawQuery = q.Encode()
	queryfullURL := queryURL.String()
	// https://api.bing.microsoft.com/v7.0/search?answerCount=2&count=5&mkt=zh-CN&promote=News%2CWebpages&q=langchain&responseFilter=News%2C%20Webpages
	// https://api.bing.microsoft.com/v7.0/search?answerCount=2&count=5&mkt=zh-CN&promote=News%2CWebpages&q=langchain&responseFilter=News,Webpages
	// Note: The URL above will return a http 400 error, while the one below will not
	queryfullURL += fmt.Sprintf("&responseFilter=%s", client.options.responseFilter)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, queryfullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating bingSearch request failed: %w", err)
	}
	request.Header.Add(AuthHeaderKey, client.options.apiKey)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("bingSearch[%s] get error: %w", queryURL, err)
	}

	defer response.Body.Close()
	code := response.StatusCode
	resp := &RespData{}
	if err := json.NewDecoder(response.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("bingSearch parse json resp get err:%w, http status code:%d", err, code)
	}
	if resp.ErrorResp != nil {
		return nil, fmt.Errorf("bingSearch get error resp from bing server: http status code:%d message:%s, code:%s", code, resp.ErrorResp.Message, resp.ErrorResp.Code)
	}
	webpagesLen := len(resp.WebPages.Value)
	newsLen := len(resp.News.NewsValues)
	p = make([]WebPage, webpagesLen+newsLen)
	if webpagesLen > 0 {
		for i, v := range resp.WebPages.Value {
			v := v
			p[i] = WebPage{
				Title:       v.Name,
				Description: v.Snippet,
				URL:         v.URL,
			}
		}
	}
	if newsLen > 0 {
		for i, v := range resp.News.NewsValues {
			v := v
			p[i+webpagesLen] = WebPage{
				Title:       v.Name,
				Description: v.Description,
				URL:         v.URL,
			}
		}
	}
	klog.V(3).Infof("bingSearch query:%s TotalEstimatedMatches:%d count:%d offset:%d webpages: %#v", query, resp.WebPages.TotalEstimatedMatches, count, offset, p)
	return p, nil
}

func FormatResults(vals []WebPage) (res string) {
	for _, val := range vals {
		res += fmt.Sprintf("Title: %s\nDescription: %s\nURL: %s\nContent: %s\n\n", val.Title, val.Description, val.URL, val.Content)
	}
	return res
}
