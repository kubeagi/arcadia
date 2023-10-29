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

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/v1alpha1"
)

var (
	ErrUnknowDatasourceType = errors.New("unknow datasource type")
)

type Datasource interface {
	Check(ctx context.Context, options any) error
}

type Unknown struct {
}

func NewUnknown(ctx context.Context, c client.Client) (*Unknown, error) {
	return &Unknown{}, nil
}

func (u *Unknown) Check(ctx context.Context, options any) error {
	return ErrUnknowDatasourceType
}

var _ Datasource = (*OSS)(nil)

type OSS struct {
	*minio.Client
}

func NewOSS(ctx context.Context, c client.Client, endpoint *v1alpha1.Endpoint, ossInfo *v1alpha1.OSS) (*OSS, error) {
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
		secretAccessKey = string(secret.Data["rootUser"])

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

// TODO: implement Check with specific `options`
func (oss *OSS) Check(ctx context.Context, options any) error {
	return nil
}
