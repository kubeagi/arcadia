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

	"k8s.io/klog/v2"
)

const (
	// https://huggingface.co/api/models/uint64/abc/refs
	revisionAPI = "%s/api/models/%s/refs"

	// https://huggingface.co/api/models/uint64/abc
	filesAPI = "%s/api/models/%s"

	// https://huggingface.co/uint64/abc/resolve/main/README.md
	downloadFileAPI = "%s/%s/resolve/%s/%s"

	huggingFaceBaseURL = "https://huggingface.co"

	readme          = "README.md"
	defaultRevision = "main"
)

type HuggingFace struct {
	option option
}

func NewHuggingFace(opts ...Option) Forward {
	h := &HuggingFace{option: option{}}
	WithBaseURL(huggingFaceBaseURL)(&h.option)
	WithRevision(defaultRevision)(&h.option)
	for _, opt := range opts {
		opt(&h.option)
	}
	return h
}

func (h *HuggingFace) buildClient() http.Client {
	c := http.Client{}
	if h.option.transport != nil {
		c.Transport = h.option.transport
	}
	return c
}

func (h *HuggingFace) do(ctx context.Context, api, method string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, api, nil)
	if err != nil {
		return nil, err
	}
	if h.option.hfToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", h.option.hfToken))
	}
	c := h.buildClient()
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (h *HuggingFace) Summary(ctx context.Context, opts ...Option) (string, error) {
	opts = append(opts, WithDownloadFile(readme))
	for _, opt := range opts {
		opt(&h.option)
	}
	r, err := h.DownloadFile(ctx)
	if err != nil {
		klog.Errorf("[HuggingFace:Summary] failed to get summary %s", err)
		return "", err
	}
	return string(r), nil
}

func (h *HuggingFace) Files(ctx context.Context, opts ...Option) ([]string, error) {
	for _, opt := range opts {
		opt(&h.option)
	}
	api := fmt.Sprintf(filesAPI, h.option.baseURL, h.option.modelID)
	klog.Infof("[HuggingFace:Files] Prepare to request model  %s's files %s", h.option.modelID, api)

	data, err := h.do(ctx, api, http.MethodGet)
	if err != nil {
		klog.Errorf("[HuggingFace:Files] failed to list model %s files error%s", h.option.modelID, err)
		return nil, err
	}
	var model Model
	if err := json.Unmarshal(data, &model); err != nil {
		klog.Errorf("[HuggingFace:Files] failed to parse responsebody error %s", err)
		return nil, err
	}
	files := make([]string, len(model.Siblings))
	for i := range model.Siblings {
		files[i] = model.Siblings[i].RFileName
	}
	return files, nil
}

func (h *HuggingFace) Revisions(ctx context.Context, opts ...Option) (Revision, error) {
	for _, opt := range opts {
		opt(&h.option)
	}

	api := fmt.Sprintf(revisionAPI, h.option.baseURL, h.option.modelID)
	klog.Infof("[HuggingFace::Revisions] prepare to get all the branches and tags through %s", api)

	r, err := h.do(ctx, api, http.MethodGet)
	if err != nil {
		klog.Errorf("[HuggingFace:Revisions] failed to get model %s's branches and tags", h.option.modelID)
		return Revision{}, err
	}
	var mr ModelRevision
	if err := json.Unmarshal(r, &mr); err != nil {
		klog.Errorf("[HuggingFace:Revisions] failed to parse responsebody error %s", err)
		return Revision{}, err
	}
	rev := Revision{Branches: make([]BranchTag, 0), Tags: make([]BranchTag, 0)}
	for _, tag := range mr.Tags {
		rev.Tags = append(rev.Tags, BranchTag{Name: tag.Name, TargetCommit: tag.TargetCommit})
	}
	for _, branch := range mr.Branches {
		rev.Branches = append(rev.Branches, BranchTag{Name: branch.Name, TargetCommit: branch.TargetCommit})
	}
	return rev, nil
}

func (h *HuggingFace) DownloadFile(ctx context.Context, opts ...Option) ([]byte, error) {
	for _, opt := range opts {
		opt(&h.option)
	}
	api := fmt.Sprintf(downloadFileAPI, h.option.baseURL,
		h.option.modelID, h.option.revision, h.option.downloadFileName)
	klog.Infof("[HuggingFace:DownloadFile] Prepare to download files from %s", api)

	data, err := h.do(ctx, api, http.MethodGet)
	if err != nil {
		klog.Errorf("[HuggingFace:DownloadFile] failed to downloadfile %s", err)
		return nil, err
	}
	return data, nil
}
