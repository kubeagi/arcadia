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

package model

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	miniogo "github.com/minio/minio-go/v7"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/config"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	pkgclient "github.com/kubeagi/arcadia/apiserver/pkg/client"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	graphqlutils "github.com/kubeagi/arcadia/apiserver/pkg/utils"
	"github.com/kubeagi/arcadia/pkg/utils"
)

func obj2modelConverter(obj client.Object) (generated.PageNode, error) {
	model, ok := obj.(*v1alpha1.Model)
	if !ok {
		return nil, errors.New("can't convert object to Model")
	}
	return obj2model(model)
}

func obj2model(model *v1alpha1.Model) (*generated.Model, error) {
	id := string(model.GetUID())
	creationtimestamp := model.GetCreationTimestamp().Time

	// conditioned status
	condition := model.Status.GetCondition(v1alpha1.TypeReady)
	updateTime := condition.LastTransitionTime.Time
	status := common.GetObjStatus(model)
	message := string(condition.Message)

	var systemModel bool
	if model.GetNamespace() == config.GetConfig().SystemNamespace {
		systemModel = true
	}

	md := generated.Model{
		ID:                &id,
		Name:              model.GetName(),
		Namespace:         model.GetNamespace(),
		Creator:           &model.Spec.Creator,
		SystemModel:       &systemModel,
		Labels:            graphqlutils.MapStr2Any(model.GetLabels()),
		Annotations:       graphqlutils.MapStr2Any(model.GetAnnotations()),
		DisplayName:       &model.Spec.DisplayName,
		Description:       &model.Spec.Description,
		Types:             model.Spec.Types,
		CreationTimestamp: &creationtimestamp,
		UpdateTimestamp:   &updateTime,
		Status:            &status,
		Message:           &message,
		HuggingFaceRepo:   &model.Spec.HuggingFaceRepo,
		ModelScopeRepo:    &model.Spec.ModelScopeRepo,
		Revision:          &model.Spec.Revision,
		ModelSource:       &model.Spec.ModelSource,
	}
	return &md, nil
}

func CreateModel(ctx context.Context, c client.Client, input generated.CreateModelInput) (*generated.Model, error) {
	model := &v1alpha1.Model{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
		},
		Spec: v1alpha1.ModelSpec{
			Types: input.Types,
		},
	}
	if *input.ModelSource == common.ModelSourceModelscope {
		if *input.Revision == "" {
			return nil, errors.New("argument revision is required")
		}
		model.Spec.ModelScopeRepo = *input.ModelScopeRepo
		model.Spec.Revision = *input.Revision
	}
	if *input.ModelSource == common.ModelSourceHuggingface {
		model.Spec.HuggingFaceRepo = *input.HuggingFaceRepo
		model.Spec.Revision = *input.Revision
	}
	model.Spec.ModelSource = *input.ModelSource
	model.Spec.DisplayName = pointer.StringDeref(input.DisplayName, model.Spec.DisplayName)
	model.Spec.Description = pointer.StringDeref(input.Description, model.Spec.Description)
	common.SetCreator(ctx, &model.Spec.CommonSpec)
	err := c.Create(ctx, model)
	if err != nil {
		return nil, err
	}
	return obj2model(model)
}

func UpdateModel(ctx context.Context, c client.Client, input *generated.UpdateModelInput) (*generated.Model, error) {
	model := &v1alpha1.Model{}
	err := c.Get(ctx, types.NamespacedName{Namespace: input.Namespace, Name: input.Name}, model)
	if err != nil {
		return nil, err
	}

	model.SetLabels(graphqlutils.MapAny2Str(input.Labels))
	model.SetAnnotations(graphqlutils.MapAny2Str(input.Annotations))
	model.Spec.DisplayName = pointer.StringDeref(input.DisplayName, model.Spec.DisplayName)
	model.Spec.Description = pointer.StringDeref(input.Description, model.Spec.Description)
	model.Spec.Types = pointer.StringDeref(input.Types, model.Spec.Types)

	if model.Spec.ModelSource == common.ModelSourceModelscope {
		if *input.Revision == "" {
			return nil, errors.New("argument revision is required")
		}
		model.Spec.ModelScopeRepo = *input.ModelScopeRepo
		model.Spec.Revision = *input.Revision
	}
	if model.Spec.ModelSource == common.ModelSourceHuggingface {
		model.Spec.HuggingFaceRepo = *input.HuggingFaceRepo
		model.Spec.Revision = *input.Revision
	}
	err = c.Update(ctx, model)
	if err != nil {
		return nil, err
	}
	return obj2model(model)
}

func DeleteModels(ctx context.Context, c client.Client, input *generated.DeleteCommonInput) (*string, error) {
	opts, err := common.DeleteAllOptions(input)
	if err != nil {
		return nil, err
	}
	err = c.DeleteAllOf(ctx, &v1alpha1.Model{}, opts...)
	return nil, err
}

func ListModels(ctx context.Context, c client.Client, input generated.ListModelInput) (*generated.PaginatedResult, error) {
	filter := make([]common.ResourceFilter, 0)
	page := pointer.IntDeref(input.Page, 1)
	pageSize := pointer.IntDeref(input.PageSize, -1)
	if input.Keyword != nil {
		filter = append(filter, common.FilterModelByKeyword(*input.Keyword))
	}

	models := &v1alpha1.ModelList{}
	opts, err := common.NewListOptions(generated.ListCommonInput{
		Namespace:     input.Namespace,
		Keyword:       input.Keyword,
		LabelSelector: input.LabelSelector,
		FieldSelector: input.FieldSelector,
		Page:          input.Page,
		PageSize:      input.PageSize,
	})
	if err != nil {
		return nil, err
	}
	err = c.List(ctx, models, opts...)
	if err != nil {
		return nil, err
	}

	// list models in kubeagi system namespace
	if input.SystemModel != nil && *input.SystemModel && input.Namespace != config.GetConfig().SystemNamespace {
		systemModels := &v1alpha1.ModelList{}
		opts, err := common.NewListOptions(generated.ListCommonInput{
			Namespace:     config.GetConfig().SystemNamespace,
			Keyword:       input.Keyword,
			LabelSelector: input.LabelSelector,
			FieldSelector: input.FieldSelector,
			Page:          input.Page,
			PageSize:      input.PageSize,
		})
		if err != nil {
			return nil, err
		}
		err = c.List(ctx, systemModels, opts...)
		if err != nil {
			return nil, err
		}
		models.Items = append(systemModels.Items, models.Items...)
	}

	items := make([]client.Object, len(models.Items))
	for i := range models.Items {
		items[i] = &models.Items[i]
	}
	return common.ListReources(items, page, pageSize, obj2modelConverter, filter...)
}

func ReadModel(ctx context.Context, c client.Client, name, namespace string) (*generated.Model, error) {
	u := &v1alpha1.Model{}
	err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, u)
	if err != nil {
		return nil, err
	}
	return obj2model(u)
}

func ModelFiles(ctx context.Context, c client.Client, modelName, namespace string, input *generated.FileFilter) (*generated.PaginatedResult, error) {
	prefix := fmt.Sprintf("model/%s/", modelName)
	keyword := ""
	if input != nil && input.Keyword != nil {
		keyword = *input.Keyword
	}

	systemClient, err := pkgclient.GetClient(nil)
	if err != nil {
		return nil, err
	}
	oss, err := common.SystemDatasourceOSS(ctx, systemClient)
	if err != nil {
		return nil, err
	}
	anyObjectInfoList, err := oss.ListObjects(ctx, namespace, miniogo.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})
	if err != nil {
		return nil, err
	}
	objectInfoList := anyObjectInfoList.([]miniogo.ObjectInfo)
	sort.Slice(objectInfoList, func(i, j int) bool {
		return objectInfoList[i].LastModified.After(objectInfoList[j].LastModified)
	})

	result := make([]generated.PageNode, 0)
	for _, obj := range objectInfoList {
		if keyword == "" || strings.Contains(obj.Key, keyword) {
			lastModified := obj.LastModified
			tf := generated.F{
				Path: strings.TrimPrefix(obj.Key, prefix),
				Time: &lastModified,
			}
			size := utils.BytesToSizedStr(obj.Size)
			tf.Size = &size
			tags, err := oss.Client.GetObjectTagging(ctx, namespace, obj.Key, miniogo.GetObjectTaggingOptions{})
			if err == nil {
				tagsMap := tags.ToMap()
				if v, ok := tagsMap[v1alpha1.ObjectTypeTag]; ok {
					tf.FileType = v
				}

				if v, ok := tagsMap[v1alpha1.ObjectCountTag]; ok {
					tf.Count = &v
				}
				if v, ok := tagsMap[common.CreationTimestamp]; ok {
					if now, err := time.Parse(time.RFC3339, v); err == nil {
						tf.CreationTimestamp = &now
					}
				}
			}
			result = append(result, tf)
		}
	}
	page, size := 1, 10
	if input != nil && input.Page != nil && *input.Page > 0 {
		page = *input.Page
	}
	if input != nil && input.PageSize != nil && *input.PageSize > 0 {
		size = *input.PageSize
	}

	total := len(result)
	end := page * size
	if end > total {
		end = total
	}
	start := (page - 1) * size
	if start < total {
		result = result[start:end]
	} else {
		result = make([]generated.PageNode, 0)
	}
	return &generated.PaginatedResult{
		TotalCount:  total,
		HasNextPage: end < total,
		Nodes:       result,
		Page:        &page,
		PageSize:    &size,
	}, nil
}
