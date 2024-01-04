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

package datasource

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/utils"
)

var (
	_      Datasource = (*PostgreSQL)(nil)
	locker sync.Mutex
)

// PostgreSQL is a wrapper to PostgreSQL
type PostgreSQL struct {
	*pgxpool.Pool
}

// NewPostgreSQL creates a new PostgreSQL pool
func NewPostgreSQL(ctx context.Context, c client.Client, dc dynamic.Interface, config *v1alpha1.PostgreSQL, endpoint *v1alpha1.Endpoint) (*PostgreSQL, error) {
	var pgUser, pgPassword, pgPassFile, pgSSLPassword string
	if endpoint.AuthSecret != nil {
		if endpoint.AuthSecret.Namespace == nil {
			return nil, errors.New("no namespace found for endpoint.authsecret")
		}
		if err := utils.ValidateClient(c, dc); err != nil {
			return nil, err
		}
		if dc != nil {
			secret, err := dc.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}).
				Namespace(*endpoint.AuthSecret.Namespace).Get(ctx, endpoint.AuthSecret.Name, v1.GetOptions{})
			if err != nil {
				return nil, err
			}
			data, _, _ := unstructured.NestedStringMap(secret.Object, "data")
			pgUser = utils.DecodeBase64Str(data[v1alpha1.PGUSER])
			pgPassword = utils.DecodeBase64Str(data[v1alpha1.PGPASSWORD])
			pgPassFile = utils.DecodeBase64Str(data[v1alpha1.PGPASSFILE])
			pgSSLPassword = utils.DecodeBase64Str(data[v1alpha1.PGSSLPASSWORD])
		}
		if c != nil {
			secret := corev1.Secret{}
			if err := c.Get(ctx, types.NamespacedName{
				Namespace: *endpoint.AuthSecret.Namespace,
				Name:      endpoint.AuthSecret.Name,
			}, &secret); err != nil {
				return nil, err
			}
			pgUser = string(secret.Data[v1alpha1.PGUSER])
			pgPassword = string(secret.Data[v1alpha1.PGPASSWORD])
			pgPassFile = string(secret.Data[v1alpha1.PGPASSFILE])
			pgSSLPassword = string(secret.Data[v1alpha1.PGSSLPASSWORD])
		}
	}
	locker.Lock()
	defer locker.Unlock()
	if pgUser != "" {
		if err := os.Setenv("PGUSER", pgUser); err != nil {
			return nil, err
		}
		defer os.Unsetenv("PGUSER")
	}
	if pgPassword != "" {
		if err := os.Setenv("PGPASSWORD", pgPassword); err != nil {
			return nil, err
		}
		defer os.Unsetenv("PGPASSWORD")
	}
	if pgPassFile != "" {
		if err := os.Setenv("PGPASSFILE", pgPassFile); err != nil {
			return nil, err
		}
		defer os.Unsetenv("PGPASSFILE")
	}
	if pgSSLPassword != "" {
		if err := os.Setenv("PGSSLPASSWORD", pgSSLPassword); err != nil {
			return nil, err
		}
		defer os.Unsetenv("PGSSLPASSWORD")
	}
	if config != nil {
		if config.Host != "" {
			if err := os.Setenv("PGHOST", config.Host); err != nil {
				return nil, err
			}
			defer os.Unsetenv("PGHOST")
		}
		if config.Port != "" {
			if err := os.Setenv("PGPORT", config.Port); err != nil {
				return nil, err
			}
			defer os.Unsetenv("PGPORT")
		}
		if config.Database != "" {
			if err := os.Setenv("PGDATABASE", config.Database); err != nil {
				return nil, err
			}
			defer os.Unsetenv("PGDATABASE")
		}
		if config.AppName != "" {
			if err := os.Setenv("PGAPPNAME", config.AppName); err != nil {
				return nil, err
			}
			defer os.Unsetenv("PGAPPNAME")
		}
		if config.ConnectTimeout != "" {
			if err := os.Setenv("PGCONNECT_TIMEOUT", config.ConnectTimeout); err != nil {
				return nil, err
			}
			defer os.Unsetenv("PGCONNECT_TIMEOUT")
		}
		if config.SSLMode != "" {
			if err := os.Setenv("PGSSLMODE", config.SSLMode); err != nil {
				return nil, err
			}
			defer os.Unsetenv("PGSSLMODE")
		}
		if config.SSLKey != "" {
			if err := os.Setenv("PGSSLKEY", config.SSLKey); err != nil {
				return nil, err
			}
			defer os.Unsetenv("PGSSLKEY")
		}
		if config.SSLCert != "" {
			if err := os.Setenv("PGSSLCERT", config.SSLCert); err != nil {
				return nil, err
			}
			defer os.Unsetenv("PGSSLCERT")
		}
		if config.SSLSni != "" {
			if err := os.Setenv("PGSSLSNI", config.SSLSni); err != nil {
				return nil, err
			}
			defer os.Unsetenv("PGSSLSNI")
		}
		if config.SSLRootCert != "" {
			if err := os.Setenv("PGSSLROOTCERT", config.SSLRootCert); err != nil {
				return nil, err
			}
			defer os.Unsetenv("PGSSLROOTCERT")
		}
		if config.TargetSessionAttrs != "" {
			if err := os.Setenv("PGTARGETSESSIONATTRS", config.TargetSessionAttrs); err != nil {
				return nil, err
			}
			defer os.Unsetenv("PGTARGETSESSIONATTRS")
		}
		if config.Service != "" {
			if err := os.Setenv("PGSERVICE", config.Service); err != nil {
				return nil, err
			}
			defer os.Unsetenv("PGSERVICE")
		}
		if config.ServiceFile != "" {
			if err := os.Setenv("PGSERVICEFILE", config.ServiceFile); err != nil {
				return nil, err
			}
			defer os.Unsetenv("PGSERVICEFILE")
		}
	}
	pool, err := pgxpool.New(context.Background(), endpoint.URL)
	if err != nil {
		return nil, err
	}
	return &PostgreSQL{pool}, nil
}

func (p *PostgreSQL) Stat(ctx context.Context, _ any) error {
	return p.Ping(ctx)
}

func (p *PostgreSQL) Remove(ctx context.Context, info any) error {
	// TODO implement me
	panic("implement me")
}

func (p *PostgreSQL) ReadFile(ctx context.Context, info any) (io.ReadCloser, error) {
	// TODO implement me
	panic("implement me")
}

func (p *PostgreSQL) StatFile(ctx context.Context, info any) (any, error) {
	// TODO implement me
	panic("implement me")
}

func (p *PostgreSQL) GetTags(ctx context.Context, info any) (map[string]string, error) {
	// TODO implement me
	panic("implement me")
}

func (p *PostgreSQL) ListObjects(ctx context.Context, source string, info any) (any, error) {
	// TODO implement me
	panic("implement me")
}
