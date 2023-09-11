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

	chroma "github.com/amikos-tech/chroma-go"
	"github.com/tmc/langchaingo/embeddings"
)

const (
	_defaultNameSpaceKey = "nameSpace"
	_defaultTextKey      = "text"
	_defaultNameSpace    = "langchain"
	_defualtDistanceFunc = chroma.L2
)

// ErrInvalidOptions is returned when the options given are invalid.
var ErrInvalidOptions = errors.New("invalid options")

type Option func(p *Store)

// WithEmbedder is an option for setting the embedder to use.Must be set.
func WithEmbedder(e embeddings.Embedder) Option {
	return func(p *Store) {
		p.embedder.Embedder = e
	}
}

// WithTextKey is an option for setting the text key in the metadata to the vectors
// in the index. The text key stores the text of the document the vector represents.
func WithTextKey(textKey string) Option {
	return func(p *Store) {
		p.textKey = textKey
	}
}

// WithNameSpaceKey is an option for setting the nameSpace key in the metadata to the vectors
// in the index. The nameSpace key stores the nameSpace of the document the vector represents.
// In chromadb, namespace represents the collection.
func WithNameSpaceKey(nameSpaceKey string) Option {
	return func(p *Store) {
		p.nameSpaceKey = nameSpaceKey
	}
}

// WithScheme is an option for setting the scheme of the chromadb server.Must be set.
func WithURL(url string) Option {
	return func(p *Store) {
		p.url = url
	}
}

// WithNameSpace is an option for setting the nameSpace to upsert and query the vectors.
// In chromadb, namespace represents the collection.
func WithNameSpace(nameSpace string) Option {
	return func(p *Store) {
		p.nameSpace = nameSpace
	}
}

func WithDistanceFunc(f chroma.DistanceFunction) Option {
	return func(p *Store) {
		p.distanceFunc = f
	}
}

// WithIncludes is an option for setting the includes to query the vectors.
func WithIncludes(includes []chroma.QueryEnum) Option {
	return func(p *Store) {
		p.includes = includes
	}
}

func applyClientOptions(opts ...Option) (Store, error) {
	o := &Store{
		textKey:      _defaultTextKey,
		nameSpaceKey: _defaultNameSpaceKey,
		nameSpace:    _defaultNameSpace,
		distanceFunc: _defualtDistanceFunc,
	}

	for _, opt := range opts {
		opt(o)
	}

	if o.url == "" {
		return Store{}, fmt.Errorf("%w: missing url", ErrInvalidOptions)
	}

	if o.embedder.Embedder == nil {
		return Store{}, fmt.Errorf("%w: missing embedder", ErrInvalidOptions)
	}

	return *o, nil
}
