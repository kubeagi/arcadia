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

package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/tmc/langchaingo/tools"
	"github.com/tmc/langchaingo/tools/scraper"
	"k8s.io/klog/v2"

	"github.com/kubeagi/arcadia/api/app-node/agent/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/appruntime/log"
	"github.com/kubeagi/arcadia/pkg/tools/bingsearch"
	"github.com/kubeagi/arcadia/pkg/tools/weather"
)

func InitTools(ctx context.Context, specTools []v1alpha1.Tool) []tools.Tool {
	logger := klog.FromContext(ctx)
	allowedTools := make([]tools.Tool, 0, len(specTools))
	for _, toolSpec := range specTools {
		switch toolSpec.Name {
		case bingsearch.ToolName:
			client, err := bingsearch.New(&toolSpec)
			if err != nil {
				logger.Error(err, "failed to create a new bingsearch tool")
				continue
			}
			client.CallbacksHandler = log.KLogHandler{LogLevel: 3}
			allowedTools = append(allowedTools, client)
		case weather.ToolName:
			tool, err := weather.New(&toolSpec)
			if err != nil {
				logger.Error(err, "failed to create a new weather tool")
				continue
			}
			tool.CallbacksHandler = log.KLogHandler{LogLevel: 3}
			allowedTools = append(allowedTools, tool)
		case tools.Calculator{}.Name():
			tool := tools.Calculator{}
			tool.CallbacksHandler = log.KLogHandler{LogLevel: 3}
			allowedTools = append(allowedTools, tool)
		case scraper.Scraper{}.Name():
			// prepare options from toolSpec
			options := make([]scraper.Options, 0)
			if toolSpec.Params["delay"] != "" {
				delay, err := strconv.ParseInt(toolSpec.Params["delay"], 10, 64)
				if err != nil {
					logger.Error(err, fmt.Sprintf("failed to parse delay %s", toolSpec.Params["delay"]))
				} else {
					options = append(options, scraper.WithDelay(delay))
				}
			}
			if toolSpec.Params["async"] != "" {
				async, err := strconv.ParseBool(toolSpec.Params["async"])
				if err != nil {
					logger.Error(err, fmt.Sprintf("failed to parse async %s", toolSpec.Params["async"]))
				} else {
					options = append(options, scraper.WithAsync(async))
				}
			}
			if toolSpec.Params["handleLinks"] != "" {
				handleLinks, err := strconv.ParseBool(toolSpec.Params["handleLinks"])
				if err != nil {
					logger.Error(err, fmt.Sprintf("failed to parse handleLinks %s", toolSpec.Params["handleLinks"]))
				} else {
					options = append(options, scraper.WithHandleLinks(handleLinks))
				}
			}
			if toolSpec.Params["blacklist"] != "" {
				blacklistArray := strings.Split(toolSpec.Params["blacklist"], ",")
				options = append(options, scraper.WithBlacklist(blacklistArray))
			}
			if toolSpec.Params["maxScrapedDataLength"] != "" {
				maxScrapedDataLength, err := strconv.Atoi(toolSpec.Params["maxScrapedDataLength"])
				if err != nil {
					klog.Errorln("failed to parse maxScrapedDataLength %s", toolSpec.Params["maxScrapedDataLength"])
				} else {
					options = append(options, scraper.WithMaxScrapedDataLength(maxScrapedDataLength))
				}
			}
			tool, err := scraper.New(options...)
			if err != nil {
				logger.Error(err, "failed to create a new scraper tool")
				continue
			}
			allowedTools = append(allowedTools, tool)
		default:
			// Just continue if the tool does not exist
			klog.Errorf("no tool found with name: %s", toolSpec.Name)
		}
	}
	return allowedTools
}

// FIXME: should add web reference into chat result
// func RunTools(ctx context.Context, args map[string]any, ts []tools.Tool) map[string]any {
//	if len(ts) == 0 {
//		return args
//	}
//	input, ok := args["question"].(string)
//	if !ok {
//		return args
//	}
//	logger := klog.FromContext(ctx)
//	logger.V(3).Info(fmt.Sprintf("tools call input: %s", input))
//	result := make([]string, len(ts))
//	resultRef := make([][]retriever.Reference, len(ts))
//	for i := range resultRef {
//		resultRef[i] = make([]retriever.Reference, 0)
//	}
//	var wg sync.WaitGroup
//	wg.Add(len(ts))
//	for i, tool := range ts {
//		i, tool := i, tool
//		go func(i int, tool tools.Tool) {
//			defer wg.Done()
//
//			switch tool.Name() { // nolint:gocritic
//			case bingsearch.ToolName:
//				bingtool, ok := tool.(*bingsearch.Tool)
//				if !ok {
//					logger.Error(errors.New("failed to convert tool to bingsearch tool"), "")
//					return
//				}
//				data, _, err := bingtool.Client.SearchGetDetailData(ctx, input)
//				if err != nil {
//					logger.Error(err, "failed to call bing search tool")
//					return
//				}
//				ref := make([]retriever.Reference, len(data))
//				for j := range data {
//					ref[j] = retriever.Reference{
//						Title:   data[j].Title,
//						Content: data[j].Description,
//						URL:     data[j].URL,
//					}
//				}
//				resultRef[i] = ref
//				result[i] = bingsearch.FormatResults(data)
//			default:
//				data, err := tool.Call(ctx, input)
//				if err != nil {
//					logger.Error(err, "failed to call tool")
//					return
//				}
//				result[i] = data
//			}
//			klog.V(3).Info("tools call done")
//		}(i, tool)
//	}
//	wg.Wait()
//	res := make([]string, 0, len(result))
//	for i := range result {
//		if s := strings.TrimSpace(result[i]); s != "" {
//			res = append(res, s)
//		}
//	}
//	toolOut := strings.Join(res, "\n")
//	old, exist := args["context"]
//	if exist {
//		toolOut = old.(string) + "\n" + toolOut
//	}
//	args["context"] = toolOut
//	for i := range resultRef {
//		args = retriever.AddReferencesToArgs(args, resultRef[i])
//	}
//	return args
//}
