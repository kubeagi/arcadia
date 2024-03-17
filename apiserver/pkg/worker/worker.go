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
	"strconv"
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/config"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	gqlmodel "github.com/kubeagi/arcadia/apiserver/pkg/model"
	graphqlutils "github.com/kubeagi/arcadia/apiserver/pkg/utils"
)

const (
	NvidiaGPU = "nvidia.com/gpu"
)

func worker2modelConverter(ctx context.Context, c client.Client, showModel bool) func(object client.Object) (generated.PageNode, error) {
	return func(object client.Object) (generated.PageNode, error) {
		u, ok := object.(*v1alpha1.Worker)
		if !ok {
			return nil, errors.New("can't convert object to Worker")
		}
		return Worker2model(ctx, c, u, showModel)
	}
}

// Worker2model convert unstructured `CR Worker` to graphql model
func Worker2model(ctx context.Context, c client.Client, worker *v1alpha1.Worker, showModel bool) (*generated.Worker, error) {
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
		replicas = fmt.Sprint(*worker.Spec.Replicas)
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

	// wrap Worker
	w := generated.Worker{
		ID:                &id,
		Name:              worker.Name,
		Namespace:         worker.Namespace,
		Creator:           &worker.Spec.Creator,
		Labels:            graphqlutils.MapStr2Any(worker.GetLabels()),
		Annotations:       graphqlutils.MapStr2Any(worker.GetAnnotations()),
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
		API:               new(string),
	}
	if r := worker.Labels[v1alpha1.WorkerModelTypesLabel]; r != "" {
		w.ModelTypes = strings.ReplaceAll(r, "_", ",")
	}

	// read worker's models
	if worker.Spec.Model != nil && showModel {
		typedModel := worker.Model()
		model, err := gqlmodel.ReadModel(ctx, c, typedModel.Name, *typedModel.Namespace)
		if err != nil {
			klog.V(1).ErrorS(err, "worker has no model defined", "worker", worker.Name)
		} else {
			w.ModelTypes = model.Types
		}
		w.Model = generated.TypedObjectReference{
			APIGroup:  typedModel.APIGroup,
			Kind:      typedModel.Kind,
			Name:      typedModel.Name,
			Namespace: typedModel.Namespace,
		}
	}

	return &w, nil
}

func CreateWorker(ctx context.Context, c client.Client, input generated.CreateWorkerInput) (*generated.Worker, error) {
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

	worker := &v1alpha1.Worker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
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

	err := c.Create(ctx, worker)
	if err != nil {
		return nil, err
	}

	api, _ := common.GetAPIServer(ctx, c, true)

	w, err := Worker2model(ctx, c, worker, true)
	if err != nil {
		return nil, err
	}
	*w.API = api
	return w, nil
}

func UpdateWorker(ctx context.Context, c client.Client, input *generated.UpdateWorkerInput) (*generated.Worker, error) {
	worker := &v1alpha1.Worker{}
	err := c.Get(ctx, types.NamespacedName{Namespace: input.Namespace, Name: input.Name}, worker)
	if err != nil {
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

	err = c.Update(ctx, worker)
	if err != nil {
		return nil, err
	}

	api, _ := common.GetAPIServer(ctx, c, true)

	w, err := Worker2model(ctx, c, worker, true)
	if err != nil {
		return nil, err
	}
	*w.API = api
	return w, nil
}

func DeleteWorkers(ctx context.Context, c client.Client, input *generated.DeleteCommonInput) (*string, error) {
	opts, err := common.DeleteAllOptions(input)
	if err != nil {
		return nil, err
	}
	err = c.DeleteAllOf(ctx, &v1alpha1.Worker{}, opts...)
	return nil, err
}

func ListWorkers(ctx context.Context, c client.Client, input generated.ListWorkerInput, showWorkerModel bool, listOpts ...common.ListOptionsFunc) (*generated.PaginatedResult, error) {
	opts := common.DefaultListOptions()
	for _, optFunc := range listOpts {
		optFunc(opts)
	}

	filter := make([]common.ResourceFilter, 0)
	page := pointer.IntDeref(input.Page, 1)
	pageSize := pointer.IntDeref(input.PageSize, -1)
	if input.Keyword != nil {
		filter = append(filter, common.FilterWorkerByKeyword(*input.Keyword))
	}
	if input.ModelTypes != nil {
		filter = append(filter, common.FilterWorkerByType(c, input.Namespace, *input.ModelTypes))
	}

	us := &v1alpha1.WorkerList{}
	options, err := common.NewListOptions(generated.ListCommonInput{
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
	err = c.List(ctx, us, options...)
	if err != nil {
		return nil, err
	}
	items := make([]client.Object, len(us.Items))
	for i := range us.Items {
		items[i] = &us.Items[i]
	}
	list, err := common.ListReources(items, page, pageSize, worker2modelConverter(ctx, c, showWorkerModel), filter...)
	if err != nil {
		return nil, err
	}
	api, _ := common.GetAPIServer(ctx, c, true)

	for i := range list.Nodes {
		tmp := list.Nodes[i].(*generated.Worker)
		*tmp.API = api
	}
	return list, nil
}

func ReadWorker(ctx context.Context, c client.Client, name, namespace string) (*generated.Worker, error) {
	u := &v1alpha1.Worker{}
	err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, u)
	if err != nil {
		return nil, err
	}
	api, _ := common.GetAPIServer(ctx, c, true)

	w, err := Worker2model(ctx, c, u, true)
	if err != nil {
		return nil, err
	}
	*w.API = api
	return w, nil
}
