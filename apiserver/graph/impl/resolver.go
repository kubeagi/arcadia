package impl

import (
	"context"
	"fmt"

	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/apiserver/pkg/auth"
	"github.com/kubeagi/arcadia/apiserver/pkg/client"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct{}

func getClientFromCtx(ctx context.Context) (runtimeclient.Client, error) {
	idtoken := auth.ForOIDCToken(ctx)
	if idtoken == nil && auth.NeedAuth {
		return nil, fmt.Errorf("need auth but can't get token from request, abort")
	}
	return client.GetClient(idtoken)
}

func getAdminClient() (runtimeclient.Client, error) {
	return client.GetClient(nil)
}
