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

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/apiserver/graph/generated"
)

var (
	once sync.Once
	url  string
)

func Init(dataprocessingURL string) {
	once.Do(func() {
		if dataprocessingURL != "" {
			url = dataprocessingURL
		}
	})
}

func ListDataprocessing(_ context.Context, _ client.Client, _ *generated.DataProcessQuery, input *generated.AllDataProcessListByPageInput) (*generated.PaginatedDataProcessItem, error) {
	// prepare http request
	jsonParams, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url+"/list-by-page", bytes.NewBuffer(jsonParams))
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

func ListDataprocessingByCount(_ context.Context, _ client.Client, _ *generated.DataProcessQuery, input *generated.AllDataProcessListByCountInput) (*generated.CountDataProcessItem, error) {
	// prepare http request
	jsonParams, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url+"/list-by-count", bytes.NewBuffer(jsonParams))
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
	countData := &generated.CountDataProcessItem{}
	err = json.NewDecoder(resp.Body).Decode(countData)
	if err != nil {
		return nil, err
	}
	return countData, nil
}

func DataProcessSupportType(_ context.Context, _ client.Client, _ *generated.DataProcessQuery) (*generated.DataProcessSupportType, error) {
	// prepare http request
	req, err := http.NewRequest("POST", url+"/text-process-type", nil)
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
	data := &generated.DataProcessSupportType{}
	err = json.NewDecoder(resp.Body).Decode(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func CreateDataProcessTask(_ context.Context, _ client.Client, _ *generated.DataProcessMutation, input *generated.AddDataProcessInput) (*generated.DataProcessResponse, error) {
	// create complete http payload to data processing service

	// prepare http request
	jsonParams, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url+"/add", bytes.NewBuffer(jsonParams))
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
	data := &generated.DataProcessResponse{}
	err = json.NewDecoder(resp.Body).Decode(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func DeleteDataProcessTask(_ context.Context, _ client.Client, _ *generated.DataProcessMutation, input *generated.DeleteDataProcessInput) (*generated.DataProcessResponse, error) {
	// prepare http request
	jsonParams, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url+"/delete-by-id", bytes.NewBuffer(jsonParams))
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
	data := &generated.DataProcessResponse{}
	err = json.NewDecoder(resp.Body).Decode(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func DataProcessDetails(_ context.Context, _ client.Client, _ *generated.DataProcessQuery, input *generated.DataProcessDetailsInput) (*generated.DataProcessDetails, error) {
	// prepare http request
	jsonParams, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url+"/info-by-id", bytes.NewBuffer(jsonParams))
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
	data := &generated.DataProcessDetails{}
	err = json.NewDecoder(resp.Body).Decode(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func CheckDataProcessTaskName(_ context.Context, _ client.Client, _ *generated.DataProcessQuery, input *generated.CheckDataProcessTaskNameInput) (*generated.DataProcessResponse, error) {
	// prepare http request
	jsonParams, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url+"/check-task-name", bytes.NewBuffer(jsonParams))
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
	data := &generated.DataProcessResponse{}
	err = json.NewDecoder(resp.Body).Decode(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func GetLogInfo(_ context.Context, _ client.Client, _ *generated.DataProcessQuery, input *generated.DataProcessDetailsInput) (*generated.DataProcessResponse, error) {
	// prepare http request
	jsonParams, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url+"/get-log-info", bytes.NewBuffer(jsonParams))
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
	data := &generated.DataProcessResponse{}
	err = json.NewDecoder(resp.Body).Decode(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func DataProcessLogInfoByFileName(_ context.Context, _ client.Client, _ *generated.DataProcessQuery, input *generated.DataProcessFileLogInput) (*generated.DataProcessResponse, error) {
	// prepare http request
	jsonParams, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url+"/get-log-by-file-name", bytes.NewBuffer(jsonParams))
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
	data := &generated.DataProcessResponse{}
	err = json.NewDecoder(resp.Body).Decode(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func DataProcessRetry(_ context.Context, _ client.Client, _ *generated.DataProcessQuery, input *generated.DataProcessRetryInput) (*generated.DataProcessResponse, error) {
	// prepare http request
	jsonParams, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", url+"/retry", bytes.NewBuffer(jsonParams))
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
	data := &generated.DataProcessResponse{}
	err = json.NewDecoder(resp.Body).Decode(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}
