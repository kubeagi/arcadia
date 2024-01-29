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
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kubeagi/arcadia/api/app-node/agent/v1alpha1"
)

func TestBingSearchTool(t *testing.T) {
	t.Parallel()
	apikey := os.Getenv("BING_KEY")
	if apikey == "" {
		t.Skip("Must set BING_KEY to run TestBingSearchTool")
	}
	rightTool := &v1alpha1.Tool{
		Params: map[string]string{
			"apiKey": apikey,
		},
	}
	tool, _ := New(rightTool)
	resp, err := tool.Call(context.Background(), "langchain")
	require.NoError(t, err)
	t.Logf("get format resp:\n%s", resp)

	wrongTool := rightTool
	wrongTool.Params["apiKey"] = "xxxxx"
	tool, _ = New(wrongTool)
	_, err = tool.Call(context.Background(), "langchain")
	t.Logf("should get err:\n%s", err)
	require.Error(t, err)
}

func TestBingSearchClient(t *testing.T) {
	t.Parallel()
	apikey := os.Getenv("BING_KEY")
	if apikey == "" {
		t.Skip("Must set BING_KEY to run TestBingSearchClient")
	}
	client := NewBingClient(WithAPIKey(apikey))
	p, _, err := client.SearchGetDetailData(context.Background(), "langchain")
	require.NoError(t, err)
	require.Equal(t, defaultOptions().count, len(p))
	for i, _p := range p {
		t.Logf("get format resp[%d]:\n%#v", i, _p)
	}
	// more count
	client = NewBingClient(WithAPIKey(apikey), WithCount(100))
	p, _, err = client.SearchGetDetailData(context.Background(), "langchain")
	require.NoError(t, err)
	require.Equal(t, 100, len(p))
	for i, _p := range p {
		t.Logf("get format resp[%d]:\n%#v", i, _p)
	}
}
