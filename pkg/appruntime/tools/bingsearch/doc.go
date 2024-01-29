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

/*

Package bingsearch based on [bing official bing-web-api-v7 search API](https://learn.microsoft.com/zh-cn/rest/api/cognitiveservices-bingsearch/bing-web-api-v7-reference),
implements the function of bing search using standard apikey, at the same time adapted `github.com/tmc/langchaingo/tools.Tool` interface,
convenient to use in langchiango agent directly.

you can create an apikey by https://portal.azure.com/#create/Microsoft.BingSearch
*/

package bingsearch
