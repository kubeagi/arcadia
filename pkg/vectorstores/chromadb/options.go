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
	"net/http"

	chroma "github.com/amikos-tech/chroma-go"
	"github.com/tmc/langchaingo/embeddings"
)

type option struct {
	host           string
	port           int
	embeddr        embeddings.Embedder
	collectionName string
	textKey        string
	distanceFunc   chroma.DistanceFunction
	transport      *http.Transport
}

type Option func(*option)

func WithHost(host string) Option {
	return func(o *option) {
		o.host = host
	}
}

func WithPort(port int) Option {
	return func(o *option) {
		o.port = port
	}
}

func WithEmbedder(e embeddings.Embedder) Option {
	return func(o *option) {
		o.embeddr = e
	}
}

func WithCollectionName(collectionName string) Option {
	return func(o *option) {
		o.collectionName = collectionName
	}
}

func WithTextKey(textKey string) Option {
	return func(o *option) {
		o.textKey = textKey
	}
}

func WithDistanceFunc(f chroma.DistanceFunction) Option {
	return func(o *option) {
		o.distanceFunc = f
	}
}

func WithTransport(tp *http.Transport) Option {
	return func(o *option) {
		o.transport = tp
	}
}
