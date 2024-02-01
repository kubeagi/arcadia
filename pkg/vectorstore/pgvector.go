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
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	"github.com/jackc/pgx/v5"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
	lanchaingoschema "github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/pgvector"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/datasource"
	"github.com/kubeagi/arcadia/pkg/utils"
)

var _ vectorstores.VectorStore = (*PGVectorStore)(nil)

type PGVectorStore struct {
	*pgx.Conn
	pgvector.Store
	*arcadiav1alpha1.PGVector
}

func NewPGVectorStore(ctx context.Context, vs *arcadiav1alpha1.VectorStore, c client.Client, dc dynamic.Interface, embedder embeddings.Embedder, collectionName string) (v *PGVectorStore, finish func(), err error) {
	v = &PGVectorStore{PGVector: vs.Spec.PGVector}
	ops := []pgvector.Option{
		pgvector.WithPreDeleteCollection(vs.Spec.PGVector.PreDeleteCollection),
	}
	if vs.Spec.PGVector.CollectionTableName != "" {
		ops = append(ops, pgvector.WithCollectionTableName(vs.Spec.PGVector.CollectionTableName))
	} else {
		v.PGVector.CollectionTableName = pgvector.DefaultCollectionStoreTableName
	}
	if vs.Spec.PGVector.EmbeddingTableName != "" {
		ops = append(ops, pgvector.WithEmbeddingTableName(vs.Spec.PGVector.EmbeddingTableName))
	} else {
		v.PGVector.EmbeddingTableName = pgvector.DefaultEmbeddingStoreTableName
	}
	if ref := vs.Spec.PGVector.DataSourceRef; ref != nil {
		if err := utils.ValidateClient(c, dc); err != nil {
			return nil, nil, err
		}
		ds := &arcadiav1alpha1.Datasource{}
		if c != nil {
			if err := c.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ref.GetNamespace(vs.GetNamespace())}, ds); err != nil {
				return nil, nil, err
			}
		} else {
			obj, err := dc.Resource(schema.GroupVersionResource{Group: "arcadia.kubeagi.k8s.com.cn", Version: "v1alpha1", Resource: "datasources"}).
				Namespace(ref.GetNamespace(vs.GetNamespace())).Get(ctx, ref.Name, metav1.GetOptions{})
			if err != nil {
				return nil, nil, err
			}
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), ds)
			if err != nil {
				return nil, nil, err
			}
		}
		vs.Spec.Endpoint = ds.Spec.Endpoint.DeepCopy()
		pool, err := datasource.GetPostgreSQLPool(ctx, c, dc, ds)
		if err != nil {
			return nil, nil, err
		}
		conn, err := pool.Acquire(ctx)
		if err != nil {
			return nil, nil, err
		}
		klog.V(5).Info("acquire pg conn from pool")
		finish = func() {
			if conn != nil {
				conn.Release()
				klog.V(5).Info("release pg conn to pool")
			}
		}
		v.Conn = conn.Conn()
		ops = append(ops, pgvector.WithConn(v.Conn))
	} else {
		conn, err := pgx.Connect(ctx, vs.Spec.Endpoint.URL)
		if err != nil {
			return nil, nil, err
		}
		v.Conn = conn
		ops = append(ops, pgvector.WithConn(conn))
	}
	if embedder != nil {
		ops = append(ops, pgvector.WithEmbedder(embedder))
	} else {
		llm, _ := openai.New()
		embedder, _ = embeddings.NewEmbedder(llm)
	}
	ops = append(ops, pgvector.WithEmbedder(embedder))
	if collectionName != "" {
		ops = append(ops, pgvector.WithCollectionName(collectionName))
		v.PGVector.CollectionName = collectionName
	} else {
		ops = append(ops, pgvector.WithCollectionName(vs.Spec.PGVector.CollectionName))
	}
	store, err := pgvector.New(ctx, ops...)
	if err != nil {
		return nil, nil, err
	}
	v.Store = store
	return v, finish, nil
}

// RemoveExist remove exist document from pgvector
// Note: it is currently assumed that the embedder of a knowledge base is constant that means the result of embedding a fixed document is fixed,
// disregarding the case where the embedder changes (and if it does, a lot of processing will need to be done in many places, not just here)
func (s *PGVectorStore) RemoveExist(ctx context.Context, log logr.Logger, document []lanchaingoschema.Document) (doc []lanchaingoschema.Document, err error) {
	// get collection_uuid from collection_table, if null, means no exits
	collectionUUID := ""
	sql := fmt.Sprintf(`SELECT uuid FROM %s WHERE name = $1 ORDER BY name limit 1`, s.PGVector.CollectionTableName)
	err = s.Conn.QueryRow(ctx, sql, s.PGVector.CollectionName).Scan(&collectionUUID)
	if collectionUUID == "" {
		return document, err
	}
	in := make([]string, 0)
	for _, d := range document {
		in = append(in, d.PageContent)
	}
	// Build a query every 100 entries to prevent the sql from being too large and causing errors
	step := 100
	start, end := 0, step
	res := make(map[string]lanchaingoschema.Document, 0)
	for i := 0; ; i++ {
		if start >= len(in) {
			break
		}
		if end > len(in) {
			end = len(in)
		}
		sql = fmt.Sprintf(`SELECT document, cmetadata FROM %s WHERE collection_id = $1 AND document = ANY($2)`, s.PGVector.EmbeddingTableName)
		rows, err := s.Conn.Query(ctx, sql, collectionUUID, in[start:end])
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			doc := lanchaingoschema.Document{}
			if err := rows.Scan(&doc.PageContent, &doc.Metadata); err != nil {
				return nil, err
			}
			res[doc.PageContent] = doc
		}
		if len(res) == 0 {
			return document, nil
		}
		start, end = end, end+step
	}
	if len(res) == len(document) {
		return nil, nil
	}
	for page := range res {
		log.V(5).Info(fmt.Sprintf("filter out exist documents[%s]", page))
	}
	doc = make([]lanchaingoschema.Document, 0, len(document))
	for _, d := range document {
		has, ok := res[d.PageContent]
		if ok {
			if reflect.DeepEqual(has.Metadata, d.Metadata) {
				continue
			}
			log.V(5).Info(fmt.Sprintf("exist document, same page content:%s, raw metadata:%v has metadata:%v", d.PageContent, d.Metadata, has.Metadata))
			for k, v := range d.Metadata {
				hasV := has.Metadata[k]
				if !reflect.DeepEqual(v, hasV) {
					log.V(5).Info(fmt.Sprintf("different metadata: raw:[%T]%v has:[%T]%v", v, v, hasV, hasV))
				}
			}
		}
		doc = append(doc, d)
	}
	return doc, nil
}
