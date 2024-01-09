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

package worker

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/config"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	gqlmodel "github.com/kubeagi/arcadia/apiserver/pkg/model"
	graphqlutils "github.com/kubeagi/arcadia/apiserver/pkg/utils"
	"github.com/kubeagi/arcadia/pkg/utils"
)

const (
	NvidiaGPU = "nvidia.com/gpu"
)

// Worker2model convert unstructured `CR Worker` to graphql model
func Worker2model(ctx context.Context, c dynamic.Interface, obj *unstructured.Unstructured) *generated.Worker {
	worker := &v1alpha1.Worker{}
	if err := utils.UnstructuredToStructured(obj, worker); err != nil {
		return &generated.Worker{}
	}

	id := string(worker.GetUID())

	creationtimestamp := worker.GetCreationTimestamp().Time

	// conditioned status
	condition := worker.Status.GetCondition(v1alpha1.TypeReady)
	updateTime := condition.LastTransitionTime.Time

	// Unknown,Pending ,Running ,Error
	status := common.GetObjStatus(worker)
	message := condition.Message

	// replicas
	var replicas string
	if worker.Spec.Replicas != nil {
		replicas = fmt.Sprint(worker.Spec.Replicas)
	}

	// resources
	cpu := worker.Spec.Resources.Limits[v1.ResourceCPU]
	cpuStr := cpu.String()
	memory := worker.Spec.Resources.Limits[v1.ResourceMemory]
	memoryStr := memory.String()
	nvidiaGPU := worker.Spec.Resources.Limits[NvidiaGPU]
	nvidiaGPUStr := nvidiaGPU.String()
	resources := generated.Resources{
		CPU:       &cpuStr,
		Memory:    &memoryStr,
		NvidiaGpu: &nvidiaGPUStr,
	}
	matchExpressions := make([]*generated.NodeSelectorRequirement, 0)
	if worker.Spec.MatchExpressions != nil {
		for _, nodeSelector := range worker.Spec.MatchExpressions {
			matchExpressions = append(matchExpressions, &generated.NodeSelectorRequirement{
				Key:      nodeSelector.Key,
				Operator: string(nodeSelector.Operator),
				Values:   nodeSelector.Values,
			})
		}
	}

	// additional envs
	additionalEnvs := make(map[string]interface{})
	if worker.Spec.AdditionalEnvs != nil {
		for _, env := range worker.Spec.AdditionalEnvs {
			additionalEnvs[env.Name] = env.Value
		}
	}

	workerType := string(worker.Type())

	api, _ := common.GetAPIServer(ctx, c, true)

	// wrap Worker
	w := generated.Worker{
		ID:                &id,
		Name:              worker.Name,
		Namespace:         worker.Namespace,
		Creator:           &worker.Spec.Creator,
		Labels:            graphqlutils.MapStr2Any(obj.GetLabels()),
		Annotations:       graphqlutils.MapStr2Any(obj.GetAnnotations()),
		DisplayName:       &worker.Spec.DisplayName,
		Description:       &worker.Spec.Description,
		Type:              &workerType,
		Status:            &status,
		Message:           &message,
		CreationTimestamp: &creationtimestamp,
		UpdateTimestamp:   &updateTime,
		Replicas:          &replicas,
		Resources:         resources,
		MatchExpressions:  matchExpressions,
		AdditionalEnvs:    additionalEnvs,
		ModelTypes:        "unknown",
		API:               &api,
	}

	// read worker's models
	if worker.Spec.Model != nil {
		typedModel := worker.Model()
		model, err := gqlmodel.ReadModel(ctx, c, typedModel.Name, *typedModel.Namespace)
		if err != nil {
			klog.V(1).ErrorS(err, "worker has no model defined", "worker")
		} else {
			w.ModelTypes = model.Types
		}
		w.Model = generated.TypedObjectReference{
			APIGroup:  &common.ArcadiaAPIGroup,
			Kind:      typedModel.Kind,
			Name:      typedModel.Name,
			Namespace: typedModel.Namespace,
		}
	}

	return &w
}

func CreateWorker(ctx context.Context, c dynamic.Interface, input generated.CreateWorkerInput) (*generated.Worker, error) {
	displayName, description := "", ""
	if input.DisplayName != nil {
		displayName = *input.DisplayName
	}
	if input.Description != nil {
		description = *input.Description
	}

	// set the model's namespace
	modelNs := input.Namespace
	if input.Model.Namespace != nil {
		modelNs = *input.Model.Namespace
		if modelNs != input.Namespace && modelNs != config.GetConfig().SystemNamespace {
			return nil, errors.Errorf("You are trying to use a model in another namespace %s which is not our system namespace: %s", modelNs, config.GetConfig().SystemNamespace)
		}
	}

	// Use `fastchat` as the default worker type
	workerType := v1alpha1.DefaultWorkerType()
	if input.Type != nil {
		workerType = v1alpha1.WorkerType(*input.Type)
	}

	// set node selectors
	var matchExpressions = make([]v1.NodeSelectorRequirement, 0)
	if input.MatchExpressions != nil {
		for _, nodeSelector := range input.MatchExpressions {
			matchExpressions = append(matchExpressions, v1.NodeSelectorRequirement{
				Key:      nodeSelector.Key,
				Operator: v1.NodeSelectorOperator(nodeSelector.Operator),
				Values:   nodeSelector.Values,
			})
		}
	}

	// set additional environment variables
	var additionalEnvs = make([]v1.EnvVar, 0)
	if input.AdditionalEnvs != nil {
		for k, v := range input.AdditionalEnvs {
			additionalEnvs = append(additionalEnvs, v1.EnvVar{
				Name:  k,
				Value: fmt.Sprint(v),
			})
		}
	}

	worker := v1alpha1.Worker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Worker",
			APIVersion: v1alpha1.GroupVersion.String(),
		},
		Spec: v1alpha1.WorkerSpec{
			CommonSpec: v1alpha1.CommonSpec{
				DisplayName: displayName,
				Description: description,
			},
			Type: workerType,
			Model: &v1alpha1.TypedObjectReference{
				Name:      input.Model.Name,
				Namespace: &modelNs,
				Kind:      "Model",
			},
			AdditionalEnvs:   additionalEnvs,
			MatchExpressions: matchExpressions,
		},
	}
	common.SetCreator(ctx, &worker.Spec.CommonSpec)

	// cpu & memory
	resources := v1.ResourceRequirements{
		Limits: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse(input.Resources.CPU),
			v1.ResourceMemory: resource.MustParse(input.Resources.Memory),
		},
	}
	// gpu (only nvidia gpu supported now)
	if input.Resources.NvidiaGpu != nil {
		resources.Limits[NvidiaGPU] = resource.MustParse(*input.Resources.NvidiaGpu)
	}
	worker.Spec.Resources = resources

	unstructuredWorker, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&worker)
	if err != nil {
		return nil, err
	}
	obj, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "worker")).
		Namespace(input.Namespace).Create(ctx, &unstructured.Unstructured{Object: unstructuredWorker}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return Worker2model(ctx, c, obj), nil
}

func UpdateWorker(ctx context.Context, c dynamic.Interface, input *generated.UpdateWorkerInput) (*generated.Worker, error) {
	obj, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "worker")).Namespace(input.Namespace).Get(ctx, input.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	worker := &v1alpha1.Worker{}
	if err := utils.UnstructuredToStructured(obj, worker); err != nil {
		return nil, err
	}

	worker.SetLabels(graphqlutils.MapAny2Str(input.Labels))
	worker.SetAnnotations(graphqlutils.MapAny2Str(input.Annotations))

	if input.DisplayName != nil {
		worker.Spec.DisplayName = *input.DisplayName
	}
	if input.Description != nil {
		worker.Spec.Description = *input.Description
	}

	// worker type
	if input.Type != nil {
		if worker.Type() != v1alpha1.WorkerType(*input.Type) {
			worker.Spec.Type = v1alpha1.WorkerType(*input.Type)
		}
	}

	// replicas
	if input.Replicas != nil {
		replicas, err := strconv.ParseInt(*input.Replicas, 10, 32)
		if err != nil {
			return nil, errors.Wrap(err, "Invalid replicas")
		}
		replicasInt32 := int32(replicas)
		worker.Spec.Replicas = &replicasInt32
	}

	// resources
	if input.Resources != nil {
		// cpu & memory
		resources := v1.ResourceRequirements{
			Limits: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse(input.Resources.CPU),
				v1.ResourceMemory: resource.MustParse(input.Resources.Memory),
			},
		}
		// gpu (only nvidia gpu supported now)
		if input.Resources.NvidiaGpu != nil {
			resources.Limits["nvidia.com/gpu"] = resource.MustParse(*input.Resources.NvidiaGpu)
		}

		worker.Spec.Resources = resources
	}

	// set node selectors
	if input.MatchExpressions != nil {
		var matchExpressions = make([]v1.NodeSelectorRequirement, 0)
		for _, nodeSelector := range input.MatchExpressions {
			matchExpressions = append(matchExpressions, v1.NodeSelectorRequirement{
				Key:      nodeSelector.Key,
				Operator: v1.NodeSelectorOperator(nodeSelector.Operator),
				Values:   nodeSelector.Values,
			})
		}
		worker.Spec.MatchExpressions = matchExpressions
	}

	// set additional environment variables
	if input.AdditionalEnvs != nil {
		var additionalEnvs = make([]v1.EnvVar, 0)
		for k, v := range input.AdditionalEnvs {
			additionalEnvs = append(additionalEnvs, v1.EnvVar{
				Name:  k,
				Value: fmt.Sprint(v),
			})
		}
		worker.Spec.AdditionalEnvs = additionalEnvs
	}

	unstructuredWorker, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&worker)
	if err != nil {
		return nil, err
	}

	updatedObject, err := common.ResouceUpdate(ctx, c, generated.TypedObjectReferenceInput{
		APIGroup:  &common.ArcadiaAPIGroup,
		Kind:      "Worker",
		Name:      input.Name,
		Namespace: &input.Namespace,
	}, unstructuredWorker, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	return Worker2model(ctx, c, updatedObject), nil
}

func DeleteWorkers(ctx context.Context, c dynamic.Interface, input *generated.DeleteCommonInput) (*string, error) {
	name := ""
	labelSelector, fieldSelector := "", ""
	if input.Name != nil {
		name = *input.Name
	}
	if input.FieldSelector != nil {
		fieldSelector = *input.FieldSelector
	}
	if input.LabelSelector != nil {
		labelSelector = *input.LabelSelector
	}
	resource := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "worker"))
	if name != "" {
		err := resource.Namespace(input.Namespace).Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil {
			return nil, err
		}
	} else {
		err := resource.Namespace(input.Namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
			LabelSelector: labelSelector,
			FieldSelector: fieldSelector,
		})
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func ListWorkers(ctx context.Context, c dynamic.Interface, input generated.ListWorkerInput, listOpts ...common.ListOptionsFunc) (*generated.PaginatedResult, error) {
	opts := common.DefaultListOptions()
	for _, optFunc := range listOpts {
		optFunc(opts)
	}

	keyword, modelTypes, labelSelector, fieldSelector := "", "", "", ""
	page, pageSize := 1, 10
	if input.Keyword != nil {
		keyword = *input.Keyword
	}
	if input.ModelTypes != nil {
		modelTypes = *input.ModelTypes
	}
	if input.FieldSelector != nil {
		fieldSelector = *input.FieldSelector
	}
	if input.LabelSelector != nil {
		labelSelector = *input.LabelSelector
	}
	if input.Page != nil && *input.Page > 0 {
		page = *input.Page
	}
	if input.PageSize != nil {
		pageSize = *input.PageSize
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	}
	us, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "worker")).Namespace(input.Namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}
	// sort by creation time
	sort.Slice(us.Items, func(i, j int) bool {
		return us.Items[i].GetCreationTimestamp().After(us.Items[j].GetCreationTimestamp().Time)
	})

	result := make([]generated.PageNode, 0, len(us.Items))
	for _, u := range us.Items {
		m := Worker2model(ctx, c, &u)
		// filter based on `keyword`
		if keyword != "" && !strings.Contains(m.Name, keyword) && !strings.Contains(*m.DisplayName, keyword) {
			continue
		}
		// filter based on `modelTypes`
		if modelTypes != "" && !strings.Contains(m.ModelTypes, modelTypes) {
			continue
		}
		result = append(result, opts.ConvertFunc(m))
	}
	totalCount := len(result)
	start, end := common.PagePosition(page, pageSize, totalCount)
	return &generated.PaginatedResult{
		TotalCount:  totalCount,
		HasNextPage: end < totalCount,
		Nodes:       result[start:end],
	}, nil
}

func ReadWorker(ctx context.Context, c dynamic.Interface, name, namespace string) (*generated.Worker, error) {
	u, err := common.ResouceGet(ctx, c, generated.TypedObjectReferenceInput{
		APIGroup:  &common.ArcadiaAPIGroup,
		Kind:      "Worker",
		Name:      name,
		Namespace: &namespace,
	}, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return Worker2model(ctx, c, u), nil
}
