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
	"errors"
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/config"
)

const (
	// tag is the same version as fastchat
	defaultFastChatImage     = "kubeagi/arcadia-fastchat-worker:v0.2.36"
	defaultFastchatVLLMImage = "kubeagi/arcadia-fastchat-worker:vllm-v0.2.36"
	// defaultKubeAGIImage for RunnerKubeAGI
	defaultKubeAGIImage = "kubeagi/core-library-cli:v0.0.1"
)

// ModelRunner run a model service
type ModelRunner interface {
	// Device used when running model
	Device() Device
	// NumberOfGPUs used when running model
	NumberOfGPUs() string
	// Build a model runner instance
	Build(ctx context.Context, model *arcadiav1alpha1.TypedObjectReference) (any, error)
}

var _ ModelRunner = (*RunnerFastchat)(nil)

// RunnerFastchat use fastchat to run a model
type RunnerFastchat struct {
	c client.Client
	w *arcadiav1alpha1.Worker

	modelFileFromRemote bool
}

func NewRunnerFastchat(c client.Client, w *arcadiav1alpha1.Worker, modelFileFromRemote bool) (ModelRunner, error) {
	return &RunnerFastchat{
		c:                   c,
		w:                   w,
		modelFileFromRemote: modelFileFromRemote,
	}, nil
}

// Device utilized by this runner
func (runner *RunnerFastchat) Device() Device {
	return DeviceBasedOnResource(runner.w.Spec.Resources.Limits)
}

// NumberOfGPUs utilized by this runner
func (runner *RunnerFastchat) NumberOfGPUs() string {
	return NumberOfGPUs(runner.w.Spec.Resources.Limits)
}

// Build a runner instance
func (runner *RunnerFastchat) Build(ctx context.Context, model *arcadiav1alpha1.TypedObjectReference) (any, error) {
	if model == nil {
		return nil, errors.New("nil model")
	}
	gw, err := config.GetGateway(ctx, runner.c)
	if err != nil {
		return nil, fmt.Errorf("failed to get arcadia config with %w", err)
	}

	modelFileDir := fmt.Sprintf("/data/models/%s", model.Name)
	additionalEnvs := []corev1.EnvVar{}
	extraArgs := fmt.Sprintf("--device %s", runner.Device().String())
	if runner.modelFileFromRemote {
		m := arcadiav1alpha1.Model{}
		if err := runner.c.Get(ctx, types.NamespacedName{Namespace: *model.Namespace, Name: model.Name}, &m); err != nil {
			return nil, err
		}
		if m.Spec.Revision != "" {
			extraArgs += fmt.Sprintf(" --revision %s ", m.Spec.Revision)
		}
		if m.Spec.ModelSource == modelSourceFromHugginfFace {
			modelFileDir = m.Spec.HuggingFaceRepo
		}
		if m.Spec.ModelSource == modelSourceFromModelScope {
			modelFileDir = m.Spec.ModelScopeRepo
			additionalEnvs = append(additionalEnvs, corev1.EnvVar{Name: "FASTCHAT_USE_MODELSCOPE", Value: "True"})
		}
	}

	additionalEnvs = append(additionalEnvs, corev1.EnvVar{Name: "FASTCHAT_MODEL_NAME_PATH", Value: modelFileDir})
	img := defaultFastChatImage
	if runner.w.Spec.Runner.Image != "" {
		img = runner.w.Spec.Runner.Image
	}
	// read worker address
	container := &corev1.Container{
		Name:            "runner",
		Image:           img,
		ImagePullPolicy: runner.w.Spec.Runner.ImagePullPolicy,
		Env: []corev1.EnvVar{
			{Name: "FASTCHAT_WORKER_NAME", Value: "fastchat.serve.model_worker"},
			{Name: "FASTCHAT_WORKER_NAMESPACE", Value: runner.w.Namespace},
			{Name: "FASTCHAT_REGISTRATION_MODEL_NAME", Value: runner.w.MakeRegistrationModelName()},
			{Name: "FASTCHAT_MODEL_NAME", Value: model.Name},
			{Name: "FASTCHAT_WORKER_ADDRESS", Value: fmt.Sprintf("http://%s.%s:%d", runner.w.Name+WokerCommonSuffix, runner.w.Namespace, arcadiav1alpha1.DefaultWorkerPort)},
			{Name: "FASTCHAT_CONTROLLER_ADDRESS", Value: gw.Controller},
			{Name: "NUMBER_GPUS", Value: runner.NumberOfGPUs()},
			{Name: "EXTRA_ARGS", Value: extraArgs},
		},
		Ports: []corev1.ContainerPort{
			{Name: "http", ContainerPort: arcadiav1alpha1.DefaultWorkerPort},
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "models", MountPath: "/data/models"},
		},
		Resources: runner.w.Spec.Resources,
	}

	container.Env = append(container.Env, additionalEnvs...)
	return container, nil
}

var _ ModelRunner = (*RunnerFastchatVLLM)(nil)

// RunnerFastchatVLLM use fastchat with vllm to run a model
type RunnerFastchatVLLM struct {
	c client.Client
	w *arcadiav1alpha1.Worker

	modelFileFromRemote bool
}

func NewRunnerFastchatVLLM(c client.Client, w *arcadiav1alpha1.Worker, modelFileFromRemote bool) (ModelRunner, error) {
	return &RunnerFastchatVLLM{
		c: c,
		w: w,

		modelFileFromRemote: modelFileFromRemote,
	}, nil
}

// Device used by this runner
func (runner *RunnerFastchatVLLM) Device() Device {
	return DeviceBasedOnResource(runner.w.Spec.Resources.Limits)
}

// NumberOfGPUs utilized by this runner
func (runner *RunnerFastchatVLLM) NumberOfGPUs() string {
	return NumberOfGPUs(runner.w.Spec.Resources.Limits)
}

// Build a runner instance
func (runner *RunnerFastchatVLLM) Build(ctx context.Context, model *arcadiav1alpha1.TypedObjectReference) (any, error) {
	if model == nil {
		return nil, errors.New("nil model")
	}
	gw, err := config.GetGateway(ctx, runner.c)
	if err != nil {
		return nil, fmt.Errorf("failed to get arcadia config with %w", err)
	}

	modelFileDir := fmt.Sprintf("/data/models/%s", model.Name)
	additionalEnvs := []corev1.EnvVar{}
	// --enforce-eager to disable cupy
	// TODO: remove --enforce-eager when https://github.com/kubeagi/arcadia/issues/878 is fixed
	extraAgrs := "--trust-remote-code --enforce-eager"
	if runner.modelFileFromRemote {
		m := arcadiav1alpha1.Model{}
		if err := runner.c.Get(ctx, types.NamespacedName{Namespace: *model.Namespace, Name: model.Name}, &m); err != nil {
			return nil, err
		}
		if m.Spec.Revision != "" {
			extraAgrs += fmt.Sprintf(" --revision %s", m.Spec.Revision)
		}
		if m.Spec.ModelSource == modelSourceFromHugginfFace {
			modelFileDir = m.Spec.HuggingFaceRepo
		}
		if m.Spec.ModelSource == modelSourceFromModelScope {
			modelFileDir = m.Spec.ModelScopeRepo
			additionalEnvs = append(additionalEnvs, corev1.EnvVar{Name: "FASTCHAT_USE_MODELSCOPE", Value: "True"})
		}
	}
	additionalEnvs = append(additionalEnvs, corev1.EnvVar{Name: "FASTCHAT_MODEL_NAME_PATH", Value: modelFileDir})

	// Get the real GPU requirement from env if configured
	// this will be the total GPU from ray resource pool, not the resource requests/limits
	gpuCount, _ := strconv.Atoi(runner.NumberOfGPUs())
	for _, envItem := range runner.w.Spec.AdditionalEnvs {
		if envItem.Name == "NUMBER_GPUS" {
			gpuCount, _ = strconv.Atoi(envItem.Value)
		}
		klog.Infof("run worker with %s GPU", runner.NumberOfGPUs())
		// Get the ray cluster address if configured
		if envItem.Name == "RAY_CLUSTER_INDEX" {
			rayClusterIndex, _ := strconv.Atoi(envItem.Value)
			rayClusters, err := config.GetRayClusters(ctx, runner.c)
			if err != nil || len(rayClusters) == 0 {
				klog.Warningln("no ray cluster configured, fallback to local resource: ", err)
			} else {
				// Use the 1st ray cluster for now
				rayClusterAddress := rayClusters[rayClusterIndex].HeadAddress
				pythonVersion := rayClusters[rayClusterIndex].GetPythonVersion()
				rayVersion := rayClusters[rayClusterIndex].GetRayVersion()
				klog.Infof("run worker using existing ray cluster: %s", rayClusterAddress)
				additionalEnvs = append(additionalEnvs,
					corev1.EnvVar{
						Name:  "RAY_ADDRESS",
						Value: rayClusterAddress,
					}, corev1.EnvVar{
						Name:  "RAY_VERSION",
						Value: rayVersion,
					}, corev1.EnvVar{
						Name:  "PYTHON_VERSION",
						Value: pythonVersion,
					})
			}
		}
	}

	img := defaultFastchatVLLMImage
	if runner.w.Spec.Runner.Image != "" {
		img = runner.w.Spec.Runner.Image
	}
	container := &corev1.Container{
		Name:            "runner",
		Image:           img,
		ImagePullPolicy: runner.w.Spec.Runner.ImagePullPolicy,
		Env: []corev1.EnvVar{
			{Name: "FASTCHAT_WORKER_NAME", Value: "fastchat.serve.vllm_worker"},
			{Name: "FASTCHAT_WORKER_NAMESPACE", Value: runner.w.Namespace},
			{Name: "FASTCHAT_REGISTRATION_MODEL_NAME", Value: runner.w.MakeRegistrationModelName()},
			{Name: "FASTCHAT_MODEL_NAME", Value: model.Name},
			{Name: "FASTCHAT_WORKER_ADDRESS", Value: fmt.Sprintf("http://%s.%s:%d", runner.w.Name+WokerCommonSuffix, runner.w.Namespace, arcadiav1alpha1.DefaultWorkerPort)},
			{Name: "FASTCHAT_CONTROLLER_ADDRESS", Value: gw.Controller},
			{Name: "EXTRA_ARGS", Value: extraAgrs},
			{Name: "NUMBER_GPUS", Value: strconv.Itoa(gpuCount)},
		},
		Ports: []corev1.ContainerPort{
			{Name: "http", ContainerPort: arcadiav1alpha1.DefaultWorkerPort},
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "models", MountPath: "/data/models"},
		},
		Resources: runner.w.Spec.Resources,
	}
	container.Env = append(container.Env, additionalEnvs...)
	return container, nil
}

var _ ModelRunner = (*KubeAGIRunner)(nil)

// KubeAGIRunner utilizes  core-library-cli(https://github.com/kubeagi/core-library/tree/main/libs/cli) to run model services
// Mainly for reranking,whisper,etc..
type KubeAGIRunner struct {
	c client.Client
	w *arcadiav1alpha1.Worker

	modelFileFromRemote bool
}

func NewKubeAGIRunner(c client.Client, w *arcadiav1alpha1.Worker, modelFileFromRemote bool) (ModelRunner, error) {
	return &KubeAGIRunner{
		c: c,
		w: w,

		modelFileFromRemote: modelFileFromRemote,
	}, nil
}

// Device used when running model
func (runner *KubeAGIRunner) Device() Device {
	return DeviceBasedOnResource(runner.w.Spec.Resources.Limits)
}

// NumberOfGPUs utilized by this runner
func (runner *KubeAGIRunner) NumberOfGPUs() string {
	return NumberOfGPUs(runner.w.Spec.Resources.Limits)
}

// Build a model runner instance
func (runner *KubeAGIRunner) Build(ctx context.Context, model *arcadiav1alpha1.TypedObjectReference) (any, error) {
	if model == nil {
		return nil, errors.New("nil model")
	}

	img := defaultKubeAGIImage
	if runner.w.Spec.Runner.Image != "" {
		img = runner.w.Spec.Runner.Image
	}

	// read worker address
	mountPath := "/data/models"
	rerankModelPath := fmt.Sprintf("%s/%s", mountPath, model.Name)

	if runner.modelFileFromRemote {
		m := arcadiav1alpha1.Model{}
		if err := runner.c.Get(ctx, types.NamespacedName{Namespace: *model.Namespace, Name: model.Name}, &m); err != nil {
			return nil, err
		}
		if m.Spec.HuggingFaceRepo != "" {
			rerankModelPath = m.Spec.HuggingFaceRepo
		}
		/*
			TODO support modelscope
			if m.Spec.ModelScopeRepo != "" {
			    rerankModelPath = m.Spec.ModelScopeRepo
			}
		*/
	}
	container := &corev1.Container{
		Name:            "runner",
		Image:           img,
		ImagePullPolicy: runner.w.Spec.Runner.ImagePullPolicy,
		Command: []string{
			"python", "kubeagi_cli/cli.py", "serve", "--host", "0.0.0.0", "--port", fmt.Sprintf("%d", arcadiav1alpha1.DefaultWorkerPort),
		},
		Env: []corev1.EnvVar{
			// Only reranking supported for now
			{Name: "RERANKING_MODEL_PATH", Value: rerankModelPath},
		},
		Ports: []corev1.ContainerPort{
			{Name: "http", ContainerPort: arcadiav1alpha1.DefaultWorkerPort},
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "models", MountPath: mountPath},
		},
		Resources: runner.w.Spec.Resources,
	}

	return container, nil
}
