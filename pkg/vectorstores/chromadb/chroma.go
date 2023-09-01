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
	"fmt"
	"net/http"

	chroma "github.com/amikos-tech/chroma-go"
	openapiclient "github.com/amikos-tech/chroma-go/swagger"
	"github.com/tmc/langchaingo/schema"
	vs "github.com/tmc/langchaingo/vectorstores"
)

type chromadb struct {
	option *option
	client *chroma.Client
}

func NewChromaDB(opts ...Option) (vs.VectorStore, error) {
	store := &chromadb{option: &option{host: "http://localhost", port: 8000, textKey: "text", distanceFunc: chroma.L2}}
	for _, opt := range opts {
		opt(store.option)
	}
	if err := store.verify(); err != nil {
		return nil, err
	}
	c := http.DefaultClient
	if store.option.transport != nil {
		c = &http.Client{
			Transport: store.option.transport,
		}
	}
	cfg := openapiclient.Configuration{
		Servers: openapiclient.ServerConfigurations{
			{
				URL:         fmt.Sprintf("%s:%d", store.option.host, store.option.port),
				Description: "chromadb server",
			},
		},
		HTTPClient: c,
	}

	store.client = &chroma.Client{ApiClient: openapiclient.NewAPIClient(&cfg)}
	return store, nil
}

func (c *chromadb) verify() error {
	if c.option.collectionName == "" {
		return fmt.Errorf("collectioName can't be empty")
	}
	if c.option.embeddr == nil {
		return fmt.Errorf("embedder is empty")
	}

	return nil
}

func (c *chromadb) getOptions(options ...vs.Option) vs.Options {
	opts := vs.Options{}
	for _, opt := range options {
		opt(&opts)
	}
	return opts
}

// find where and where documents
func (c *chromadb) getFilter(opts vs.Options) (map[string]interface{}, map[string]interface{}) {
	mustBeArray, ok := opts.Filters.([]interface{})
	if !ok {
		return nil, nil
	}
	if len(mustBeArray) != 2 {
		return nil, nil
	}
	a, oka := mustBeArray[0].(map[string]interface{})
	b, okb := mustBeArray[1].(map[string]interface{})
	if oka && okb {
		return a, b
	}

	return nil, nil
}

func (c *chromadb) addDocuments(ctx context.Context, texts, ids []string, metadatas []map[string]interface{}) error {
	localEmbedder := &LocalEmbedder{Embedder: c.option.embeddr}
	collection, err := c.client.CreateCollection(c.option.collectionName, map[string]interface{}{}, true, localEmbedder, c.option.distanceFunc)
	if err != nil {
		return err
	}
	vectors, err := localEmbedder.CreateEmbedding(texts)
	if err != nil {
		return err
	}
	if len(vectors) != len(texts) {
		return fmt.Errorf("number of vectors from embedder does not match number of documents")
	}
	_, err = collection.Add(vectors, metadatas, texts, ids)
	return err
}

func (c *chromadb) AddDocuments(ctx context.Context, docs []schema.Document, options ...vs.Option) error {
	texts := make([]string, 0, len(docs))
	ids := make([]string, len(docs))
	for idx, doc := range docs {
		texts = append(texts, doc.PageContent)
		ids[idx] = fmt.Sprintf("%d", idx)
	}

	metadatas := make([]map[string]interface{}, 0)
	for i := 0; i < len(docs); i++ {
		metadata := make(map[string]interface{})
		for k, v := range docs[i].Metadata {
			metadata[k] = v
		}
		metadata[c.option.textKey] = texts[i]
		metadatas = append(metadatas, metadata)
	}
	return c.addDocuments(ctx, texts, ids, metadatas)
}

func (c *chromadb) SimilaritySearch(ctx context.Context, query string, numDocuments int, options ...vs.Option) ([]schema.Document, error) {
	localEmbedder := &LocalEmbedder{Embedder: c.option.embeddr}
	collection, err := c.client.GetCollection(c.option.collectionName, localEmbedder)
	if err != nil {
		return nil, err
	}
	opts := c.getOptions(options...)
	where, whereDocument := c.getFilter(opts)
	result, err := collection.Query([]string{query}, int32(numDocuments), where, whereDocument, nil)
	if err != nil {
		return nil, err
	}

	dl := len(result.Documents[0])
	documents := make([]schema.Document, dl)
	// {"ids":[["0001","0003"]],"distances":[[0.0005762028712337219,0.0005762028712337219]],"metadatas":[[{"chapter":"3","verse":"16"},{"chapter":"29","verse":"11"}]],"embeddings":null,"documents":[["doc1","doc3"]]}
	for i := 0; i < dl; i++ {
		doc := schema.Document{
			Metadata:    result.Metadatas[0][i],
			PageContent: result.Documents[0][i],
		}
		documents[i] = doc
	}

	return documents, nil
}
