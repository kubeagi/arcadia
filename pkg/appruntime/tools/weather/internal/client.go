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
package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"k8s.io/klog/v2"
)

// Use the seniverse API to get the weather data
const _url = "https://api.seniverse.com/v3/weather/now.json"

type response struct {
	Results []struct {
		Location struct {
			ID             string `json:"id"`
			Name           string `json:"name"`
			Country        string `json:"country"`
			Path           string `json:"path"`
			Timezone       string `json:"timezone"`
			TimezoneOffset string `json:"timezone_offset"`
		} `json:"location"`
		Now struct {
			Text        string `json:"text"`
			Code        string `json:"code"`
			Temperature string `json:"temperature"`
		} `json:"now"`
		LastUpdate time.Time `json:"last_update"`
	} `json:"results"`
}

type Client struct {
	apiKey string
}

func New(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
	}
}

// input will be from the previous LLM
func (s *Client) GetData(ctx context.Context, input string) (string, error) {
	var cityInput map[string]interface{}
	klog.Infoln("input for weather api is", input)
	city := input
	err := json.Unmarshal([]byte(input), &cityInput)
	if err != nil {
		// Use the input directly if failed to parse
		klog.Errorln("failed to parse the input:", err)
	} else {
		// might be city/location/query
		if value, ok := cityInput["city"].(string); ok {
			city = value
		} else if value, ok := cityInput["location"].(string); ok {
			city = value
		} else if value, ok := cityInput["query"].(string); ok {
			city = value
		}
	}

	params := make(url.Values)
	params.Add("key", s.apiKey)
	params.Add("location", city)
	params.Add("language", "zh-Hans")
	params.Add("unit", "c")

	reqURL := fmt.Sprintf("%s?%s", _url, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request in weatherapi: %w", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("doing response in weatherapi: %w", err)
	}
	defer res.Body.Close()
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, res.Body)
	if err != nil {
		return "", fmt.Errorf("coping data in weatherapi: %w", err)
	}
	klog.Infof("result from weather API: %s", buf.String())
	result := response{}
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		return "", fmt.Errorf("unmarshal data from weatherapi: %w", err)
	}
	return getWeahterDescription(result), nil
}

// return the description of weather info
func getWeahterDescription(result response) string {
	// the response should be in the format '{"results":[{"location":{"id":"WTW3SJ5ZBJUY","name":"上海","country":"CN","path":"上海,上海,中国","timezone":"Asia/Shanghai",
	// "timezone_offset":"+08:00"},"now":{"text":"多云","code":"4","temperature":"9"},"last_update":"2024-01-18T22:27:50+08:00"}]}'
	if len(result.Results) > 0 {
		return fmt.Sprintf("the weather situation: %s, temperature: %s, update time: %s",
			result.Results[0].Now.Text, result.Results[0].Now.Temperature, result.Results[0].LastUpdate)
	}
	return "No weather data available"
}
