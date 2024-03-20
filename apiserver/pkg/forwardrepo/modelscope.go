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
package forwardrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"k8s.io/klog/v2"
)

const (
	modelScopeBaseURL = "https://modelscope.cn"

	summaryAPI   = "%s/api/v1/models/%s"
	revisionsAPI = "%s/api/v1/models/%s/revisions"
)

type ModelScope struct {
	option option
}

func NewModelScope(opts ...Option) Forward {
	m := &ModelScope{option: option{}}
	WithBaseURL(modelScopeBaseURL)(&m.option)
	for _, opt := range opts {
		opt(&m.option)
	}
	return m
}

func (m *ModelScope) buidClient() *http.Client {
	c := &http.Client{}
	if m.option.transport != nil {
		c.Transport = m.option.transport
	}
	return c
}

func (m *ModelScope) do(ctx context.Context, api, method string, query map[string]string) ([]byte, error) {
	values := url.Values{}
	for k, v := range query {
		values.Add(k, v)
	}
	if len(values) > 0 {
		api += fmt.Sprintf("?%s", values.Encode())
	}

	klog.Infof("[ModelScope:do] send request to %s", api)
	req, err := http.NewRequestWithContext(ctx, method, api, nil)
	if err != nil {
		return nil, err
	}
	resp, err := m.buidClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (m *ModelScope) Summary(ctx context.Context, opts ...Option) (string, error) {
	for _, opt := range opts {
		opt(&m.option)
	}
	api := fmt.Sprintf(summaryAPI, m.option.baseURL, m.option.modelID)
	q := make(map[string]string)
	if m.option.revision != "" {
		q["Revision"] = m.option.revision
	}
	klog.Infof("[ModelScope:Summary] prepare get model %s's summary from %s", m.option.modelID, api)

	data, err := m.do(ctx, api, http.MethodGet, q)
	if err != nil {
		klog.Errorf("[ModelScope:Summary] failed to get summary %s", err)
		return "", err
	}
	var ms ModelScopeSummary
	if err := json.Unmarshal(data, &ms); err != nil {
		return "", err
	}
	if ms.Code != 200 || !ms.Success {
		return "", fmt.Errorf("error message: %s", ms.Message)
	}
	return ms.Data.ReadMeContent, nil
}

// TODO: don't support
func (m *ModelScope) Files(ctx context.Context, opts ...Option) ([]string, error) {
	return nil, nil
}

func (m *ModelScope) Revisions(ctx context.Context, opts ...Option) (Revision, error) {
	for _, opt := range opts {
		opt(&m.option)
	}
	api := fmt.Sprintf(revisionsAPI, m.option.baseURL, m.option.modelID)
	klog.Infof("[ModelScope::Revisions] prepare to get all the branches and tags through %s", api)
	data, err := m.do(ctx, api, http.MethodGet, nil)
	if err != nil {
		klog.Errorf("[ModelScope:Revisions] failed to get model %s's branches and tags", m.option.modelID)
		return Revision{}, err
	}

	var revisions ModelScopeRevision
	if err := json.Unmarshal(data, &revisions); err != nil {
		klog.Errorf("[ModelScope:Revisions] failed to parse responsebody error %s", err)
		return Revision{}, err
	}
	if revisions.Code != 200 || !revisions.Success {
		return Revision{}, fmt.Errorf("error message: %s", revisions.Message)
	}
	r := Revision{}
	for _, b := range revisions.Data.RevisionMap.Branches {
		r.Branches = append(r.Branches, BranchTag{Name: b.Revision})
	}
	for _, t := range revisions.Data.RevisionMap.Tags {
		r.Tags = append(r.Tags, BranchTag{Name: t.Revision})
	}
	return r, nil
}

// TODO: don't support
func (m *ModelScope) DownloadFile(ctx context.Context, opts ...Option) ([]byte, error) {
	return nil, nil
}
