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
package rag

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/minio/minio-go/v7"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	evav1alpha1 "github.com/kubeagi/arcadia/api/evaluation/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	graphqlutils "github.com/kubeagi/arcadia/apiserver/pkg/utils"
	"github.com/kubeagi/arcadia/pkg/utils"
)

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyz0123456789"
)

func generateRandomString(length int) string {
	r := rand.New(rand.NewSource(time.Now().UnixMicro()))
	b := make([]byte, length)
	for i := range b {
		b[i] = letterBytes[r.Intn(len(letterBytes))]
	}
	return string(b)
}

func generateKubernetesResourceName(prefix string, length int) string {
	randomString := generateRandomString(length)
	return fmt.Sprintf("%s-%s", prefix, randomString)
}
func setRAGSatus(rag *evav1alpha1.RAG, r *generated.Rag) {
	status, phase, phaseMsg := evav1alpha1.RagStatus(rag)
	*r.Phase = string(phase)
	*r.PhaseMessage = phaseMsg
	r.Status = status
	r.Suspend = rag.Spec.Suspend
}

func gen2storage(p generated.PersistentVolumeClaimSpecInput) *corev1.PersistentVolumeClaimSpec {
	pvc := &corev1.PersistentVolumeClaimSpec{
		VolumeName: p.VolumeName,
	}
	if len(pvc.AccessModes) > 0 {
		pvc.AccessModes = make([]corev1.PersistentVolumeAccessMode, 0)
		for _, s := range pvc.AccessModes {
			pvc.AccessModes = append(pvc.AccessModes, corev1.PersistentVolumeAccessMode(s))
		}
	}
	if p.Selector != nil {
		pvc.Selector = &v1.LabelSelector{}
		if len(p.Selector.MatchLabels) > 0 {
			pvc.Selector.MatchLabels = make(map[string]string)
			for k, v := range p.Selector.MatchLabels {
				pvc.Selector.MatchLabels[k] = v.(string)
			}
		}
		if len(p.Selector.MatchExpressions) > 0 {
			pvc.Selector.MatchExpressions = make([]v1.LabelSelectorRequirement, 0)
			for _, item := range p.Selector.MatchExpressions {
				i := v1.LabelSelectorRequirement{
					Key:      *item.Key,
					Values:   make([]string, 0),
					Operator: v1.LabelSelectorOperator(*item.Operator),
				}
				for _, s := range item.Values {
					i.Values = append(i.Values, *s)
				}
				pvc.Selector.MatchExpressions = append(pvc.Selector.MatchExpressions, i)
			}
		}
	}
	if p.Resources != nil {
		pvc.Resources = corev1.ResourceRequirements{}
		if len(p.Resources.Limits) > 0 {
			pvc.Resources.Limits = make(corev1.ResourceList)
			for k, v := range p.Resources.Limits {
				q, _ := resource.ParseQuantity(v.(string))
				pvc.Resources.Limits[corev1.ResourceName(k)] = q
			}
		}
		if len(p.Resources.Requests) > 0 {
			pvc.Resources.Requests = make(corev1.ResourceList)
			for k, v := range p.Resources.Requests {
				q, _ := resource.ParseQuantity(v.(string))
				pvc.Resources.Requests[corev1.ResourceName(k)] = q
			}
		}
	}

	if p.StorageClassName != nil {
		pvc.StorageClassName = p.StorageClassName
	}
	if p.VolumeMode != nil {
		a := corev1.PersistentVolumeMode(*p.VolumeMode)
		pvc.VolumeMode = &a
	}
	// TODO set datasource
	return pvc
}
func storage2gen(pvcSpec *corev1.PersistentVolumeClaimSpec) generated.PersistentVolumeClaimSpec {
	pvc := generated.PersistentVolumeClaimSpec{}
	if pvcSpec.VolumeName != "" {
		pvc.VolumeName = new(string)
		*pvc.VolumeName = pvcSpec.VolumeName
	}
	if pvcSpec.StorageClassName != nil {
		pvc.StorageClassName = new(string)
		*pvc.StorageClassName = *pvcSpec.StorageClassName
	}
	if pvcSpec.VolumeMode != nil {
		pvc.VolumeMode = new(string)
		*pvc.VolumeMode = string(*pvcSpec.VolumeMode)
	}

	for _, am := range pvcSpec.AccessModes {
		pvc.AccessModes = append(pvc.AccessModes, string(am))
	}

	if pvcSpec.Selector != nil {
		pvc.Selector = new(generated.Selector)
		if len(pvcSpec.Selector.MatchLabels) > 0 {
			pvc.Selector.MatchLabels = make(map[string]interface{})
			for k, v := range pvcSpec.Selector.MatchLabels {
				pvc.Selector.MatchLabels[k] = v
			}
		}
		if len(pvcSpec.Selector.MatchExpressions) > 0 {
			pvc.Selector.MatchExpressions = make([]*generated.LabelSelectorRequirement, 0)
			for _, item := range pvcSpec.Selector.MatchExpressions {
				a := &generated.LabelSelectorRequirement{
					Key:      new(string),
					Operator: new(string),
					Values:   make([]*string, 0),
				}
				*a.Key = item.Key
				*a.Operator = string(item.Operator)
				for i := 0; i < len(item.Values); i++ {
					a.Values = append(a.Values, &item.Values[i])
				}
			}
		}
	}

	if len(pvcSpec.Resources.Requests) > 0 || len(pvcSpec.Resources.Limits) > 0 {
		pvc.Resources = new(generated.Resource)
	}
	if len(pvcSpec.Resources.Limits) > 0 {
		pvc.Resources.Limits = make(map[string]interface{})
		for k, v := range pvcSpec.Resources.Limits {
			pvc.Resources.Limits[string(k)] = v
		}
	}
	if len(pvcSpec.Resources.Requests) > 0 {
		pvc.Resources.Requests = make(map[string]interface{})
		for k, v := range pvcSpec.Resources.Requests {
			pvc.Resources.Requests[string(k)] = v
		}
	}

	if pvcSpec.DataSource != nil {
		pvc.Datasource = new(generated.TypedObjectReference)
		pvc.Datasource.APIGroup = pvcSpec.DataSource.APIGroup
		pvc.Datasource.Kind = pvcSpec.DataSource.Kind
		pvc.Datasource.Name = pvcSpec.DataSource.Name
	}
	if pvcSpec.DataSourceRef != nil {
		pvc.DataSourceRef = new(generated.TypedObjectReference)
		pvc.DataSourceRef.APIGroup = pvcSpec.DataSourceRef.APIGroup
		pvc.DataSourceRef.Kind = pvcSpec.DataSource.Kind
		pvc.DataSourceRef.Name = pvcSpec.DataSource.Name
	}

	return pvc
}

func rag2modelConverter(u *unstructured.Unstructured) (generated.PageNode, error) {
	return rag2model(u)
}

func rag2model(o *unstructured.Unstructured) (*generated.Rag, error) {
	r := &generated.Rag{
		Name:               o.GetName(),
		Namespace:          o.GetNamespace(),
		Labels:             map[string]interface{}{},
		Annotations:        map[string]interface{}{},
		Creator:            new(string),
		DisplayName:        new(string),
		Description:        new(string),
		CreationTimestamp:  new(time.Time),
		CompleteTimestamp:  new(time.Time),
		ServiceAccountName: "",
		Suspend:            false,
		Status:             "",
		Phase:              new(string),
		PhaseMessage:       new(string),
	}

	structuredRag := evav1alpha1.RAG{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(o.Object, &structuredRag)
	if err != nil {
		return nil, err
	}
	for k, v := range o.GetLabels() {
		r.Labels[k] = v
	}
	for k, v := range o.GetAnnotations() {
		r.Annotations[k] = v
	}
	*r.Creator = structuredRag.Spec.Creator
	*r.DisplayName = structuredRag.Spec.DisplayName
	*r.Description = structuredRag.Spec.Description
	*r.CreationTimestamp = o.GetCreationTimestamp().Time
	if structuredRag.Status.CompletionTime != nil {
		*r.CompleteTimestamp = structuredRag.Status.CompletionTime.Time
	}
	r.ServiceAccountName = structuredRag.Spec.ServiceAccountName
	r.Storage = storage2gen(structuredRag.Spec.Storage)
	setRAGSatus(&structuredRag, r)
	return r, nil
}

func CreateRAG(ctx context.Context, kubeClient dynamic.Interface, input *generated.CreateRAGInput) (*generated.Rag, error) {
	rag := &evav1alpha1.RAG{
		TypeMeta: v1.TypeMeta{
			Kind:       "RAG",
			APIVersion: evav1alpha1.GroupVersion.String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Namespace:   input.Namespace,
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
		},
		Spec: evav1alpha1.RAGSpec{},
	}
	name := generateKubernetesResourceName("rag", 10)
	if input.Name != nil {
		name = *input.Name
	}
	rag.Name = name
	if input.DisplayName != nil {
		rag.Spec.DisplayName = *input.DisplayName
	}
	if input.Description != nil {
		rag.Spec.Description = *input.Description
	}
	rag.Spec.Application = &v1alpha1.TypedObjectReference{
		APIGroup:  input.Application.APIGroup,
		Kind:      input.Application.Kind,
		Name:      input.Application.Name,
		Namespace: input.Application.Namespace,
	}
	rag.Spec.Datasets = make([]evav1alpha1.Dataset, 0)
	for i, set := range input.Datasets {
		ds := evav1alpha1.Dataset{
			Source: &v1alpha1.TypedObjectReference{
				APIGroup:  input.Datasets[i].Source.APIGroup,
				Kind:      input.Datasets[i].Source.Kind,
				Name:      input.Datasets[i].Source.Name,
				Namespace: input.Datasets[i].Source.Namespace,
			},
			Files: set.Files,
		}
		rag.Spec.Datasets = append(rag.Spec.Datasets, ds)
	}

	rag.Spec.JudgeLLM = &v1alpha1.TypedObjectReference{
		APIGroup:  input.JudgeLlm.APIGroup,
		Kind:      input.JudgeLlm.Kind,
		Name:      input.JudgeLlm.Name,
		Namespace: input.JudgeLlm.Namespace,
	}

	rag.Spec.Metrics = make([]evav1alpha1.Metric, 0)
	for _, m := range input.Metrics {
		mm := evav1alpha1.Metric{
			Parameters: make([]evav1alpha1.Parameter, 0),
		}
		if m.MetricKind != nil {
			mm.Kind = evav1alpha1.MetricsKind(*m.MetricKind)
		}
		if m.ToleranceThreshbold != nil {
			mm.ToleranceThreshbold = *m.ToleranceThreshbold
		}
		for _, p := range m.Parameters {
			mm.Parameters = append(mm.Parameters, evav1alpha1.Parameter{
				Key:   *p.Key,
				Value: *p.Value,
			})
		}
		rag.Spec.Metrics = append(rag.Spec.Metrics, mm)
	}

	rag.Spec.Storage = gen2storage(input.Storage)
	if input.ServiceAccountName != nil {
		rag.Spec.ServiceAccountName = *input.ServiceAccountName
	}
	if input.Suspend != nil {
		rag.Spec.Suspend = *input.Suspend
	}

	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(rag)
	if err != nil {
		return nil, err
	}
	u1, err := kubeClient.Resource(schema.GroupVersionResource{Group: evav1alpha1.Group, Version: evav1alpha1.Version, Resource: "rags"}).Namespace(input.Namespace).Create(ctx, &unstructured.Unstructured{Object: u}, v1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return rag2model(u1)
}

func UpdateRAG(ctx context.Context, kubeClient dynamic.Interface, input *generated.UpdateRAGInput) (*generated.Rag, error) {
	obj, err := kubeClient.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "RAG")).Namespace(input.Namespace).Get(ctx, input.Name, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	rag := &evav1alpha1.RAG{}
	if err := utils.UnstructuredToStructured(obj, rag); err != nil {
		return nil, err
	}

	if input.Labels != nil {
		rag.SetLabels(graphqlutils.MapAny2Str(input.Labels))
	}
	if input.Annotations != nil {
		rag.SetAnnotations(graphqlutils.MapAny2Str(input.Annotations))
	}
	if input.DisplayName != nil {
		rag.Spec.DisplayName = *input.DisplayName
	}
	if input.Description != nil {
		rag.Spec.Description = *input.Description
	}
	if input.Application != nil {
		rag.Spec.Application = &v1alpha1.TypedObjectReference{
			APIGroup:  input.Application.APIGroup,
			Kind:      input.Application.Kind,
			Name:      input.Application.Name,
			Namespace: input.Application.Namespace,
		}
	}
	if input.Datasets != nil {
		rag.Spec.Datasets = make([]evav1alpha1.Dataset, 0)
		for i, dataset := range input.Datasets {
			ds := evav1alpha1.Dataset{
				Source: &v1alpha1.TypedObjectReference{
					APIGroup:  input.Datasets[i].Source.APIGroup,
					Kind:      input.Datasets[i].Source.Kind,
					Name:      input.Datasets[i].Source.Name,
					Namespace: input.Datasets[i].Source.Namespace,
				},
				Files: dataset.Files,
			}
			rag.Spec.Datasets = append(rag.Spec.Datasets, ds)
		}
	}
	if input.JudgeLlm != nil {
		rag.Spec.JudgeLLM = &v1alpha1.TypedObjectReference{
			APIGroup:  input.JudgeLlm.APIGroup,
			Kind:      input.JudgeLlm.Kind,
			Name:      input.JudgeLlm.Name,
			Namespace: input.JudgeLlm.Namespace,
		}
	}
	if input.Metrics != nil {
		rag.Spec.Metrics = make([]evav1alpha1.Metric, 0)
		for _, m := range input.Metrics {
			mm := evav1alpha1.Metric{
				Parameters: make([]evav1alpha1.Parameter, 0),
			}
			if m.MetricKind != nil {
				mm.Kind = evav1alpha1.MetricsKind(*m.MetricKind)
			}
			if m.ToleranceThreshbold != nil {
				mm.ToleranceThreshbold = *m.ToleranceThreshbold
			}
			for _, p := range m.Parameters {
				mm.Parameters = append(mm.Parameters, evav1alpha1.Parameter{
					Key:   *p.Key,
					Value: *p.Value,
				})
			}
			rag.Spec.Metrics = append(rag.Spec.Metrics, mm)
		}
	}

	if input.Storage != nil {
		rag.Spec.Storage = gen2storage(*input.Storage)
	}
	if input.Suspend != nil {
		rag.Spec.Suspend = *input.Suspend
	}

	unstructuredRag, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&rag)
	if err != nil {
		return nil, err
	}
	updatedRag, err := common.ResouceUpdate(ctx, kubeClient, generated.TypedObjectReferenceInput{
		APIGroup:  &common.ArcadiaAPIGroup,
		Kind:      "RAG",
		Namespace: &rag.Namespace,
		Name:      rag.Name,
	}, unstructuredRag, v1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return rag2model(updatedRag)
}

func ListRAG(ctx context.Context, kubeClient dynamic.Interface, input *generated.ListRAGInput) (*generated.PaginatedResult, error) {
	listOptions := v1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", evav1alpha1.EvaluationApplicationLabel, input.AppName),
	}
	page, size := 1, 10
	if input.Page != nil && *input.Page > 0 {
		page = *input.Page
	}
	if input.PageSize != nil && *input.PageSize > 0 {
		size = *input.PageSize
	}
	filter := make([]common.ResourceFilter, 0)
	if input.Keyword != nil {
		filter = append(filter, common.FilterByRAGKeyword(*input.Keyword))
	}
	if input.Status != nil {
		filter = append(filter, common.FilterRAGByStatus(*input.Status))
	}
	list, err := kubeClient.Resource(schema.GroupVersionResource{
		Group:    evav1alpha1.Group,
		Version:  evav1alpha1.Version,
		Resource: "rags",
	}).Namespace(input.Namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}
	return common.ListReources(list, page, size, rag2modelConverter, filter...)
}

func GetRAG(ctx context.Context, kubeClient dynamic.Interface, name, namespace string) (*generated.Rag, error) {
	u, err := kubeClient.Resource(schema.GroupVersionResource{
		Group:    evav1alpha1.Group,
		Version:  evav1alpha1.Version,
		Resource: "rags",
	}).Namespace(namespace).Get(ctx, name, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return rag2model(u)
}

func GetV1alpha1RAG(ctx context.Context, kubeClient dynamic.Interface, name, namespace string) (*evav1alpha1.RAG, error) {
	u, err := kubeClient.Resource(schema.GroupVersionResource{
		Group:    evav1alpha1.Group,
		Version:  evav1alpha1.Version,
		Resource: "rags",
	}).Namespace(namespace).Get(ctx, name, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	structuredRag := evav1alpha1.RAG{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &structuredRag)
	if err != nil {
		return nil, err
	}
	return &structuredRag, nil
}

func getFiles(ctx context.Context, kubeClient dynamic.Interface, bucket string, files []string) ([]*generated.F, error) {
	oss, err := common.SystemDatasourceOSS(ctx, nil, kubeClient)
	if err != nil {
		return nil, err
	}
	fs := make([]*generated.F, 0)
	for _, f := range files {
		obj, err := oss.Client.StatObject(ctx, bucket, f, minio.GetObjectOptions{})
		if err != nil {
			return nil, err
		}

		size := utils.BytesToSizedStr(obj.Size)
		gf := &generated.F{
			Path: f,
			Size: &size,
		}
		tags, err := oss.Client.GetObjectTagging(ctx, bucket, f, minio.GetObjectTaggingOptions{})
		if err != nil {
			return nil, err
		}
		tagsMap := tags.ToMap()
		if v, ok := tagsMap[v1alpha1.ObjectTypeTag]; ok {
			gf.FileType = v
		}

		if v, ok := tagsMap[v1alpha1.ObjectCountTag]; ok {
			gf.Count = &v
		}
		fs = append(fs, gf)
	}
	return fs, nil
}
func GetRAGMetrics(ctx context.Context, kubeClient dynamic.Interface, name, namespace string) ([]*generated.RAGMetric, error) {
	rag, err := GetV1alpha1RAG(ctx, kubeClient, name, namespace)
	if err != nil {
		return nil, err
	}
	metrics := make([]*generated.RAGMetric, 0)
	for _, m := range rag.Spec.Metrics {
		mm := &generated.RAGMetric{
			ToleranceThreshbold: new(int),
			Parameters:          make([]*generated.Parameter, 0),
		}
		*mm.ToleranceThreshbold = m.ToleranceThreshbold
		if r := string(m.Kind); r != "" {
			mm.MetricKind = new(string)
			*mm.MetricKind = r
		}
		for _, p := range m.Parameters {
			pp := &generated.Parameter{
				Key:   &p.Key,
				Value: &p.Value,
			}
			mm.Parameters = append(mm.Parameters, pp)
		}
		metrics = append(metrics, mm)
	}
	return metrics, nil
}

func GetRAGDatasets(ctx context.Context, kubeClient dynamic.Interface, name, namespace string) ([]*generated.RAGDataset, error) {
	rag, err := GetV1alpha1RAG(ctx, kubeClient, name, namespace)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	nodes := make([]*generated.RAGDataset, 0)
	for _, ds := range rag.Spec.Datasets {
		// TODO, enen, versioneddataset is ok
		if ds.Source.Kind == "VersionedDataset" {
			ns := namespace
			if ds.Source.Namespace != nil {
				ns = *ds.Source.Namespace
			}
			files, err := getFiles(ctx, kubeClient, ns, ds.Files)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, &generated.RAGDataset{
				Source: &generated.TypedObjectReference{
					APIGroup:  ds.Source.APIGroup,
					Kind:      ds.Source.Kind,
					Name:      ds.Source.Name,
					Namespace: ds.Source.Namespace,
				},
				Files: files,
			})
		}
	}
	return nodes, nil
}

func DeleteRAG(ctx context.Context, kubeClient dynamic.Interface, input *generated.DeleteRAGInput) error {
	if input.Name != "" {
		return kubeClient.Resource(schema.GroupVersionResource{Group: evav1alpha1.Group, Version: evav1alpha1.Version, Resource: "rags"}).
			Namespace(input.Namespace).Delete(ctx, input.Name, v1.DeleteOptions{})
	}
	if input.LabelSelector != nil {
		return kubeClient.Resource(schema.GroupVersionResource{Group: evav1alpha1.Group, Version: evav1alpha1.Version, Resource: "rags"}).
			Namespace(input.Namespace).DeleteCollection(ctx, v1.DeleteOptions{}, v1.ListOptions{
			LabelSelector: *input.LabelSelector,
		})
	}
	return nil
}
