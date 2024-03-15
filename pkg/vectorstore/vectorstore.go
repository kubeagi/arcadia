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

package vectorstore

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/tmc/langchaingo/embeddings"
	lanchaingoschema "github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/chroma"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
)

var (
	ErrUnsupportedVectorStoreType = errors.New("unsupported vectorstore type")
)

func NewVectorStore(ctx context.Context, vs *arcadiav1alpha1.VectorStore, embedder embeddings.Embedder, collectionName string, c client.Client) (v vectorstores.VectorStore, finish func(), err error) {
	switch vs.Spec.Type() {
	case arcadiav1alpha1.VectorStoreTypeChroma:
		ops := []chroma.Option{
			chroma.WithChromaURL(vs.Spec.Endpoint.URL),
			chroma.WithDistanceFunction(vs.Spec.Chroma.DistanceFunction),
		}
		if embedder != nil {
			ops = append(ops, chroma.WithEmbedder(embedder))
		} else {
			ops = append(ops, chroma.WithOpenAiAPIKey("fake_key_just_for_chroma_heartbeat"))
		}
		if collectionName != "" {
			ops = append(ops, chroma.WithNameSpace(collectionName))
		}
		v, err = chroma.New(ops...)
	case arcadiav1alpha1.VectorStoreTypePGVector:
		v, finish, err = NewPGVectorStore(ctx, vs, c, embedder, collectionName)
	case arcadiav1alpha1.VectorStoreTypeUnknown:
		fallthrough
	default:
		err = ErrUnsupportedVectorStoreType
	}
	return v, finish, err
}

func RemoveCollection(ctx context.Context, log logr.Logger, vs *arcadiav1alpha1.VectorStore, collectionName string, c client.Client) (err error) {
	switch vs.Spec.Type() {
	case arcadiav1alpha1.VectorStoreTypeChroma:
		ops := []chroma.Option{
			chroma.WithChromaURL(vs.Spec.Endpoint.URL),
			chroma.WithDistanceFunction(vs.Spec.Chroma.DistanceFunction),
			chroma.WithOpenAiAPIKey("fake_key_just_for_chroma_heartbeat"),
		}
		if collectionName != "" {
			ops = append(ops, chroma.WithNameSpace(collectionName))
		}
		v, err := chroma.New(ops...)
		if err != nil {
			log.Error(err, "reconcile delete: init vector store error, may leave garbage data")
			return err
		}
		if err = v.RemoveCollection(); err != nil {
			log.Error(err, "reconcile delete: remove vector store error, may leave garbage data")
			return err
		}
	case arcadiav1alpha1.VectorStoreTypePGVector:
		v, finish, err := NewPGVectorStore(ctx, vs, c, nil, collectionName)
		defer func() {
			if finish != nil {
				finish()
			}
		}()
		if err != nil {
			log.Error(err, "reconcile delete: init pgvector error, may leave garbage data")
			return err
		}
		tx, err := v.Conn.Begin(ctx)
		if err != nil {
			log.Error(err, "reconcile delete: get tx error, may leave garbage data")
			return err
		}
		if err = v.RemoveCollection(ctx, tx); err != nil {
			log.Error(err, "reconcile delete: remove vector store error, may leave garbage data")
			return err
		}

	case arcadiav1alpha1.VectorStoreTypeUnknown:
		fallthrough
	default:
		err = ErrUnsupportedVectorStoreType
	}
	return err
}

func AddDocuments(ctx context.Context, log logr.Logger, vs *arcadiav1alpha1.VectorStore, embedder embeddings.Embedder, collectionName string, c client.Client, documents []lanchaingoschema.Document) (err error) {
	s, finish, err := NewVectorStore(ctx, vs, embedder, collectionName, c)
	if err != nil {
		return err
	}
	log.Info("handle file: add documents to embedder")
	if store, ok := s.(*PGVectorStore); ok {
		// now only pgvector support Row-level updates
		log.V(3).Info("handle file: use pgvector, filter out exist documents...")
		if documents, err = store.RemoveExist(ctx, log, documents); err != nil {
			return err
		}
		log.V(3).Info("handle file: use pgvector, filter out exist documents done")
	}
	for i, doc := range documents {
		log.V(5).Info(fmt.Sprintf("add doc to vectorstore, document[%d]: embedding:%s, metadata:%v", i, doc.PageContent, doc.Metadata))
	}
	log.V(3).Info("handle file: add documents, may take long time...")
	if _, err = s.AddDocuments(ctx, documents); err != nil {
		return err
	}
	log.V(3).Info("handle file: add documents done")
	if finish != nil {
		finish()
	}
	log.V(3).Info("handle file succeeded")
	return nil
}
