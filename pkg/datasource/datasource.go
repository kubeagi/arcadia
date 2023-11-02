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
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/v1alpha1"
)

var (
	ErrUnknowDatasourceType = errors.New("unknow datasource type")
	ErrBucketNotProvided    = errors.New("no bucket provided")
	ErrOSSNoSuchBucket      = errors.New("no such bucket")
	ErrOSSNoSuchObject      = errors.New("no such object in bucket")
	ErrOSSNoConfig          = errors.New("no bucket or object config")
)

type Datasource interface {
	Stat(ctx context.Context, info any) error
	Remove(ctx context.Context, info any) error
	ReadFile(ctx context.Context, info any) (io.ReadCloser, error)
	StatFile(ctx context.Context, info any) (any, error)
	GetTags(ctx context.Context, info any) (map[string]string, error)
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

var _ Datasource = (*OSS)(nil)

// OSS is a wrapper to object storage service
type OSS struct {
	*minio.Client
}

var (
	ossDefaultGetOpt    = minio.GetObjectOptions{}
	ossDefaultGetTagOpt = minio.GetObjectTaggingOptions{}
)

func NewOSS(ctx context.Context, c client.Client, endpoint *v1alpha1.Endpoint) (*OSS, error) {
	var accessKeyID, secretAccessKey string
	if endpoint.AuthSecret != nil {
		secret := corev1.Secret{}
		if err := c.Get(ctx, types.NamespacedName{
			Namespace: *endpoint.AuthSecret.Namespace,
			Name:      endpoint.AuthSecret.Name,
		}, &secret); err != nil {
			return nil, err
		}
		accessKeyID = string(secret.Data["rootUser"])
		secretAccessKey = string(secret.Data["rootPassword"])

		// TODO: implement https(secure check)
		// if !endpoint.Insecure {
		// }
	}

	mc, err := minio.New(endpoint.URL, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: !endpoint.Insecure,
	})
	if err != nil {
		return nil, err
	}

	return &OSS{Client: mc}, nil
}

// Check oss agains info()
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

// TODO: implement `Remove` against info
func (oss *OSS) Remove(ctx context.Context, info any) error {
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
		// The object by `ListObjects` will trim "/" automatically,so we also need to trim "/" to make sure name comparision successful
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
	return oss.Client.GetObject(ctx, ossInfo.Bucket, ossInfo.Object, ossDefaultGetOpt)
}

func (oss *OSS) StatFile(ctx context.Context, info any) (any, error) {
	ossInfo, err := oss.preCheck(info)
	if err != nil {
		return nil, err
	}
	return oss.Client.StatObject(ctx, ossInfo.Bucket, ossInfo.Object, ossDefaultGetOpt)
}

func (oss *OSS) GetTags(ctx context.Context, info any) (map[string]string, error) {
	ossInfo, err := oss.preCheck(info)
	if err != nil {
		return nil, err
	}
	tags, err := oss.Client.GetObjectTagging(ctx, ossInfo.Bucket, ossInfo.Object, ossDefaultGetTagOpt)
	if err != nil {
		return nil, err
	}
	return tags.ToMap(), nil
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
