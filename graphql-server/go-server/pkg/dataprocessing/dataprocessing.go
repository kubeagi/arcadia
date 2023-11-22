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

package dataprocessing

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/kubeagi/arcadia/graphql-server/go-server/graph/generated"
)

var (
	once sync.Once
	url  string
)

func Init(dataprocessingURL string) {
	once.Do(func() {
		url = dataprocessingURL
	})
}

func ListDataprocessing(ctx context.Context, obj *generated.DataProcessQuery, input *generated.AllDataProcessListByPageInput) (*generated.PaginatedDataProcessItem, error) {
	// prepare http request
	jsonParams, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonParams))
	if err != nil {
		return nil, err
	}

	// call dataprocessing server
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// parse http response
	pagedData := &generated.PaginatedDataProcessItem{}
	err = json.NewDecoder(resp.Body).Decode(pagedData)
	if err != nil {
		return nil, err
	}
	return pagedData, nil
}
