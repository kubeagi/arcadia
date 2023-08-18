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

// NOTE: Reference zhipuai's python sdk: model_api/api.py

package zhipuai

import (
	"errors"
	"fmt"
	"time"
)

const (
	ZHIPUAI_MODEL_API_URL         = "https://open.bigmodel.cn/api/paas/v3/model-api"
	ZHIPUAI_MODEL_Default_Timeout = 300 * time.Second
)

type Model string

const (
	ZhiPuAILite Model = "chatglm_lite"
	ZhiPuAIStd  Model = "chatglm_std"
	ZhiPuAIPro  Model = "chatglm_pro"
)

type Method string

const (
	// POST
	ZhiPuAIInvoke      Method = "invoke"
	ZhiPuAIAsyncInvoke Method = "async-invoke"
	ZhiPuAISSEInvoke   Method = "sse-invoke"
	// GET
	ZhiPuAIAsyncGet Method = "async-get"
)

func BuildAPIURL(model Model, method Method) string {
	return fmt.Sprintf("%s/%s/%s", ZHIPUAI_MODEL_API_URL, model, method)
}

type ZhiPuAI struct {
	apiKey string
}

func NewZhiPuAI(apiKey string) *ZhiPuAI {
	return &ZhiPuAI{
		apiKey: apiKey,
	}
}

// Invoke calls zhipuai and returns result immediately
func (z *ZhiPuAI) Invoke(params ModelParams) (map[string]interface{}, error) {
	url := BuildAPIURL(params.Model, ZhiPuAIInvoke)
	token, err := GenerateToken(z.apiKey, API_TOKEN_TTL_SECONDS)
	if err != nil {
		return nil, err
	}

	return Post(url, token, params, ZHIPUAI_MODEL_Default_Timeout)
}

// AsyncInvoke only returns a task id which can be used to get result of task later
func (z *ZhiPuAI) AsyncInvoke(params ModelParams) (map[string]interface{}, error) {
	url := BuildAPIURL(params.Model, ZhiPuAIAsyncInvoke)
	token, err := GenerateToken(z.apiKey, API_TOKEN_TTL_SECONDS)
	if err != nil {
		return nil, err
	}

	return Post(url, token, params, ZHIPUAI_MODEL_Default_Timeout)
}

// Get result of task async-invoke
func (z *ZhiPuAI) Get(params ModelParams) (map[string]interface{}, error) {
	if params.TaskID == "" {
		return nil, errors.New("TaskID is required when running Get with method AsyncInvoke")
	}

	// url with task id
	url := fmt.Sprintf("%s/%s", BuildAPIURL(params.Model, ZhiPuAIAsyncInvoke), params.TaskID)
	token, err := GenerateToken(z.apiKey, API_TOKEN_TTL_SECONDS)
	if err != nil {
		return nil, err
	}

	return Get(url, token, ZHIPUAI_MODEL_Default_Timeout)
}
