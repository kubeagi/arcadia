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

package knowledgebase

import (
	"context"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	graphqlutils "github.com/kubeagi/arcadia/apiserver/pkg/utils"
	"github.com/kubeagi/arcadia/pkg/config"
)

func knowledgebase2modelConverter(ctx context.Context, c client.Client) func(obj client.Object) (generated.PageNode, error) {
	return func(u client.Object) (generated.PageNode, error) {
		kb, ok := u.(*v1alpha1.KnowledgeBase)
		if !ok {
			return nil, errors.New("can't convert object to Knowledgebase")
		}
		return knowledgebase2model(ctx, c, kb)
	}
}

func knowledgebase2model(ctx context.Context, c client.Client, knowledgebase *v1alpha1.KnowledgeBase) (*generated.KnowledgeBase, error) {
	id := string(knowledgebase.GetUID())

	creationtimestamp := knowledgebase.GetCreationTimestamp().Time

	// conditioned status
	condition := knowledgebase.Status.GetCondition(v1alpha1.TypeReady)
	status := common.GetObjStatus(knowledgebase)
	reason := string(condition.Reason)
	message := condition.Message

	// if delete timestamp is not nil, mark status as Deletting
	if knowledgebase.DeletionTimestamp != nil {
		status = "Deleting"
	}

	var filegroupdetails []*generated.Filegroupdetail
	for _, filegroupdetail := range knowledgebase.Status.FileGroupDetail {
		var filedetails []*generated.Filedetail
		fns := filegroupdetail.Source.Namespace
		for _, detail := range filegroupdetail.FileDetails {
			filedetail := &generated.Filedetail{
				FileType:        detail.Type,
				Count:           detail.Count,
				Size:            detail.Size,
				Path:            detail.Path,
				Phase:           string(detail.Phase),
				UpdateTimestamp: &detail.LastUpdateTime.Time,
			}
			filedetails = append(filedetails, filedetail)
		}
		filegroupdetail := &generated.Filegroupdetail{
			Source: &generated.TypedObjectReference{
				Kind:      filegroupdetail.Source.Kind,
				Name:      filegroupdetail.Source.Name,
				Namespace: fns,
			},
			Filedetails: filedetails,
		}
		filegroupdetails = append(filegroupdetails, filegroupdetail)
	}

	embedderResource := &v1alpha1.Embedder{}
	embedder := generated.TypedObjectReference{
		Kind:      knowledgebase.Spec.Embedder.Kind,
		Name:      knowledgebase.Spec.Embedder.Name,
		Namespace: knowledgebase.Spec.Embedder.Namespace,
	}
	err := c.Get(ctx, types.NamespacedName{Namespace: knowledgebase.Spec.Embedder.GetNamespace(knowledgebase.Namespace), Name: knowledgebase.Spec.Embedder.Name}, embedderResource)
	// read displayname
	var embedderType string
	if err != nil {
		displayName := fmt.Sprintf("Unknown: %s", err.Error())
		embedder.DisplayName = &displayName
		embedderType = "Unknown"
	} else {
		embedder.DisplayName = &embedderResource.Spec.DisplayName
		embedderType = string(embedderResource.Spec.Provider.GetType())
	}

	embeddingOptions := knowledgebase.EmbeddingOptions()

	md := generated.KnowledgeBase{
		ID:                &id,
		Name:              knowledgebase.GetName(),
		Namespace:         knowledgebase.GetNamespace(),
		Creator:           &knowledgebase.Spec.Creator,
		Labels:            graphqlutils.MapStr2Any(knowledgebase.GetLabels()),
		Annotations:       graphqlutils.MapStr2Any(knowledgebase.GetAnnotations()),
		DisplayName:       &knowledgebase.Spec.DisplayName,
		Description:       &knowledgebase.Spec.Description,
		CreationTimestamp: &creationtimestamp,
		UpdateTimestamp:   &condition.LastTransitionTime.Time,
		// Embedder info
		Embedder:     &embedder,
		EmbedderType: &embedderType,
		// Vector info
		VectorStore: &generated.TypedObjectReference{
			Kind:      knowledgebase.Spec.VectorStore.Kind,
			Name:      knowledgebase.Spec.VectorStore.Name,
			Namespace: knowledgebase.Spec.VectorStore.Namespace,
		},
		FileGroupDetails: filegroupdetails,
		ChunkSize:        &embeddingOptions.ChunkSize,
		ChunkOverlap:     embeddingOptions.ChunkOverlap,
		BatchSize:        &embeddingOptions.BatchSize,

		// Status info
		Status:  &status,
		Reason:  &reason,
		Message: &message,
	}
	return &md, nil
}

func CreateKnowledgeBase(ctx context.Context, c client.Client, input generated.CreateKnowledgeBaseInput) (*generated.KnowledgeBase, error) {
	var filegroups []v1alpha1.FileGroup
	var vectorstore v1alpha1.TypedObjectReference
	vector, _ := config.GetVectorStore(ctx, c)
	displayname, description, embedder := "", "", ""
	if input.DisplayName != nil {
		displayname = *input.DisplayName
	}
	if input.Description != nil {
		description = *input.Description
	}
	if input.VectorStore != nil {
		vectorstore = v1alpha1.TypedObjectReference(*input.VectorStore)
	} else {
		vectorstore = *vector
	}
	if input.Embedder != "" {
		embedder = input.Embedder
	}
	if input.FileGroups != nil {
		for _, f := range input.FileGroups {
			filegroup := v1alpha1.FileGroup{
				Source: (*v1alpha1.TypedObjectReference)(&f.Source),
				Paths:  f.Path,
			}
			filegroups = append(filegroups, filegroup)
		}
	}

	// Embedding options
	chunkSize := v1alpha1.DefaultChunkSize
	if input.ChunkSize != nil {
		chunkSize = *input.ChunkSize
	}
	chunkOverlap := pointer.Int(v1alpha1.DefaultChunkOverlap)
	if input.ChunkOverlap != nil {
		chunkOverlap = input.ChunkOverlap
	}
	batchSize := v1alpha1.DefaultBatchSize
	if input.BatchSize != nil {
		batchSize = *input.BatchSize
	}

	knowledgebase := &v1alpha1.KnowledgeBase{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
		},
		Spec: v1alpha1.KnowledgeBaseSpec{
			CommonSpec: v1alpha1.CommonSpec{
				DisplayName: displayname,
				Description: description,
			},
			Embedder: &v1alpha1.TypedObjectReference{
				Kind:      "Embedder",
				Name:      embedder,
				Namespace: &input.Namespace,
			},
			VectorStore: &vectorstore,
			FileGroups:  filegroups,
			EmbeddingOptions: v1alpha1.EmbeddingOptions{
				ChunkSize:    chunkSize,
				ChunkOverlap: chunkOverlap,
				BatchSize:    batchSize,
			},
		},
	}
	common.SetCreator(ctx, &knowledgebase.Spec.CommonSpec)

	err := c.Create(ctx, knowledgebase)
	if err != nil {
		return nil, err
	}
	kb, err := knowledgebase2model(ctx, c, knowledgebase)
	if err != nil {
		return nil, err
	}
	if kb.FileGroupDetails == nil {
		// fill in file group without any details
		details := make([]*generated.Filegroupdetail, len(filegroups))
		for index, fg := range filegroups {
			fgDetail := &generated.Filegroupdetail{
				Source: &generated.TypedObjectReference{
					APIGroup:  fg.Source.APIGroup,
					Kind:      fg.Source.Kind,
					Name:      fg.Source.Name,
					Namespace: fg.Source.Namespace,
				},
			}
			fileDetails := make([]*generated.Filedetail, len(fg.Paths))
			for findex, path := range fg.Paths {
				fileDetails[findex] = &generated.Filedetail{
					Path:  path,
					Phase: "",
				}
			}
			fgDetail.Filedetails = fileDetails
			details[index] = fgDetail
		}
		kb.FileGroupDetails = details
	}
	return kb, nil
}

func UpdateKnowledgeBase(ctx context.Context, c client.Client, input *generated.UpdateKnowledgeBaseInput) (*generated.KnowledgeBase, error) {
	kb := &v1alpha1.KnowledgeBase{}
	err := c.Get(ctx, types.NamespacedName{Namespace: input.Namespace, Name: input.Name}, kb)
	if err != nil {
		return nil, err
	}

	if input.DisplayName != nil && *input.DisplayName != kb.Spec.DisplayName {
		kb.Spec.DisplayName = *input.DisplayName
	}
	if input.Description != nil && *input.Description != kb.Spec.Description {
		kb.Spec.Description = *input.Description
	}

	if input.FileGroups != nil {
		filegroups := make([]v1alpha1.FileGroup, len(input.FileGroups))
		for index, f := range input.FileGroups {
			filegroup := v1alpha1.FileGroup{
				Source: (*v1alpha1.TypedObjectReference)(&f.Source),
				Paths:  f.Path,
			}
			filegroups[index] = filegroup
		}
		kb.Spec.FileGroups = filegroups
	}

	if input.ChunkSize != nil {
		kb.Spec.ChunkSize = *input.ChunkSize
	}
	if input.ChunkOverlap != nil {
		kb.Spec.ChunkOverlap = input.ChunkOverlap
	}
	if input.BatchSize != nil {
		kb.Spec.BatchSize = *input.BatchSize
	}

	err = c.Update(ctx, kb)
	if err != nil {
		return nil, err
	}

	return knowledgebase2model(ctx, c, kb)
}

func DeleteKnowledgeBase(ctx context.Context, c client.Client, name, namespace, labelSelector, fieldSelector string) (*string, error) {
	opts, err := common.DeleteAllOptions(&generated.DeleteCommonInput{
		Name:          &name,
		Namespace:     namespace,
		LabelSelector: &labelSelector,
		FieldSelector: &fieldSelector,
	})
	if err != nil {
		return nil, err
	}
	err = c.DeleteAllOf(ctx, &v1alpha1.KnowledgeBase{}, opts...)
	return nil, err
}

func ReadKnowledgeBase(ctx context.Context, c client.Client, name, namespace string) (*generated.KnowledgeBase, error) {
	kb := &v1alpha1.KnowledgeBase{}
	err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, kb)
	if err != nil {
		return nil, err
	}
	return knowledgebase2model(ctx, c, kb)
}

func ListKnowledgeBases(ctx context.Context, c client.Client, input generated.ListKnowledgeBaseInput) (*generated.PaginatedResult, error) {
	page, pageSize := 1, 10
	filter := make([]common.ResourceFilter, 0)
	if input.Name != nil {
		filter = append(filter, common.FilterByNameContains(*input.Name))
	}
	if input.DisplayName != nil {
		filter = append(filter, common.FilterKnowledgeByDisplayName(*input.DisplayName))
	}
	if input.Page != nil && *input.Page > 0 {
		page = *input.Page
	}
	if input.PageSize != nil && *input.PageSize > 0 {
		pageSize = *input.PageSize
	}

	us := &v1alpha1.KnowledgeBaseList{}
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
	err = c.List(ctx, us, opts...)
	if err != nil {
		return nil, err
	}
	items := make([]client.Object, len(us.Items))
	for i := range us.Items {
		items[i] = &us.Items[i]
	}
	return common.ListReources(items, page, pageSize, knowledgebase2modelConverter(ctx, c), filter...)
}
