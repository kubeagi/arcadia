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
package chromadb

import (
	"errors"
	"fmt"
	"net/http"

	chroma "github.com/amikos-tech/chroma-go"
	"github.com/tmc/langchaingo/embeddings"
)

const (
	_defaultNameSpaceKey = "nameSpace"
	_defaultTextKey      = "text"
	_defaultNameSpace    = "default"
	_defualtDistanceFunc = chroma.L2
)

// ErrInvalidOptions is returned when the options given are invalid.
var ErrInvalidOptions = errors.New("invalid options")

type Option func(p *Store)

func WithBasePath(basePath string) Option {
	return func(c *Store) {
		c.basePath = basePath
	}
}

func WithEmbedder(e embeddings.Embedder) Option {
	return func(c *Store) {
		c.embedder.Embedder = e
	}
}

func WithNameSpace(nameSpace string) Option {
	return func(c *Store) {
		c.nameSpace = nameSpace
	}
}

func WithTextKey(textKey string) Option {
	return func(c *Store) {
		c.textKey = textKey
	}
}

func WithNameSpaceKey(nameSpaceKey string) Option {
	return func(c *Store) {
		c.namespaceKey = nameSpaceKey
	}
}

func WithDistanceFunc(f chroma.DistanceFunction) Option {
	return func(c *Store) {
		c.distanceFunc = f
	}
}

func WithTransport(tp *http.Transport) Option {
	return func(c *Store) {
		c.transport = tp
	}
}

func applyClientOptions(opts ...Option) (Store, error) {
	s := &Store{
		textKey:      _defaultTextKey,
		namespaceKey: _defaultNameSpaceKey,
		nameSpace:    _defaultNameSpace,

		distanceFunc: _defualtDistanceFunc,
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.basePath == "" {
		return Store{}, fmt.Errorf("%w: missing url", ErrInvalidOptions)
	}

	if s.embedder.Embedder == nil {
		return Store{}, fmt.Errorf("%w: missing embedder", ErrInvalidOptions)
	}

	return *s, nil
}
