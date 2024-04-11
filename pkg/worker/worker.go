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
	"reflect"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
)

const (
	WokerCommonSuffix = "-worker"

	RDMANodeLabel = "arcadia.kubeagi.k8s.com.cn/rdma"

	modelSourceFromLocal       = "local"
	modelSourceFromHugginfFace = "huggingface"
	modelSourceFromModelScope  = "modelscope"
)

var (
	ErrNotImplementedYet = errors.New("not implemented yet")
	ErrModelNotReady     = errors.New("worker's model is not ready")

	// Default replicas for a worker
	// Only support 1 for now
	DefaultWorkerReplicas int32 = 1
)

type Action string

const (
	Create Action = "create"
	Update Action = "update"
	Panic  Action = "panic"
)

func ActionOnError(err error) Action {
	if err == nil {
		return Update
	} else if !k8serrors.IsNotFound(err) {
		return Panic
	}
	return Create
}

// Worker implement the lifecycle management of a LLM worker
type Worker interface {
	// Worker that this is for
	Worker() *arcadiav1alpha1.Worker
	// Model that this worker is running for
	Model() *arcadiav1alpha1.Model

	// Actions to do before start this worker
	BeforeStart(ctx context.Context) error
	// Actions to do when Start this worker
	Start(ctx context.Context) error
	// Actions to do after start this worker
	AfterStart(ctx context.Context) error

	// Actions to do before stop this worker
	BeforeStop(ctx context.Context) error
	// Actions to do when Stop this worker
	Stop(ctx context.Context) error

	// State of this worker
	State(context.Context) (any, error)
}

var _ Worker = (*PodWorker)(nil)

// PodWorker hosts this worker in a single pod but with different loader and runner based on Worker's configuration
type PodWorker struct {
	c client.Client
	s *runtime.Scheme

	// worker's namespacedname
	types.NamespacedName
	// worker instance
	w *arcadiav1alpha1.Worker
	// model this worker is for
	m *arcadiav1alpha1.Model

	// ModelLoader provides a way to load this model
	l ModelLoader
	// ModelRunner provides a way to run this model
	r ModelRunner

	// fields to start a worker
	service    corev1.Service
	deployment appsv1.Deployment
	storage    corev1.Volume
}

func (podWorker *PodWorker) SuffixedName() string {
	return podWorker.Name + WokerCommonSuffix
}

func NewPodWorker(ctx context.Context, c client.Client, s *runtime.Scheme, w *arcadiav1alpha1.Worker, d *arcadiav1alpha1.Datasource) (*PodWorker, error) {
	model := w.Spec.Model.DeepCopy()
	if model.Namespace == nil {
		model.Namespace = &w.Namespace
	}

	podWorker := &PodWorker{
		c: c,
		s: s,
		w: w.DeepCopy(),
		NamespacedName: types.NamespacedName{
			Namespace: w.Namespace,
			Name:      w.Name,
		},
	}

	// check model
	m := &arcadiav1alpha1.Model{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: *model.Namespace, Name: model.Name}, m); err != nil {
		return nil, err
	}
	if !m.Status.IsReady() {
		klog.Errorf("%s/%s model is not ready", m.Namespace, m.Name)
		return nil, ErrModelNotReady
	}
	podWorker.m = m

	// default fields in a worker
	storage := corev1.Volume{
		Name: "models",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	if d.Spec.Type() == arcadiav1alpha1.DatasourceTypeRDMA {
		storage = corev1.Volume{
			Name: "models",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					// /rdma/abc/uid -> /data/models
					// The path has been restricted to contain a suffix `/`
					Path: fmt.Sprintf("%s%s", d.Spec.RDMA.Path, w.GetUID()),
				},
			},
		}
	}

	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podWorker.SuffixedName(),
			Namespace: podWorker.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				arcadiav1alpha1.WorkerPodSelectorLabel: podWorker.SuffixedName(),
			},
			Ports: []corev1.ServicePort{
				{Name: "http", Port: arcadiav1alpha1.DefaultWorkerPort, TargetPort: intstr.Parse("http"), Protocol: corev1.ProtocolTCP},
			},
		},
	}

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podWorker.SuffixedName(),
			Namespace: w.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					arcadiav1alpha1.WorkerPodSelectorLabel: podWorker.SuffixedName(),
				},
			},
			Strategy: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
			Replicas: &DefaultWorkerReplicas,
		},
	}

	// set the worker replicas
	if w.Spec.Replicas != nil {
		deployment.Spec.Replicas = w.Spec.Replicas
	}

	podWorker.storage = storage
	podWorker.service = service
	podWorker.deployment = deployment

	switch d.Spec.Type() {
	case arcadiav1alpha1.DatasourceTypeOSS:
		// init loader(Only oss supported yet)
		endpoint := d.Spec.Endpoint.DeepCopy()
		if endpoint.AuthSecret != nil && endpoint.AuthSecret.Namespace == nil {
			endpoint.AuthSecret.WithNameSpace(d.Namespace)
		}
		l, err := NewLoaderOSS(ctx, c, endpoint, w)
		if err != nil {
			return nil, fmt.Errorf("failed to new a loader with %w", err)
		}
		podWorker.l = l
	case arcadiav1alpha1.DatasourceTypeRDMA:
		l := NewRDMALoader(c, w.Spec.Model.Name, string(w.GetUID()), d, w)
		podWorker.l = l
	default:
		return nil, fmt.Errorf("datasource %s with type %s not supported in worker", d.Name, d.Spec.Type())
	}

	return podWorker, nil
}

func (podWorker *PodWorker) Worker() *arcadiav1alpha1.Worker {
	return podWorker.w
}

// Model that this worker is running for
func (podWorker *PodWorker) Model() *arcadiav1alpha1.Model {
	return podWorker.m.DeepCopy()
}

// BeforeStart will create resources which are related to this Worker
// Now we have a pvc(if configured), service, LLM(if a llm model), Embedder(if a embedding model)
func (podWorker *PodWorker) BeforeStart(ctx context.Context) error {
	var err error

	// Capability Checks
	if podWorker.Model().IsRerankingModel() && podWorker.Worker().Type() != arcadiav1alpha1.WorkerTypeKubeAGI {
		return errors.New("only kubeagi runner can host reranking models")
	}

	// If the local directory is mounted, there is no need to create the pvc
	if podWorker.Worker().Spec.Storage != nil && podWorker.storage.HostPath == nil {
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: podWorker.Namespace,
				Name:      podWorker.SuffixedName(),
			},
			Spec: *podWorker.Worker().Spec.Storage.DeepCopy(),
		}
		err = controllerutil.SetControllerReference(podWorker.Worker(), pvc, podWorker.s)
		if err != nil {
			return fmt.Errorf("failed to set owner reference with %w", err)
		}

		err = podWorker.c.Get(ctx, types.NamespacedName{Namespace: pvc.Namespace, Name: pvc.Name}, &corev1.PersistentVolumeClaim{})
		switch ActionOnError(err) {
		case Panic:
			return err
		case Update:
			// DO NOT UPDATE PVC which is easily caused a conflict
			// if err = podWorker.c.Update(ctx, pvc); err != nil {
			// 	return err
			// }
		case Create:
			err = podWorker.c.Create(ctx, pvc)
			if err != nil {
				return err
			}
		}
		podWorker.storage = corev1.Volume{
			Name: "models",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: podWorker.SuffixedName(),
				},
			},
		}
	}

	// prepare svc
	svc := podWorker.service.DeepCopy()
	err = controllerutil.SetControllerReference(podWorker.Worker(), svc, podWorker.s)
	if err != nil {
		return err
	}
	err = podWorker.c.Get(ctx, types.NamespacedName{Namespace: svc.Namespace, Name: svc.Name}, &corev1.Service{})
	switch ActionOnError(err) {
	case Panic:
		return err
	case Update:
		if err := podWorker.c.Update(ctx, svc); err != nil {
			return err
		}
	case Create:
		if err := podWorker.c.Create(ctx, svc); err != nil {
			return err
		}
	}

	// prepare LLM/Embedder
	model := podWorker.Model()

	// If worker is utilizing vllm, then we should not create embedder for it
	// vllm doesn't support embedding apis.See https://github.com/vllm-project/vllm/issues/183
	if model.IsEmbeddingModel() && podWorker.Worker().Type() != arcadiav1alpha1.WorkerTypeFastchatVLLM {
		embedder := &arcadiav1alpha1.Embedder{}
		err := podWorker.c.Get(ctx, types.NamespacedName{Namespace: podWorker.Namespace, Name: podWorker.Name}, embedder)
		switch ActionOnError(err) {
		case Create:
			// Create when not found
			embedder = podWorker.Worker().BuildEmbedder()
			if err = controllerutil.SetControllerReference(podWorker.Worker(), embedder, podWorker.c.Scheme()); err != nil {
				return err
			}
			if err = podWorker.c.Create(ctx, embedder); err != nil {
				// Ignore error when already exists
				if !k8serrors.IsAlreadyExists(err) {
					return err
				}
			}
		case Update:
			embedder1 := podWorker.Worker().BuildEmbedder()
			if !reflect.DeepEqual(embedder.Spec, embedder1.Spec) {
				embedder.Spec = embedder1.Spec
				return podWorker.c.Update(ctx, embedder)
			}
		case Panic:
			return err
		}
	}

	if model.IsLLMModel() {
		llm := &arcadiav1alpha1.LLM{}
		err := podWorker.c.Get(ctx, types.NamespacedName{Namespace: podWorker.Namespace, Name: podWorker.Name}, llm)
		switch ActionOnError(err) {
		case Create:
			// Create when not found
			llm = podWorker.Worker().BuildLLM()
			if err = controllerutil.SetControllerReference(podWorker.Worker(), llm, podWorker.c.Scheme()); err != nil {
				return err
			}
			if err = podWorker.c.Create(ctx, llm); err != nil {
				// Ignore error when already exists
				if !k8serrors.IsAlreadyExists(err) {
					return err
				}
			}
		case Update:
			llm1 := podWorker.Worker().BuildLLM()
			if !reflect.DeepEqual(llm.Spec, llm1.Spec) {
				llm.Spec = llm1.Spec
				return podWorker.c.Update(ctx, llm)
			}
		case Panic:
			return err
		}
	}

	return nil
}

// Start will build and create worker pod which will host model service
func (podWorker *PodWorker) Start(ctx context.Context) error {
	var (
		err    error
		loader any
	)

	// define the way to load model
	if podWorker.m.Spec.ModelSource == "" || podWorker.m.Spec.ModelSource == modelSourceFromLocal {
		loader, err = podWorker.l.Build(ctx, &arcadiav1alpha1.TypedObjectReference{Namespace: &podWorker.m.Namespace, Name: podWorker.m.Name})
		if err != nil {
			return fmt.Errorf("failed to build loader with %w", err)
		}
	}

	switch podWorker.w.Type() {
	case arcadiav1alpha1.WorkerTypeFastchatVLLM:
		r, err := NewRunnerFastchatVLLM(podWorker.c, podWorker.w.DeepCopy(), loader == nil)
		if err != nil {
			return fmt.Errorf("failed to new a runner with %w", err)
		}
		podWorker.r = r
	case arcadiav1alpha1.WorkerTypeFastchatNormal:
		r, err := NewRunnerFastchat(podWorker.c, podWorker.w.DeepCopy(), loader == nil)
		if err != nil {
			return fmt.Errorf("failed to new a runner with %w", err)
		}
		podWorker.r = r
	case arcadiav1alpha1.WorkerTypeKubeAGI:
		r, err := NewKubeAGIRunner(podWorker.c, podWorker.w.DeepCopy(), loader == nil)
		if err != nil {
			return fmt.Errorf("failed to new a runner with %w", err)
		}
		podWorker.r = r
	default:
		return fmt.Errorf("worker %s with type %s not supported in worker", podWorker.w.Name, podWorker.w.Type())
	}

	// define the way to run model
	runner, err := podWorker.r.Build(ctx, &arcadiav1alpha1.TypedObjectReference{Namespace: &podWorker.m.Namespace, Name: podWorker.m.Name})
	if err != nil {
		return fmt.Errorf("failed to build runner with %w", err)
	}
	conRunner, _ := runner.(*corev1.Container)

	// TODO: better way to handle local model file, maybe user want to use it as local cache
	if podWorker.storage.HostPath != nil {
		conRunner.Lifecycle = &corev1.Lifecycle{
			PreStop: &corev1.LifecycleHandler{
				Exec: &corev1.ExecAction{
					Command: []string{"/bin/bash", "-c", fmt.Sprintf("rm -rf /data/models/%s", podWorker.Model().Name)},
				},
			},
		}
	}

	// initialize deployment
	desiredDep := podWorker.deployment.DeepCopy()
	// configure pod template
	podSpecTemplate := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				arcadiav1alpha1.WorkerPodSelectorLabel: podWorker.SuffixedName(),
				arcadiav1alpha1.WorkerPodLabel:         podWorker.Worker().Name,
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyAlways,
			Containers:    []corev1.Container{*conRunner},
			Volumes:       []corev1.Volume{podWorker.storage},
		},
	}
	if loader != nil {
		conLoader, _ := loader.(*corev1.Container)
		podSpecTemplate.Spec.InitContainers = []corev1.Container{*conLoader}
	}
	if podWorker.storage.HostPath != nil {
		podSpecTemplate.Spec.Affinity = &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Operator: corev1.NodeSelectorOpExists,
									Key:      RDMANodeLabel,
								},
							},
						},
					},
				},
			},
		}
		podSpecTemplate.Spec.Volumes = append(podSpecTemplate.Spec.Volumes, corev1.Volume{
			Name: "tmp",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/tmp",
				},
			},
		})
	}

	// Configure node affinity
	if podWorker.w.Spec.MatchExpressions != nil {
		if podSpecTemplate.Spec.Affinity != nil {
			// Append to the current node selector
			podSpecTemplate.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions =
				append(podSpecTemplate.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions, podWorker.w.Spec.MatchExpressions...)
		} else {
			// Create a new one
			podSpecTemplate.Spec.Affinity = &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: podWorker.w.Spec.MatchExpressions,
							},
						},
					},
				},
			}
		}
	}

	// Configure environment variable
	// download from huggingface need HF-TOKEN env
	if len(podSpecTemplate.Spec.Containers) > 0 {
		podSpecTemplate.Spec.Containers[0].Env = append(podSpecTemplate.Spec.Containers[0].Env, podWorker.w.Spec.AdditionalEnvs...)
	}

	desiredDep.Spec.Template = podSpecTemplate
	err = controllerutil.SetControllerReference(podWorker.Worker(), desiredDep, podWorker.s)
	if err != nil {
		return fmt.Errorf("failed to set owner reference with %w", err)
	}

	currDep := &appsv1.Deployment{}
	err = podWorker.c.Get(ctx, types.NamespacedName{Namespace: desiredDep.Namespace, Name: desiredDep.Name}, currDep)
	switch ActionOnError(err) {
	case Panic:
		return err
	case Update:
		merged := MakeMergedDeployment(currDep, desiredDep)
		// Update only when spec changed
		err = podWorker.c.Patch(ctx, merged, client.MergeFrom(currDep))
		if err != nil {
			return errors.Wrap(err, "Failed to update worker")
		}

	case Create:
		err = podWorker.c.Create(ctx, desiredDep)
		if err != nil {
			return fmt.Errorf("failed to create deployment with %w", err)
		}
	}

	return nil
}

func MakeMergedDeployment(target *appsv1.Deployment, desired *appsv1.Deployment) *appsv1.Deployment {
	merged := target.DeepCopy()

	// merge this deployment with desired
	merged.Spec = desired.Spec

	return merged
}

// Actions to do after start this worker
func (podWorker *PodWorker) AfterStart(ctx context.Context) error {
	// get worker's latest state
	status, err := podWorker.State(ctx)
	if err != nil {
		return errors.Wrap(err, "Failed to do State")
	}

	// check & patch state
	podStatus := status.(*corev1.PodStatus)
	switch podStatus.Phase {
	case corev1.PodRunning, corev1.PodSucceeded:
		var condition arcadiav1alpha1.Condition
		for _, container := range podStatus.ContainerStatuses {
			// When pod phase is running or succeeded but container state is waiting,we use ErrorCondition
			if container.State.Waiting != nil || container.State.Terminated != nil {
				msg := ""
				if container.State.Waiting != nil {
					msg = fmt.Sprintf("Reason:%s Message:%s", container.State.Waiting.Reason, container.State.Waiting.Message)
				} else if container.State.Terminated != nil {
					msg = fmt.Sprintf("Reason:%s Message:%s", container.State.Terminated.Reason, container.State.Terminated.Message)
				}

				condition = podWorker.Worker().ErrorCondition(msg)
				break
			}
			if container.State.Running != nil {
				condition = podWorker.Worker().ReadyCondition()
			}
		}
		podWorker.Worker().Status.SetConditions(condition)
	case corev1.PodPending:
		podWorker.Worker().Status.SetConditions(podWorker.Worker().PendingCondition())
	case corev1.PodUnknown:
		// If pod is unknown and replicas is zero,then this must be offline
		if *podWorker.w.Spec.Replicas == 0 {
			podWorker.Worker().Status.SetConditions(podWorker.Worker().OfflineCondition())
		} else {
			podWorker.Worker().Status.SetConditions(podWorker.Worker().PendingCondition())
		}
	case corev1.PodFailed:
		podWorker.Worker().Status.SetConditions(podWorker.Worker().ErrorCondition("Pod failed"))
	}

	podWorker.Worker().Status.PodStatus = *podStatus

	return nil
}

// TODO: BeforeStop
func (podWorker *PodWorker) BeforeStop(ctx context.Context) error {
	return nil
}

// TODO: Stop
func (podWorker *PodWorker) Stop(ctx context.Context) error {
	return nil
}

// State of this worker
func (podWorker *PodWorker) State(ctx context.Context) (any, error) {
	podList := &corev1.PodList{}
	err := podWorker.c.List(ctx, podList, &client.ListOptions{
		LabelSelector: labels.Set{
			arcadiav1alpha1.WorkerPodSelectorLabel: podWorker.SuffixedName(),
		}.AsSelector(),
	})

	if err != nil {
		return nil, err
	}

	if len(podList.Items) != 1 {
		return &corev1.PodStatus{
			Phase:   corev1.PodUnknown,
			Message: fmt.Sprintf("Expected one pod but got %d", len(podList.Items)),
		}, nil
	}

	return &podList.Items[0].Status, nil
}
