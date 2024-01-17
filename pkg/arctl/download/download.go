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

package download

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type downloadOption struct {
	endpoint  string
	accessKey string
	secretKey string
	bucket    string
	src, dst  string

	secure bool

	minioClient *minio.Client
}

type DownloadOptionFunc func(*downloadOption)

type Download struct {
	option downloadOption
}

func WithMinioClient(c *minio.Client) DownloadOptionFunc {
	return func(do *downloadOption) {
		do.minioClient = c
	}
}
func WithBucket(bucket string) DownloadOptionFunc {
	return func(do *downloadOption) {
		do.bucket = bucket
	}
}

func WithSecure(secure bool) DownloadOptionFunc {
	return func(do *downloadOption) {
		do.secure = secure
	}
}

func WithEndpoint(api string) DownloadOptionFunc {
	return func(do *downloadOption) {
		do.endpoint = api
	}
}
func WithAccessKey(accessKey string) DownloadOptionFunc {
	return func(do *downloadOption) {
		do.accessKey = accessKey
	}
}

func WithSecretKey(secretKey string) DownloadOptionFunc {
	return func(do *downloadOption) {
		do.secretKey = secretKey
	}
}

func WithSrcFile(srcFile string) DownloadOptionFunc {
	return func(do *downloadOption) {
		do.src = srcFile
	}
}

func WithDstFile(dst string) DownloadOptionFunc {
	return func(do *downloadOption) {
		do.dst = dst
	}
}

func NewDownloader(options ...DownloadOptionFunc) *Download {
	d := &Download{option: downloadOption{}}
	for _, opt := range options {
		opt(&d.option)
	}
	return d
}

func (d *Download) minioClient(ctx context.Context) (*minio.Client, error) {
	return minio.New(d.option.endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(d.option.accessKey, d.option.secretKey, ""),
		Secure: d.option.secure,
	})
}

func (d *Download) parseDstFile(ctx context.Context) {
	_, fileName := filepath.Split(d.option.src)

	if d.option.dst == "" || d.option.dst == "." {
		d.option.dst = fmt.Sprintf("./%s", fileName)
	}
	if strings.HasSuffix(d.option.dst, "/") {
		d.option.dst += fileName
	}
}

func (d *Download) MinioClient() *minio.Client {
	return d.option.minioClient
}

func (d *Download) Download(ctx context.Context, options ...DownloadOptionFunc) error {
	for _, opt := range options {
		opt(&d.option)
	}
	var err error
	d.parseDstFile(ctx)
	if d.option.minioClient == nil {
		d.option.minioClient, err = d.minioClient(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to connect minio error %s\n", err)
			return err
		}
	}
	if dir, _ := filepath.Split(d.option.dst); dir != "" {
		if _, err = os.Stat(dir); err != nil {
			if !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "check that the directory %s is present fails error %s\n", dir, err)
				return err
			}
			if err = os.MkdirAll(dir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "failed to create dir %s error %s\n", dir, err)
				return err
			}
		}
	}
	fmt.Fprintf(os.Stdout, "copy source /%s/%s to %s\n", d.option.bucket, d.option.src, d.option.dst)

	object, err := d.option.minioClient.GetObject(ctx, d.option.bucket, d.option.src, minio.GetObjectOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get minio file object %s\n", err)
		return err
	}

	f, err := os.OpenFile(d.option.dst, os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open/create file %s error %s\n", d.option.dst, err)
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, object)
	return err
}
