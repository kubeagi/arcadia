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
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/config"
)

// ModelRunner run a model service
type ModelRunner interface {
	// Device used when running model
	Device() Device
	// Build a model runner instance
	Build(ctx context.Context, model *arcadiav1alpha1.TypedObjectReference) (any, error)
}

var _ ModelRunner = (*RunnerFastchat)(nil)

// RunnerFastchat use fastchat to run a model
type RunnerFastchat struct {
	c client.Client
	w *arcadiav1alpha1.Worker
}

func NewRunnerFastchat(c client.Client, w *arcadiav1alpha1.Worker) (ModelRunner, error) {
	return &RunnerFastchat{
		c: c,
		w: w,
	}, nil
}

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
	gw, err := config.GetGateway(ctx, runner.c, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get arcadia config with %w", err)
	}

	// read worker address
	container := &corev1.Container{
		Name:            "runner",
		Image:           "kubeagi/arcadia-fastchat-worker:v0.2.0",
		ImagePullPolicy: "IfNotPresent",
		Env: []corev1.EnvVar{
			{Name: "FASTCHAT_WORKER_NAME", Value: "fastchat.serve.model_worker"},
			{Name: "FASTCHAT_WORKER_NAMESPACE", Value: runner.w.Namespace},
			{Name: "FASTCHAT_REGISTRATION_MODEL_NAME", Value: runner.w.MakeRegistrationModelName()},
			{Name: "FASTCHAT_MODEL_NAME", Value: model.Name},
			{Name: "FASTCHAT_WORKER_ADDRESS", Value: fmt.Sprintf("http://%s.%s:21002", runner.w.Name+WokerCommonSuffix, runner.w.Namespace)},
			{Name: "FASTCHAT_CONTROLLER_ADDRESS", Value: gw.Controller},
			{Name: "NUMBER_GPUS", Value: runner.NumberOfGPUs()},
			{Name: "EXTRA_ARGS", Value: fmt.Sprintf("--device %s", runner.Device().String())},
		},
		Ports: []corev1.ContainerPort{
			{Name: "http", ContainerPort: 21002},
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "models", MountPath: "/data/models"},
		},
		Resources: runner.w.Spec.Resources,
	}

	return container, nil
}

var _ ModelRunner = (*RunnerFastchatVLLM)(nil)

// RunnerFastchatVLLM use fastchat with vllm to run a model
type RunnerFastchatVLLM struct {
	c client.Client
	w *arcadiav1alpha1.Worker
}

func NewRunnerFastchatVLLM(c client.Client, w *arcadiav1alpha1.Worker) (ModelRunner, error) {
	return &RunnerFastchatVLLM{
		c: c,
		w: w,
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
	gw, err := config.GetGateway(ctx, runner.c, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get arcadia config with %w", err)
	}

	rayClusterAddress := ""
	pythonVersion := ""

	// Get the real GPU requirement from env if configured
	// this will be the total GPU from ray resource pool, not the resource requests/limits
	gpuCount, _ := strconv.Atoi(runner.NumberOfGPUs())
	rayClusterIndex := 0
	for _, envItem := range runner.w.Spec.AdditionalEnvs {
		if envItem.Name == "NUMBER_GPUS" {
			gpuCount, _ = strconv.Atoi(envItem.Value)
		}
		if envItem.Name == "RAY_CLUSTER_INDEX" {
			rayClusterIndex, _ = strconv.Atoi(envItem.Value)
		}
	}

	// Get ray config from configMap
	if gpuCount > 1 {
		rayClusters, err := config.GetRayClusters(ctx, runner.c, nil)
		if err != nil || len(rayClusters) == 0 {
			klog.Warningln("no ray cluster configured, fallback to local resource: ", err)
		} else {
			// Use the 1st ray cluster for now
			// TODO: let user to select with ray cluster to use
			rayClusterAddress = rayClusters[rayClusterIndex].HeadAddress
			pythonVersion = rayClusters[rayClusterIndex].PythonVersion
			klog.Infof("run worker using ray: %s, number of GPU: %s", rayClusterAddress, runner.NumberOfGPUs())
		}
	} else {
		klog.Infof("run worker with %s GPU", runner.NumberOfGPUs())
	}

	container := &corev1.Container{
		Name:            "runner",
		Image:           "kubeagi/arcadia-fastchat-worker:vllm-v0.2.0",
		ImagePullPolicy: "IfNotPresent",
		Env: []corev1.EnvVar{
			{Name: "FASTCHAT_WORKER_NAME", Value: "fastchat.serve.vllm_worker"},
			{Name: "FASTCHAT_WORKER_NAMESPACE", Value: runner.w.Namespace},
			{Name: "FASTCHAT_REGISTRATION_MODEL_NAME", Value: runner.w.MakeRegistrationModelName()},
			{Name: "FASTCHAT_MODEL_NAME", Value: model.Name},
			{Name: "FASTCHAT_WORKER_ADDRESS", Value: fmt.Sprintf("http://%s.%s:21002", runner.w.Name+WokerCommonSuffix, runner.w.Namespace)},
			{Name: "FASTCHAT_CONTROLLER_ADDRESS", Value: gw.Controller},
			{Name: "EXTRA_ARGS", Value: "--trust-remote-code"},
			// Need python version and ray address for distributed inference
			{Name: "PYTHON_VERSION", Value: pythonVersion},
			{Name: "RAY_ADDRESS", Value: rayClusterAddress},
		},
		Ports: []corev1.ContainerPort{
			{Name: "http", ContainerPort: 21002},
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "models", MountPath: "/data/models"},
		},
		Resources: runner.w.Spec.Resources,
	}

	return container, nil
}
