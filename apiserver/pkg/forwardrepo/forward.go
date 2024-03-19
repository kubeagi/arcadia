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
	"fmt"
	"net/http"
)

const (
	HuggingFaceForward = "huggingface"
	ModelScopeForward  = "modelscope"
)

type option struct {
	baseURL string

	// The name of the file to be downloaded
	downloadFileName string

	modelID, revision, hfToken string

	// To access huggingface, you may need to configure a proxy.
	// The system proxy is used by default.
	transport *http.Transport
}

type Option func(*option)

func WithDownloadFile(fn string) Option {
	return func(o *option) {
		o.downloadFileName = fn
	}
}
func WithBaseURL(url string) Option {
	return func(o *option) {
		o.baseURL = url
	}
}

func WithModelID(modelID string) Option {
	return func(o *option) {
		o.modelID = modelID
	}
}

func WithRevision(revision string) Option {
	return func(o *option) {
		o.revision = revision
	}
}

func WithHFToken(hfToken string) Option {
	return func(o *option) {
		o.hfToken = hfToken
	}
}

func WithTransport(tp *http.Transport) Option {
	return func(o *option) {
		o.transport = tp
	}
}

type BranchTag struct {
	Name         string `json:"name"`
	TargetCommit string `json:"targetCommit"`
}

type Revision struct {
	Tags     []BranchTag `json:"tags"`
	Branches []BranchTag `json:"branches"`
}

type Forward interface {
	Summary(context.Context, ...Option) (string, error)
	Files(context.Context, ...Option) ([]string, error)
	Revisions(context.Context, ...Option) (Revision, error)
	DownloadFile(context.Context, ...Option) ([]byte, error)
}

func NewForward(forwardType string, opts ...Option) (Forward, error) {
	switch forwardType {
	case HuggingFaceForward:
		return NewHuggingFace(opts...), nil
	case ModelScopeForward:
		return NewModelScope(opts...), nil
	default:
		return nil, fmt.Errorf("unsupported repository type %s", forwardType)
	}
}
