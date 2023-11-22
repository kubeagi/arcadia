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

package minio

import (
	"context"
	"os"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"k8s.io/klog/v2"

	client3 "github.com/kubeagi/arcadia/graphql-server/go-server/pkg/client"
	"github.com/kubeagi/arcadia/pkg/config"
	"github.com/kubeagi/arcadia/pkg/utils"
)

var minioClient *minio.Client = nil
var coreClient *minio.Core = nil

var mutex *sync.Mutex

var (
	minioAddress         string
	minioAccessKeyID     string
	minioSecretAccessKey string
	minioSecure          bool
	minioBucket          string
	minioBasePath        string
)

func init() {
	mutex = new(sync.Mutex)

	if err := utils.SetSelfNamespace(); err != nil {
		klog.Errorf("unable to get self namespace: %s", err)
		os.Exit(1)
	}

	c, err := client3.GetClient(nil)
	if err != nil {
		panic(err)
	}
	minioConfig, err := config.GetMinIO(context.TODO(), c)
	if err != nil {
		panic(err)
	}
	minioAddress = minioConfig.MinioAddress
	minioAccessKeyID = minioConfig.MinioAccessKeyID
	minioSecretAccessKey = minioConfig.MinioSecretAccessKey
	minioBucket = minioConfig.MinioBucket
	minioBasePath = minioConfig.MinioBasePath
	minioSecure = minioConfig.MinioSecure
}

func GetClients() (*minio.Client, *minio.Core, error) {
	var client1 *minio.Client
	var client2 *minio.Core
	var err error

	mutex.Lock()

	if minioClient != nil && coreClient != nil {
		client1 = minioClient
		client2 = coreClient
		mutex.Unlock()
		return client1, client2, nil
	}

	aliasedURL := minioAddress
	accessKeyID := minioAccessKeyID
	secretAccessKey := minioSecretAccessKey
	secure := minioSecure

	if minioClient == nil {
		minioClient, err = minio.New(aliasedURL, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
			Secure: secure,
		})
	}
	if err != nil {
		mutex.Unlock()
		return nil, nil, err
	}

	client1 = minioClient

	if coreClient == nil {
		coreClient, err = minio.NewCore(aliasedURL, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
			Secure: secure,
		})
	}

	if err != nil {
		mutex.Unlock()
		return nil, nil, err
	}

	client2 = coreClient

	mutex.Unlock()

	return client1, client2, nil
}
