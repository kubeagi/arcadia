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
	"strconv"

	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/tools"
	"k8s.io/klog/v2"

	"github.com/kubeagi/arcadia/api/app-node/agent/v1alpha1"
)

const (
	ToolName         = "Bing Search API"
	ParamAPIKey      = "apiKey"
	ParamCount       = "count"
	ParamScraperPage = "scraperPage"
)

type Tool struct {
	client           *BingClient
	CallbacksHandler callbacks.Handler
}

var _ tools.Tool = Tool{}

// New creates a new bing search tool to search on internet
func New(tool *v1alpha1.Tool) (*Tool, error) {
	client, err := NewFromToolSpec(tool)
	return &Tool{client: client}, err
}

func NewFromToolSpec(tool *v1alpha1.Tool) (client *BingClient, err error) {
	var countVal int
	apikey := tool.Params[ParamAPIKey]
	count, ok := tool.Params[ParamCount]
	if ok {
		atoi, err := strconv.Atoi(count)
		if err != nil {
			return nil, err
		}
		countVal = atoi
	}
	scraperPage := true
	p, ok := tool.Params[ParamScraperPage]
	if ok {
		scraperPage, err = strconv.ParseBool(p)
		if err != nil {
			return nil, err
		}
	}
	return NewBingClient(WithAPIKey(apikey), WithCount(countVal), WithScraperPage(scraperPage)), nil
}

func (t Tool) Name() string {
	return ToolName
}

func (t Tool) Description() string {
	return "Invoke API to get the realtime bing search data."
}

func (t Tool) Call(ctx context.Context, input string) (string, error) {
	klog.Infof("running tool %s", ToolName)
	if t.CallbacksHandler != nil {
		t.CallbacksHandler.HandleToolStart(ctx, input)
	}
	result, err := t.client.Search(ctx, input)
	if err != nil {
		if t.CallbacksHandler != nil {
			t.CallbacksHandler.HandleToolError(ctx, err)
		}
		return "", err
	}
	if t.CallbacksHandler != nil {
		t.CallbacksHandler.HandleToolEnd(ctx, input)
	}
	return result, nil
}

type WebPage struct {
	Title       string
	Description string
	URL         string
	Content     string
}
