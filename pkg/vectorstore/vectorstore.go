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

	"github.com/go-logr/logr"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/chroma"
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

var (
	ErrUnsupportedVectorStoreType = errors.New("unsupported vectorstore type")
)

func NewVectorStore(ctx context.Context, vs *arcadiav1alpha1.VectorStore, embedder embeddings.Embedder, collectionName string, c client.Client, dc dynamic.Interface) (v vectorstores.VectorStore, finish func(), err error) {
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
		ops := []pgvector.Option{
			pgvector.WithPreDeleteCollection(vs.Spec.PGVector.PreDeleteCollection),
		}
		if vs.Spec.PGVector.CollectionTableName != "" {
			ops = append(ops, pgvector.WithCollectionTableName(vs.Spec.PGVector.CollectionTableName))
		}
		if vs.Spec.PGVector.EmbeddingTableName != "" {
			ops = append(ops, pgvector.WithEmbeddingTableName(vs.Spec.PGVector.EmbeddingTableName))
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
			ops = append(ops, pgvector.WithConn(conn.Conn()))
		} else {
			ops = append(ops, pgvector.WithConnectionURL(vs.Spec.Endpoint.URL))
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
		} else {
			ops = append(ops, pgvector.WithCollectionName(vs.Spec.PGVector.CollectionName))
		}
		v, err = pgvector.New(ctx, ops...)
	case arcadiav1alpha1.VectorStoreTypeUnknown:
		fallthrough
	default:
		err = ErrUnsupportedVectorStoreType
	}
	return v, finish, err
}

func RemoveCollection(ctx context.Context, log logr.Logger, vs *arcadiav1alpha1.VectorStore, collectionName string) (err error) {
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
		ops := []pgvector.Option{
			pgvector.WithConnectionURL(vs.Spec.Endpoint.URL),
			pgvector.WithPreDeleteCollection(vs.Spec.PGVector.PreDeleteCollection),
			pgvector.WithCollectionTableName(vs.Spec.PGVector.CollectionTableName),
		}
		if collectionName != "" {
			ops = append(ops, pgvector.WithCollectionName(collectionName))
		} else {
			ops = append(ops, pgvector.WithCollectionName(vs.Spec.PGVector.CollectionName))
		}
		v, err := pgvector.New(ctx, ops...)
		if err != nil {
			log.Error(err, "reconcile delete: init vector store error, may leave garbage data")
			return err
		}
		if err = v.RemoveCollection(ctx); err != nil {
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
