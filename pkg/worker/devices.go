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
	corev1 "k8s.io/api/core/v1"
)

// Device defines different types like cpu,gpu,xpu,npu which runs the model
type Device string

const (
	CPU  Device = "cpu"
	CUDA Device = "cuda"
	// Not supported yet
	XPU Device = "xpu"
	// Not supported yet
	NPU Device = "npu"
)

func (device Device) String() string {
	return string(device)
}

const (
	// Resource
	ResourceNvidiaGPU corev1.ResourceName = "nvidia.com/gpu"
)

// DeviceBasedOnResource returns the device type based on the resource list
func DeviceBasedOnResource(resource corev1.ResourceList) Device {
	_, ok := resource[ResourceNvidiaGPU]
	if ok {
		return CUDA
	}
	return CPU
}

// NumberOfGPUs from ResourceList
func NumberOfGPUs(resource corev1.ResourceList) string {
	gpu, ok := resource[ResourceNvidiaGPU]
	if !ok {
		return "0"
	}
	return gpu.String()
}
