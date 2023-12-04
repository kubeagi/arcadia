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

package datasource

import (
	"context"
	"errors"
	"io"

	"github.com/minio/minio-go/v7"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
)

var (
	ErrUnknowDatasourceType = errors.New("unknow datasource type")
	ErrBucketNotProvided    = errors.New("no bucket provided")
	ErrOSSNoSuchBucket      = errors.New("no such bucket")
	ErrOSSNoSuchObject      = errors.New("no such object in bucket")
	ErrOSSNoConfig          = errors.New("no bucket or object config")
)

type ChunkUploaderConf struct {
	bucket, relativeDir, fileName, md5 string

	// This uploadid is a unique identifier generated for the chunked upload of files
	uploadID string

	partNumber string

	annotations map[string]string
}

type ChunkUploaderOption func(*ChunkUploaderConf)

func WithBucket(bucket string) ChunkUploaderOption {
	return func(cuo *ChunkUploaderConf) {
		cuo.bucket = bucket
	}
}

func WithBucketPath(relativeDir string) ChunkUploaderOption {
	return func(cuo *ChunkUploaderConf) {
		cuo.relativeDir = relativeDir
	}
}

func WithFileName(fileName string) ChunkUploaderOption {
	return func(cuo *ChunkUploaderConf) {
		cuo.fileName = fileName
	}
}

func WithMD5(md5 string) ChunkUploaderOption {
	return func(cuo *ChunkUploaderConf) {
		cuo.md5 = md5
	}
}

func WithAnnotations(annotations map[string]string) ChunkUploaderOption {
	return func(cuo *ChunkUploaderConf) {
		cuo.annotations = annotations
	}
}

func WithUploadID(uploadID string) ChunkUploaderOption {
	return func(cuo *ChunkUploaderConf) {
		cuo.uploadID = uploadID
	}
}

func WithPartNumber(partNumber string) ChunkUploaderOption {
	return func(cuo *ChunkUploaderConf) {
		cuo.partNumber = partNumber
	}
}

/*
Uploading files in chunks.
1. Get completed blocks
2. if there is no unique uploadid, you need to request a unique uploadid
3. request the upload file URL
4. Upload the file via the URL
5. Update the information of the uploaded block
6. merge files
*/
type ChunkUploader interface {
	// Queries for blocks that have already been uploaded.
	CompletedChunks(context.Context, ...ChunkUploaderOption) (any, error)

	// CompletedChunks(context.Context, ...chunkUploaderOption) (any, error)
	// Generate uploadID for uploaded files
	NewMultipartIdentifier(context.Context, ...ChunkUploaderOption) (string, error)

	// Generate URLs for uploading files
	GenMultipartSignedURL(context.Context, ...ChunkUploaderOption) (string, error)

	// After all the chunks are uploaded, this interface is called and the backend merges the files.
	Complete(context.Context, ...ChunkUploaderOption) error

	// To stop the upload, the user needs to destroy the chunked data.
	Abort(context.Context, ...ChunkUploaderOption) error
}

type Datasource interface {
	Stat(ctx context.Context, info any) error
	Remove(ctx context.Context, info any) error
	ReadFile(ctx context.Context, info any) (io.ReadCloser, error)
	StatFile(ctx context.Context, info any) (any, error)
	GetTags(ctx context.Context, info any) (map[string]string, error)
	ListObjects(ctx context.Context, source string, info any) (any, error)
}

var _ Datasource = (*Unknown)(nil)

type Unknown struct {
}

func NewUnknown(ctx context.Context, c client.Client) (*Unknown, error) {
	return &Unknown{}, nil
}

func (u *Unknown) Stat(ctx context.Context, info any) error {
	return ErrUnknowDatasourceType
}

func (u *Unknown) Remove(ctx context.Context, info any) error {
	return ErrUnknowDatasourceType
}

func (u *Unknown) ReadFile(ctx context.Context, info any) (io.ReadCloser, error) {
	return nil, ErrUnknowDatasourceType
}

func (u *Unknown) StatFile(ctx context.Context, info any) (any, error) {
	return nil, ErrUnknowDatasourceType
}

func (u *Unknown) GetTags(ctx context.Context, info any) (map[string]string, error) {
	return nil, ErrUnknowDatasourceType
}

func (*Unknown) ListObjects(ctx context.Context, source string, info any) (any, error) {
	return nil, ErrUnknowDatasourceType
}

var _ Datasource = (*Local)(nil)

// Local is a special datasource which use the system datasource as oss to store user-uploaded local files
// - `oss` in `Local` represents the system datasource oss client along with the `Local`'s oss info
type Local struct {
	oss *OSS
}

func NewLocal(ctx context.Context, c client.Client, endpoint *v1alpha1.Endpoint) (*Local, error) {
	oss, err := NewOSS(ctx, c, endpoint)
	if err != nil {
		return nil, err
	}
	return &Local{oss: oss}, nil
}

// Stat `Local` with `OSS`
func (local *Local) Stat(ctx context.Context, options any) (err error) {
	err = local.oss.Stat(ctx, options)
	if err != nil && errors.Is(err, ErrOSSNoSuchBucket) {
		ossInfo, ok := options.(*v1alpha1.OSS)
		if !ok {
			return errors.New("invalid stat info for OSS")
		}
		defautlMakeBucketOptions := minio.MakeBucketOptions{}
		err = local.oss.MakeBucket(ctx, ossInfo.Bucket, defautlMakeBucketOptions)
	}
	return err
}

// Remove object from OSS
func (local *Local) Remove(ctx context.Context, info any) error {
	return local.oss.Remove(ctx, info)
}

func (local *Local) ReadFile(ctx context.Context, info any) (io.ReadCloser, error) {
	return local.oss.ReadFile(ctx, info)
}

func (local *Local) StatFile(ctx context.Context, info any) (any, error) {
	return local.oss.StatFile(ctx, info)
}

func (local *Local) GetTags(ctx context.Context, info any) (map[string]string, error) {
	return local.oss.GetTags(ctx, info)
}
func (local *Local) ListObjects(ctx context.Context, source string, info any) (any, error) {
	return local.oss.ListObjects(ctx, source, info)
}
