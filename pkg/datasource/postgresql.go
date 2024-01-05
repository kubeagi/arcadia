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
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
)

var (
	_          Datasource = (*PostgreSQL)(nil)
	locker     sync.Mutex
	poolsMutex sync.Mutex
	pools      = make(map[string]*PostgreSQL)
)

func GetPostgreSQLPool(ctx context.Context, c client.Client, dc dynamic.Interface, datasource *v1alpha1.Datasource) (*PostgreSQL, error) {
	if datasource.Spec.Type() != v1alpha1.DatasourceTypePostgreSQL {
		return nil, ErrUnknowDatasourceType
	}
	pg, ok := pools[string(datasource.GetUID())]
	if ok && pg.Ref.GetGeneration() == datasource.GetGeneration() {
		return pg, nil
	}
	pg, err := newPostgreSQL(ctx, c, dc, datasource.Spec.PostgreSQL, &datasource.Spec.Endpoint)
	if err != nil {
		return nil, err
	}
	pg.Ref = datasource.DeepCopy()
	poolsMutex.Lock()
	pools[string(datasource.GetUID())] = pg
	poolsMutex.Unlock()
	return pg, nil
}

func RemovePostgreSQLPool(datasource v1alpha1.Datasource) {
	pg, ok := pools[string(datasource.GetUID())]
	if !ok {
		return
	}
	pg.Pool.Close()
	poolsMutex.Lock()
	delete(pools, string(datasource.GetUID()))
	poolsMutex.Unlock()
}

// PostgreSQL is a wrapper to PostgreSQL
type PostgreSQL struct {
	*pgxpool.Pool
	Ref *v1alpha1.Datasource
}

// NewPostgreSQL creates a new PostgreSQL pool
func newPostgreSQL(ctx context.Context, c client.Client, dc dynamic.Interface, config *v1alpha1.PostgreSQL, endpoint *v1alpha1.Endpoint) (*PostgreSQL, error) {
	var pgUser, pgPassword, pgPassFile, pgSSLPassword string
	if endpoint.AuthSecret != nil {
		if endpoint.AuthSecret.Namespace == nil {
			return nil, errors.New("no namespace found for endpoint.authsecret")
		}
		data, err := endpoint.AuthData(ctx, *endpoint.AuthSecret.Namespace, c, dc)
		if err != nil {
			return nil, err
		}
		pgUser = string(data[v1alpha1.PGUSER])
		pgPassword = string(data[v1alpha1.PGPASSWORD])
		pgPassFile = string(data[v1alpha1.PGPASSFILE])
		pgSSLPassword = string(data[v1alpha1.PGSSLPASSWORD])
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
	return &PostgreSQL{Pool: pool}, nil
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
