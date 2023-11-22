package impl

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.40

import (
	"context"

	"github.com/kubeagi/arcadia/graphql-server/go-server/graph/generated"
	"github.com/kubeagi/arcadia/graphql-server/go-server/pkg/auth"
	"github.com/kubeagi/arcadia/graphql-server/go-server/pkg/client"
	defaultobject "github.com/kubeagi/arcadia/graphql-server/go-server/pkg/default_object"
	"github.com/kubeagi/arcadia/graphql-server/go-server/pkg/versioneddataset"
)

// VersionedDataset is the resolver for the VersionedDataset field.
func (r *mutationResolver) VersionedDataset(ctx context.Context) (*generated.VersionedDatasetMutation, error) {
	return &generated.VersionedDatasetMutation{}, nil
}

// VersionedDataset is the resolver for the VersionedDataset field.
func (r *queryResolver) VersionedDataset(ctx context.Context) (*generated.VersionedDatasetQuery, error) {
	return &generated.VersionedDatasetQuery{}, nil
}

// Files is the resolver for the files field.
func (r *versionedDatasetResolver) Files(ctx context.Context, obj *generated.VersionedDataset, input *generated.FileFilter) (*generated.PaginatedResult, error) {
	idtoken := auth.ForOIDCToken(ctx)
	c, err := client.GetClient(idtoken)
	if err != nil {
		return &defaultobject.DefaultPaginatedResult, err
	}
	if obj == nil {
		return &defaultobject.DefaultPaginatedResult, nil
	}
	return versioneddataset.VersionFiles(ctx, c, obj, input)
}

// CreateVersionedDataset is the resolver for the createVersionedDataset field.
func (r *versionedDatasetMutationResolver) CreateVersionedDataset(ctx context.Context, obj *generated.VersionedDatasetMutation, input generated.CreateVersionedDatasetInput) (*generated.VersionedDataset, error) {
	idtoken := auth.ForOIDCToken(ctx)
	c, err := client.GetClient(idtoken)
	if err != nil {
		return &defaultobject.DefaultVersioneddataset, err
	}
	return versioneddataset.CreateVersionedDataset(ctx, c, &input)
}

// UpdateVersionedDataset is the resolver for the updateVersionedDataset field.
func (r *versionedDatasetMutationResolver) UpdateVersionedDataset(ctx context.Context, obj *generated.VersionedDatasetMutation, input generated.UpdateVersionedDatasetInput) (*generated.VersionedDataset, error) {
	idtoken := auth.ForOIDCToken(ctx)
	c, err := client.GetClient(idtoken)
	if err != nil {
		return &generated.VersionedDataset{}, err
	}
	return versioneddataset.UpdateVersionedDataset(ctx, c, &input)
}

// DeleteVersionedDatasets is the resolver for the deleteVersionedDatasets field.
func (r *versionedDatasetMutationResolver) DeleteVersionedDatasets(ctx context.Context, obj *generated.VersionedDatasetMutation, input generated.DeleteVersionedDatasetInput) (*string, error) {
	idtoken := auth.ForOIDCToken(ctx)
	c, err := client.GetClient(idtoken)
	if err != nil {
		return &defaultobject.DefaultString, err
	}
	return versioneddataset.DeleteVersionedDatasets(ctx, c, &input)
}

// GetVersionedDataset is the resolver for the getVersionedDataset field.
func (r *versionedDatasetQueryResolver) GetVersionedDataset(ctx context.Context, obj *generated.VersionedDatasetQuery, name string, namespace string) (*generated.VersionedDataset, error) {
	idtoken := auth.ForOIDCToken(ctx)
	c, err := client.GetClient(idtoken)
	if err != nil {
		return &defaultobject.DefaultVersioneddataset, err
	}
	return versioneddataset.GetVersionedDataset(ctx, c, name, namespace)
}

// ListVersionedDatasets is the resolver for the listVersionedDatasets field.
func (r *versionedDatasetQueryResolver) ListVersionedDatasets(ctx context.Context, obj *generated.VersionedDatasetQuery, input generated.ListVersionedDatasetInput) (*generated.PaginatedResult, error) {
	idtoken := auth.ForOIDCToken(ctx)
	c, err := client.GetClient(idtoken)
	if err != nil {
		return &defaultobject.DefaultPaginatedResult, err
	}
	return versioneddataset.ListVersionedDatasets(ctx, c, &input)
}

// VersionedDataset returns generated.VersionedDatasetResolver implementation.
func (r *Resolver) VersionedDataset() generated.VersionedDatasetResolver {
	return &versionedDatasetResolver{r}
}

// VersionedDatasetMutation returns generated.VersionedDatasetMutationResolver implementation.
func (r *Resolver) VersionedDatasetMutation() generated.VersionedDatasetMutationResolver {
	return &versionedDatasetMutationResolver{r}
}

// VersionedDatasetQuery returns generated.VersionedDatasetQueryResolver implementation.
func (r *Resolver) VersionedDatasetQuery() generated.VersionedDatasetQueryResolver {
	return &versionedDatasetQueryResolver{r}
}

type versionedDatasetResolver struct{ *Resolver }
type versionedDatasetMutationResolver struct{ *Resolver }
type versionedDatasetQueryResolver struct{ *Resolver }
