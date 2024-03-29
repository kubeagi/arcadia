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

package versioneddataset

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	miniogo "github.com/minio/minio-go/v7"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	pkgclient "github.com/kubeagi/arcadia/apiserver/pkg/client"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	graphqlutils "github.com/kubeagi/arcadia/apiserver/pkg/utils"
	"github.com/kubeagi/arcadia/pkg/utils"
)

func versionedDataset2modelConverter(obj client.Object) (generated.PageNode, error) {
	vd, ok := obj.(*v1alpha1.VersionedDataset)
	if !ok {
		return nil, errors.New("can't convert object to VersionedDataset")
	}
	return versionedDataset2model(vd)
}

func versionedDataset2model(versioneddataset *v1alpha1.VersionedDataset) (*generated.VersionedDataset, error) {
	vds := &generated.VersionedDataset{}
	id := string(versioneddataset.GetUID())
	vds.ID = &id
	vds.Name = versioneddataset.GetName()
	vds.Namespace = versioneddataset.GetNamespace()
	vds.Labels = graphqlutils.MapStr2Any(versioneddataset.GetLabels())
	vds.Annotations = graphqlutils.MapStr2Any(versioneddataset.GetAnnotations())
	vds.CreationTimestamp = versioneddataset.GetCreationTimestamp().Time
	vds.UpdateTimestamp = &vds.CreationTimestamp
	vds.Creator = &versioneddataset.Spec.Creator
	vds.DisplayName = &versioneddataset.Spec.DisplayName
	vds.Description = &versioneddataset.Spec.Description
	vds.Dataset = generated.TypedObjectReference{
		APIGroup:  versioneddataset.Spec.Dataset.APIGroup,
		Kind:      versioneddataset.Spec.Dataset.Kind,
		Name:      versioneddataset.Spec.Dataset.Name,
		Namespace: versioneddataset.Spec.Dataset.Namespace,
	}

	vds.Version = versioneddataset.Spec.Version
	vds.SyncStatus = new(string)
	vds.SyncMsg = new(string)
	vds.DataProcessStatus = new(string)
	vds.DataProcessMsg = new(string)
	first := true
	for _, cond := range versioneddataset.Status.Conditions {
		cond := cond
		if cond.Type == v1alpha1.TypeReady {
			*vds.SyncStatus = string(cond.Reason)
			*vds.SyncMsg = cond.Message
		}
		if cond.Type == v1alpha1.TypeDataProcessing {
			*vds.DataProcessStatus = string(cond.Reason)
			*vds.DataProcessMsg = cond.Message
		}
		if !cond.LastTransitionTime.IsZero() {
			if first || vds.UpdateTimestamp.Before(cond.LastSuccessfulTime.Time) {
				vds.UpdateTimestamp = &cond.LastTransitionTime.Time
				first = false
			}
		}
	}

	if s := common.GetObjStatus(versioneddataset); s != "" {
		*vds.SyncStatus = s
	}
	vds.Released = int(versioneddataset.Spec.Released)
	return vds, nil
}

func VersionFiles(ctx context.Context, _ client.Client, input *generated.VersionedDataset, filter *generated.FileFilter) (*generated.PaginatedResult, error) {
	prefix := fmt.Sprintf("dataset/%s/%s/", input.Dataset.Name, input.Version)
	keyword := ""
	if filter != nil && filter.Keyword != nil {
		keyword = *filter.Keyword
	}

	systemClient, err := pkgclient.GetClient(nil)
	if err != nil {
		return nil, err
	}
	oss, err := common.SystemDatasourceOSS(ctx, systemClient)
	if err != nil {
		return nil, err
	}
	anyObjectInfoList, err := oss.ListObjects(ctx, input.Namespace, miniogo.ListObjectsOptions{
		Prefix:       prefix,
		Recursive:    true,
		WithVersions: true,
	})
	if err != nil {
		return nil, err
	}
	objectInfoList := anyObjectInfoList.([]miniogo.ObjectInfo)
	sort.Slice(objectInfoList, func(i, j int) bool {
		return objectInfoList[i].LastModified.After(objectInfoList[j].LastModified)
	})

	existMap := make(map[string]struct{})
	objMap := make(map[string][]string)
	for _, obj := range objectInfoList {
		objMap[obj.Key] = append(objMap[obj.Key], obj.VersionID)
	}

	result := make([]generated.PageNode, 0)
	for _, obj := range objectInfoList {
		if _, ok := existMap[obj.Key]; ok {
			continue
		}
		if obj.IsDeleteMarker {
			continue
		}
		if keyword == "" || strings.Contains(obj.Key, keyword) {
			existMap[obj.Key] = struct{}{}
			lastModifiedTime := obj.LastModified
			tf := generated.F{
				Path:     strings.TrimPrefix(obj.Key, prefix),
				Time:     &lastModifiedTime,
				Versions: objMap[obj.Key],
			}

			size := utils.BytesToSizedStr(obj.Size)
			tf.Size = &size

			// parse tags
			tags, err := oss.Client.GetObjectTagging(ctx, input.Namespace, obj.Key, miniogo.GetObjectTaggingOptions{})
			if err == nil {
				tagsMap := tags.ToMap()
				if v, ok := tagsMap[v1alpha1.ObjectTypeTag]; ok {
					tf.FileType = v
				}

				if v, ok := tagsMap[v1alpha1.ObjectCountTag]; ok {
					tf.Count = &v
				}
			}

			if v, ok := obj.UserTags[common.CreationTimestamp]; ok {
				if now, err := time.Parse(time.RFC3339, v); err == nil {
					tf.CreationTimestamp = &now
				}
			}
			result = append(result, tf)
		}
	}
	page, size := 1, 10
	if filter != nil && filter.Page != nil && *filter.Page > 0 {
		page = *filter.Page
	}
	if filter != nil && filter.PageSize != nil && *filter.PageSize > 0 {
		size = *filter.PageSize
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

func ListVersionedDatasets(ctx context.Context, c client.Client, input *generated.ListVersionedDatasetInput) (*generated.PaginatedResult, error) {
	page := pointer.IntDeref(input.Page, 1)
	size := pointer.IntDeref(input.PageSize, -1)
	ns := "default"
	if input.Namespace != nil {
		ns = *input.Namespace
	}
	filter := make([]common.ResourceFilter, 0)
	if input.DisplayName != nil {
		filter = append(filter, common.FilterVersionedDatasetByDisplayName(*input.DisplayName))
	}
	if input.Keyword != nil {
		filter = append(filter, common.FilterVersionedDatasetByKeyword(*input.Keyword))
	}

	us := &v1alpha1.VersionedDatasetList{}
	opts, err := common.NewListOptions(generated.ListCommonInput{
		Namespace:     ns,
		Keyword:       input.Keyword,
		LabelSelector: input.LabelSelector,
		FieldSelector: input.FieldSelector,
		Page:          input.Page,
		PageSize:      input.PageSize,
	})
	if err != nil {
		return nil, err
	}
	err = c.List(ctx, us, opts...)
	if err != nil {
		return nil, err
	}
	items := make([]client.Object, len(us.Items))
	for i := range us.Items {
		items[i] = &us.Items[i]
	}
	return common.ListReources(items, page, size, versionedDataset2modelConverter, filter...)
}

func GetVersionedDataset(ctx context.Context, c client.Client, name, namespace string) (*generated.VersionedDataset, error) {
	obj := &v1alpha1.VersionedDataset{}
	err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, obj)
	if err != nil {
		return nil, err
	}
	return versionedDataset2model(obj)
}

func DeleteVersionedDatasets(ctx context.Context, c client.Client, input *generated.DeleteVersionedDatasetInput) (*string, error) {
	none := ""
	opts, err := common.DeleteAllOptions(&generated.DeleteCommonInput{
		Name:          input.Name,
		Namespace:     input.Namespace,
		LabelSelector: input.LabelSelector,
		FieldSelector: input.FieldSelector,
	})
	if err != nil {
		return nil, err
	}
	err = c.DeleteAllOf(ctx, &v1alpha1.VersionedDataset{}, opts...)
	return &none, err
}

func UpdateVersionedDataset(ctx context.Context, c client.Client, input *generated.UpdateVersionedDatasetInput) (*generated.VersionedDataset, error) {
	obj := &v1alpha1.VersionedDataset{}
	err := c.Get(ctx, types.NamespacedName{Namespace: input.Namespace, Name: input.Name}, obj)
	if err != nil {
		return nil, err
	}
	obj.SetLabels(graphqlutils.MapAny2Str(input.Labels))
	obj.SetAnnotations(graphqlutils.MapAny2Str(input.Annotations))
	if input.Released != nil {
		obj.Spec.Released = uint8(*input.Released)
	}
	obj.Spec.DisplayName = pointer.StringDeref(input.DisplayName, obj.Spec.DisplayName)
	obj.Spec.Description = pointer.StringDeref(input.Description, obj.Spec.Description)
	fg := make([]v1alpha1.FileGroup, 0)
	for _, item := range input.FileGroups {
		tmp := v1alpha1.FileGroup{
			Source: &v1alpha1.TypedObjectReference{
				Kind:      item.Source.Kind,
				Name:      item.Source.Name,
				Namespace: item.Source.Namespace,
			},
			Files: make([]v1alpha1.FileWithVersion, 0),
		}
		for _, fv := range item.Files {
			tmpFv := v1alpha1.FileWithVersion{
				Path:    fv.Path,
				Version: "",
			}
			if fv.Version != nil {
				tmpFv.Version = *fv.Version
			}
			tmp.Files = append(tmp.Files, tmpFv)
		}
		fg = append(fg, tmp)
	}
	obj.Spec.FileGroups = fg
	err = c.Update(ctx, obj)
	if err != nil {
		return nil, err
	}
	return versionedDataset2model(obj)
}

func CreateVersionedDataset(ctx context.Context, c client.Client, input *generated.CreateVersionedDatasetInput) (*generated.VersionedDataset, error) {
	vds := &v1alpha1.VersionedDataset{}
	vds.Name = input.Name
	vds.Namespace = input.Namespace
	vds.Spec = v1alpha1.VersionedDatasetSpec{
		Version: input.Version,
		Dataset: &v1alpha1.TypedObjectReference{
			Kind:      "Dataset",
			Name:      input.DatasetName,
			Namespace: &input.Namespace,
		},
		Released: 0,
	}
	vds.Spec.DisplayName = pointer.StringDeref(input.DisplayName, vds.Spec.DisplayName)
	vds.Spec.Description = pointer.StringDeref(input.Description, vds.Spec.Description)
	common.SetCreator(ctx, &vds.Spec.CommonSpec)
	vds.SetLabels(graphqlutils.MapAny2Str(input.Labels))
	vds.SetAnnotations(graphqlutils.MapAny2Str(input.Annotations))
	if len(input.FileGrups) > 0 {
		fg := make([]v1alpha1.FileGroup, 0)
		for _, item := range input.FileGrups {
			tmp := v1alpha1.FileGroup{
				Source: &v1alpha1.TypedObjectReference{
					Kind:      item.Source.Kind,
					Name:      item.Source.Name,
					Namespace: item.Source.Namespace,
				},
				Files: make([]v1alpha1.FileWithVersion, 0),
			}
			for _, fv := range item.Files {
				tmpFv := v1alpha1.FileWithVersion{
					Path:    fv.Path,
					Version: "",
				}
				if fv.Version != nil {
					tmpFv.Version = *fv.Version
				}
				tmp.Files = append(tmp.Files, tmpFv)
			}
			fg = append(fg, tmp)
		}
		vds.Spec.FileGroups = fg
	}
	if input.InheritedFrom != nil {
		vds.Spec.InheritedFrom = *input.InheritedFrom
		// 选中目标的versionDataset
		versionList := &v1alpha1.VersionedDatasetList{}
		err := c.List(ctx, versionList, client.MatchingLabels{v1alpha1.LabelVersionedDatasetVersionOwner: input.DatasetName})
		if err != nil {
			return nil, err
		}

		for _, v := range versionList.Items {
			if v.Spec.Version == *input.InheritedFrom {
				isReady := false
				var errMessage error
				for _, cond := range v.Status.Conditions {
					if cond.Type == v1alpha1.TypeReady && cond.Status == v1.ConditionTrue {
						isReady = true
						break
					}
					if cond.Type == v1alpha1.TypeReady && cond.Status != v1.ConditionTrue {
						errMessage = fmt.Errorf("inherit from a version with an incorrect synchronization state will not be created. reason: %s, errMsg: %s", cond.Reason, cond.Message)
					}
				}
				if !isReady {
					return nil, errMessage
				}
			}
		}
	}
	err := c.Create(ctx, vds)
	if err != nil {
		return nil, err
	}
	return versionedDataset2model(vds)
}
