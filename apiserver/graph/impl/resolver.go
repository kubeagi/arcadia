package impl

import (
	"context"

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
	return client.GetClient(idtoken)
}
