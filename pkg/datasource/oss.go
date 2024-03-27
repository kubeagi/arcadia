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
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
)

var (
	_ Datasource    = (*OSS)(nil)
	_ ChunkUploader = (*OSS)(nil)
)

const ObjectNotExistMsg = "The specified key does not exist."

// OSS is a wrapper to object storage service
type OSS struct {
	*minio.Client
	*minio.Core
}

func NewOSS(ctx context.Context, c client.Client, endpoint *v1alpha1.Endpoint) (*OSS, error) {
	var accessKeyID, secretAccessKey string
	if endpoint.AuthSecret != nil {
		if endpoint.AuthSecret.Namespace == nil {
			return nil, errors.New("no namespace found for endpoint.authsecret")
		}
		data, err := endpoint.AuthData(ctx, *endpoint.AuthSecret.Namespace, c)
		if err != nil {
			return nil, err
		}
		accessKeyID = string(data["rootUser"])
		secretAccessKey = string(data["rootPassword"])
	}

	mc, err := minio.New(endpoint.URL, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: !endpoint.Insecure,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	core, err := minio.NewCore(endpoint.URL, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: !endpoint.Insecure,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return &OSS{Client: mc, Core: core}, nil
}

// Check oss against info()
func (oss *OSS) Stat(ctx context.Context, info any) error {
	if info == nil {
		return nil
	}
	ossInfo, ok := info.(*v1alpha1.OSS)
	if !ok {
		return ErrOSSNoConfig
	}

	return oss.statObject(ctx, ossInfo)
}

func (oss *OSS) Remove(ctx context.Context, info any) error {
	ossInfo, err := oss.preCheck(info)
	if err != nil {
		return err
	}

	// NOTE: all versions of a file need to be deleted,
	// so when deleting a file, the prefix is set to the full path to ensure that all versions are deleted.
	for e := range oss.Client.RemoveObjects(
		ctx,
		ossInfo.Bucket,
		oss.Client.ListObjects(ctx, ossInfo.Bucket, minio.ListObjectsOptions{Prefix: ossInfo.Object, Recursive: true}),
		minio.RemoveObjectsOptions{}) {
		if e.Err != nil {
			return e.Err
		}
	}

	return nil
}

// StatObject against oss info
// Q: Why not using client.StatObject() ?
// A: The `StateObject()` won't treat path(directory) as a valid object
func (oss *OSS) statObject(ctx context.Context, ossInfo *v1alpha1.OSS) error {
	if ossInfo.Bucket == "" {
		return ErrBucketNotProvided
	}

	// check whether bucket exists
	isExist, err := oss.Client.BucketExists(ctx, ossInfo.Bucket)
	if err != nil {
		return err
	}
	if !isExist {
		return ErrOSSNoSuchBucket
	}

	// check whether object exists
	if ossInfo.Object != "" {
		// The object by `ListObjects` will trim "/" automatically,so we also need to trim "/" to make sure name comparison successful
		ossInfo.Object = strings.TrimPrefix(ossInfo.Object, "/")
		// When object contains "/" which means it is a directory,'ListObjects' will show all objects under that directory without object itself
		// After we remove "/", the objects by `ListObjects` will have object itself included.
		trimmedObjectPath := strings.TrimSuffix(ossInfo.Object, "/")
		for objInfo := range oss.Client.ListObjects(
			ctx, ossInfo.Bucket, minio.ListObjectsOptions{
				Prefix: trimmedObjectPath,
			},
		) {
			if objInfo.Key == ossInfo.Object {
				return nil
			}
		}
		return ErrOSSNoSuchObject
	}

	return nil
}

func (oss *OSS) ReadFile(ctx context.Context, info any) (io.ReadCloser, error) {
	ossInfo, err := oss.preCheck(info)
	if err != nil {
		return nil, err
	}
	return oss.Client.GetObject(ctx, ossInfo.Bucket, ossInfo.Object, minio.GetObjectOptions{VersionID: ossInfo.VersionID})
}

func (oss *OSS) StatFile(ctx context.Context, info any) (any, error) {
	ossInfo, err := oss.preCheck(info)
	if err != nil {
		return nil, err
	}
	return oss.Client.StatObject(ctx, ossInfo.Bucket, ossInfo.Object, minio.GetObjectOptions{VersionID: ossInfo.VersionID})
}

func (oss *OSS) GetTags(ctx context.Context, info any) (map[string]string, error) {
	ossInfo, err := oss.preCheck(info)
	if err != nil {
		return nil, err
	}
	tags, err := oss.Client.GetObjectTagging(ctx, ossInfo.Bucket, ossInfo.Object, minio.GetObjectTaggingOptions{VersionID: ossInfo.VersionID})
	if err != nil {
		return nil, err
	}
	return tags.ToMap(), nil
}

func (oss *OSS) ListObjects(ctx context.Context, bucketName string, info any) (any, error) {
	result := make([]minio.ObjectInfo, 0)
	listOption, ok := info.(minio.ListObjectsOptions)
	if !ok {
		return result, fmt.Errorf("info should be of type ListObjectOptions")
	}
	for object := range oss.Client.ListObjects(ctx, bucketName, listOption) {
		result = append(result, object)
	}
	return result, nil
}

func (oss *OSS) CompletedChunks(ctx context.Context, options ...ChunkUploaderOption) (any, error) {
	s := ChunkUploaderConf{}
	for _, opt := range options {
		opt(&s)
	}
	objectName := fmt.Sprintf("%s/%s", s.relativeDir, s.fileName)
	return oss.Core.ListObjectParts(ctx, s.bucket, objectName, s.uploadID, 0, 10000)
}

func (oss *OSS) NewMultipartIdentifier(ctx context.Context, options ...ChunkUploaderOption) (string, error) {
	s := ChunkUploaderConf{}
	for _, opt := range options {
		opt(&s)
	}
	objectName := fmt.Sprintf("%s/%s", s.relativeDir, s.fileName)

	return oss.Core.NewMultipartUpload(ctx, s.bucket, objectName, minio.PutObjectOptions{
		UserMetadata: s.annotations,
	})
}

func (oss *OSS) GenMultipartSignedURL(ctx context.Context, options ...ChunkUploaderOption) (string, error) {
	s := ChunkUploaderConf{}
	for _, opt := range options {
		opt(&s)
	}

	objectName := fmt.Sprintf("%s/%s", s.relativeDir, s.fileName)
	u, err := oss.Core.Presign(ctx, http.MethodPut, s.bucket, objectName, 24*time.Hour*7, url.Values{
		"uploadId":   []string{s.uploadID},
		"partNumber": []string{s.partNumber},
	})
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (oss *OSS) Complete(ctx context.Context, options ...ChunkUploaderOption) error {
	s := ChunkUploaderConf{}
	for _, opt := range options {
		opt(&s)
	}

	objectName := fmt.Sprintf("%s/%s", s.relativeDir, s.fileName)
	parts, err := oss.Core.ListObjectParts(ctx, s.bucket, objectName, s.uploadID, 0, 10000)
	if err != nil {
		return err
	}
	completeParts := make([]minio.CompletePart, len(parts.ObjectParts))
	for idx, obj := range parts.ObjectParts {
		completeParts[idx] = minio.CompletePart{
			PartNumber: obj.PartNumber,
			ETag:       obj.ETag,
		}
	}
	sort.Slice(completeParts, func(i int, j int) bool {
		return completeParts[i].PartNumber < completeParts[j].PartNumber
	})
	_, err = oss.Core.CompleteMultipartUpload(ctx, s.bucket, objectName, s.uploadID, completeParts, minio.PutObjectOptions{})
	return err
}

func (oss *OSS) Abort(ctx context.Context, options ...ChunkUploaderOption) error {
	s := ChunkUploaderConf{}
	for _, opt := range options {
		opt(&s)
	}

	objectName := fmt.Sprintf("%s/%s", s.relativeDir, s.fileName)
	return oss.Core.AbortMultipartUpload(ctx, s.bucket, objectName, s.uploadID)
}

func (oss *OSS) preCheck(info any) (*v1alpha1.OSS, error) {
	if info == nil {
		return nil, ErrOSSNoConfig
	}
	ossInfo, ok := info.(*v1alpha1.OSS)
	if !ok || ossInfo.Bucket == "" || ossInfo.Object == "" {
		return nil, ErrOSSNoConfig
	}
	return ossInfo, nil
}

func (oss *OSS) IncompleteUpload(ctx context.Context, options ...ChunkUploaderOption) (string, error) {
	s := ChunkUploaderConf{}
	for _, opt := range options {
		opt(&s)
	}

	objectName := fmt.Sprintf("%s/%s", s.relativeDir, s.fileName)
	var (
		uploadID string
		cur      time.Time
	)
	first := true
	for id := range oss.Client.ListIncompleteUploads(ctx, s.bucket, objectName, true) {
		if first || id.Initiated.After(cur) {
			uploadID = id.UploadID
			cur = id.Initiated
			first = false
		}
	}
	return uploadID, nil
}
