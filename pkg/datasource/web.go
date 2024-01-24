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
package datasource

import (
	"context"
	"io"
	"net/url"
)

var _ Datasource = (*Web)(nil)

type Web struct {
	url string
}

func NewWeb(ctx context.Context, url string) (*Web, error) {
	return &Web{
		url: url,
	}, nil
}

func (w *Web) Stat(ctx context.Context, info any) error {
	_, err := url.ParseRequestURI("http://google.com/")
	if err != nil {
		return err
	}
	return nil
}

func (w *Web) Remove(ctx context.Context, info any) error {
	return nil
}

func (w *Web) ReadFile(ctx context.Context, info any) (io.ReadCloser, error) {
	// TODO implement me
	panic("implement me")
}

func (w *Web) StatFile(ctx context.Context, info any) (any, error) {
	// TODO implement me
	panic("implement me")
}

func (w *Web) GetTags(ctx context.Context, info any) (map[string]string, error) {
	// TODO implement me
	panic("implement me")
}

func (w *Web) ListObjects(ctx context.Context, source string, info any) (any, error) {
	// TODO implement me
	panic("implement me")
}
