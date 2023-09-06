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
	"context"
	"errors"
	"fmt"
	"net/http"

	chroma "github.com/amikos-tech/chroma-go"
	chromaopenapi "github.com/amikos-tech/chroma-go/swagger"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
)

var (
	// ErrEmbedderWrongNumberVectors is returned when if the embedder returns a number
	// of vectors that is not equal to the number of documents given.
	ErrEmbedderWrongNumberVectors = errors.New(
		"number of vectors from embedder does not match number of documents",
	)
	ErrInvalidScoreThreshold = errors.New(
		"score threshold must be between 0 and 1")
)

type Store struct {
	embedder wrappedEmbeddingFunction
	client   *chroma.Client

	namespaceKey string
	textKey      string

	nameSpace string

	basePath string

	distanceFunc chroma.DistanceFunction
	transport    *http.Transport
}

var _ vectorstores.VectorStore = Store{}

func New(ctx context.Context, opts ...Option) (vectorstores.VectorStore, error) {
	s, err := applyClientOptions(opts...)
	if err != nil {
		return nil, err
	}

	configuration := chromaopenapi.NewConfiguration()
	configuration.Servers = chromaopenapi.ServerConfigurations{
		{
			URL:         s.basePath,
			Description: "Chromadb server url for this store",
		},
	}
	s.client = &chroma.Client{
		ApiClient: chromaopenapi.NewAPIClient(configuration),
	}

	return s, nil
}

func (s Store) AddDocuments(ctx context.Context, docs []schema.Document, options ...vectorstores.Option) error {
	opts := s.getOptions(options...)
	nameSpace := s.getNameSpace(opts)

	texts := make([]string, 0, len(docs))
	ids := make([]string, len(docs))
	for idx, doc := range docs {
		texts = append(texts, doc.PageContent)
		ids[idx] = fmt.Sprintf("%d", idx)
	}

	collection, err := s.client.CreateCollection(s.nameSpace, map[string]interface{}{}, true, s.embedder, s.distanceFunc)
	if err != nil {
		return err
	}

	vectors, err := s.embedder.CreateEmbedding(texts)
	if err != nil {
		return err
	}
	if len(vectors) != len(texts) {
		return ErrEmbedderWrongNumberVectors
	}

	metadatas := make([]map[string]any, 0)
	for i := 0; i < len(docs); i++ {
		metadata := make(map[string]any)
		for k, v := range docs[i].Metadata {
			metadata[k] = v
		}
		metadata[s.textKey] = texts[i]
		metadata[s.namespaceKey] = nameSpace

		metadatas = append(metadatas, metadata)
	}

	if _, err = collection.Add(vectors, metadatas, texts, ids); err != nil {
		return err
	}

	return err
}

func (s Store) SimilaritySearch(
	ctx context.Context,
	query string,
	numDocuments int,
	options ...vectorstores.Option,
) ([]schema.Document, error) {
	opts := s.getOptions(options...)
	nameSpace := s.getNameSpace(opts)

	// FIXME: use scoreThreshold
	// scoreThreshold, err := s.getScoreThreshold(opts)
	// if err != nil {
	// 	return nil, err
	// }

	collection, err := s.client.GetCollection(nameSpace, s.embedder)
	if err != nil {
		return nil, err
	}

	where, whereDocument := s.getFilters(opts)
	result, err := collection.Query([]string{query}, int32(numDocuments), where, whereDocument, nil)
	if err != nil {
		return nil, err
	}

	dl := len(result.Documents[0])
	documents := make([]schema.Document, dl)
	for i := 0; i < dl; i++ {
		doc := schema.Document{
			Metadata:    result.Metadatas[0][i],
			PageContent: result.Documents[0][i],
		}
		documents[i] = doc
	}

	return documents, nil
}

func (s Store) getNameSpace(opts vectorstores.Options) string {
	if opts.NameSpace != "" {
		return opts.NameSpace
	}
	return s.nameSpace
}

// func (s Store) getScoreThreshold(opts vectorstores.Options) (float32, error) {
// 	if opts.ScoreThreshold < 0 || opts.ScoreThreshold > 1 {
// 		return 0, ErrInvalidScoreThreshold
// 	}
// 	f32 := float32(opts.ScoreThreshold)
// 	return f32, nil
// }

func (s Store) getFilters(opts vectorstores.Options) (where, whereDocument map[string]any) {
	filters, ok := opts.Filters.([]map[string]any)
	if !ok || len(filters) != 2 {
		return nil, nil
	}
	return filters[0], filters[1]
}

func (s Store) getOptions(options ...vectorstores.Option) vectorstores.Options {
	opts := vectorstores.Options{}
	for _, opt := range options {
		opt(&opts)
	}
	return opts
}
