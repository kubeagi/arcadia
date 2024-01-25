package impl

import (
	"context"
	"fmt"

	"k8s.io/client-go/dynamic"

	"github.com/kubeagi/arcadia/apiserver/pkg/auth"
	"github.com/kubeagi/arcadia/apiserver/pkg/client"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct{}

func getClientFromCtx(ctx context.Context) (dynamic.Interface, error) {
	idtoken := auth.ForOIDCToken(ctx)
	if idtoken == nil && auth.NeedAuth {
		return nil, fmt.Errorf("need auth but can't get token from request, abort")
	}
	return client.GetClient(idtoken)
}

func getAdminClient() (dynamic.Interface, error) {
	return client.GetClient(nil)
}
