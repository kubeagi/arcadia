package impl

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.
// Code generated by github.com/99designs/gqlgen version v0.17.40

import (
	"context"

	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/dataprocessing"
)

// CreateDataProcessTask is the resolver for the createDataProcessTask field.
func (r *dataProcessMutationResolver) CreateDataProcessTask(ctx context.Context, obj *generated.DataProcessMutation, input *generated.AddDataProcessInput) (*generated.DataProcessResponse, error) {
	c, err := getClientFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dataprocessing.CreateDataProcessTask(ctx, c, obj, input)
}

// DeleteDataProcessTask is the resolver for the deleteDataProcessTask field.
func (r *dataProcessMutationResolver) DeleteDataProcessTask(ctx context.Context, obj *generated.DataProcessMutation, input *generated.DeleteDataProcessInput) (*generated.DataProcessResponse, error) {
	c, err := getClientFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dataprocessing.DeleteDataProcessTask(ctx, c, obj, input)
}

// AllDataProcessListByPage is the resolver for the allDataProcessListByPage field.
func (r *dataProcessQueryResolver) AllDataProcessListByPage(ctx context.Context, obj *generated.DataProcessQuery, input *generated.AllDataProcessListByPageInput) (*generated.PaginatedDataProcessItem, error) {
	c, err := getClientFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dataprocessing.ListDataprocessing(ctx, c, obj, input)
}

// AllDataProcessListByCount is the resolver for the allDataProcessListByCount field.
func (r *dataProcessQueryResolver) AllDataProcessListByCount(ctx context.Context, obj *generated.DataProcessQuery, input *generated.AllDataProcessListByCountInput) (*generated.CountDataProcessItem, error) {
	c, err := getClientFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dataprocessing.ListDataprocessingByCount(ctx, c, obj, input)
}

// DataProcessSupportType is the resolver for the dataProcessSupportType field.
func (r *dataProcessQueryResolver) DataProcessSupportType(ctx context.Context, obj *generated.DataProcessQuery) (*generated.DataProcessSupportType, error) {
	c, err := getClientFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dataprocessing.DataProcessSupportType(ctx, c, obj)
}

// DataProcessDetails is the resolver for the dataProcessDetails field.
func (r *dataProcessQueryResolver) DataProcessDetails(ctx context.Context, obj *generated.DataProcessQuery, input *generated.DataProcessDetailsInput) (*generated.DataProcessDetails, error) {
	c, err := getClientFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dataprocessing.DataProcessDetails(ctx, c, obj, input)
}

// CheckDataProcessTaskName is the resolver for the checkDataProcessTaskName field.
func (r *dataProcessQueryResolver) CheckDataProcessTaskName(ctx context.Context, obj *generated.DataProcessQuery, input *generated.CheckDataProcessTaskNameInput) (*generated.DataProcessResponse, error) {
	c, err := getClientFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dataprocessing.CheckDataProcessTaskName(ctx, c, obj, input)
}

// GetLogInfo is the resolver for the getLogInfo field.
func (r *dataProcessQueryResolver) GetLogInfo(ctx context.Context, obj *generated.DataProcessQuery, input *generated.DataProcessDetailsInput) (*generated.DataProcessResponse, error) {
	c, err := getClientFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	return dataprocessing.GetLogInfo(ctx, c, obj, input)
}

// DataProcess is the resolver for the dataProcess field.
func (r *mutationResolver) DataProcess(ctx context.Context) (*generated.DataProcessMutation, error) {
	return &generated.DataProcessMutation{}, nil
}

// DataProcess is the resolver for the dataProcess field.
func (r *queryResolver) DataProcess(ctx context.Context) (*generated.DataProcessQuery, error) {
	return &generated.DataProcessQuery{}, nil
}

// DataProcessMutation returns generated.DataProcessMutationResolver implementation.
func (r *Resolver) DataProcessMutation() generated.DataProcessMutationResolver {
	return &dataProcessMutationResolver{r}
}

// DataProcessQuery returns generated.DataProcessQueryResolver implementation.
func (r *Resolver) DataProcessQuery() generated.DataProcessQueryResolver {
	return &dataProcessQueryResolver{r}
}

type dataProcessMutationResolver struct{ *Resolver }
type dataProcessQueryResolver struct{ *Resolver }
