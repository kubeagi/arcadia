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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/v1alpha1"
)

const (
	WokerCommonSuffix = "-worker"
)

var (
	ErrNotImplementedYet = errors.New("not implemented yet")
	ErrModelNotReady     = errors.New("worker's model is not ready")
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
	// Model that this worker is running for
	Model() *arcadiav1alpha1.Model

	// Actions to do before start this worker
	BeforeStart(ctx context.Context) error
	// Actiosn to do when Start this worker
	Start(ctx context.Context) error

	// Actions to do before stop this worker
	BeforeStop(ctx context.Context) error
	// Actions to do when Stop this worker
	Stop(ctx context.Context) error

	// State of this worker
	State(context.Context) (any, error)
}

var _ Worker = (*PodWorker)(nil)

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

func (worker *PodWorker) SuffixedName() string {
	return worker.Name + WokerCommonSuffix
}

func NewPodWorker(ctx context.Context, c client.Client, s *runtime.Scheme, w *arcadiav1alpha1.Worker, d *arcadiav1alpha1.Datasource) (*PodWorker, error) {
	model := w.Spec.Model.DeepCopy()
	if model.Namespace == nil {
		model.Namespace = &w.Namespace
	}

	worker := &PodWorker{
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
		return nil, ErrModelNotReady
	}
	worker.m = m

	// default fields in a worker
	storage := corev1.Volume{
		Name: "models",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	service := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      worker.SuffixedName(),
			Namespace: worker.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app.kubernetes.io/name": worker.SuffixedName(),
			},
			Ports: []corev1.ServicePort{
				{Name: "http", Port: 21002, TargetPort: intstr.Parse("http"), Protocol: corev1.ProtocolTCP},
			},
		},
	}

	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      worker.SuffixedName(),
			Namespace: w.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": worker.SuffixedName(),
				},
			},
		},
	}

	worker.storage = storage
	worker.service = service
	worker.deployment = deployment

	// init loader(Only oss supported yet)
	endpoint := d.Spec.Enpoint.DeepCopy()
	if endpoint.AuthSecret != nil && endpoint.AuthSecret.Namespace == nil {
		endpoint.AuthSecret.WithNameSpace(d.Namespace)
	}
	switch d.Spec.Type() {
	case arcadiav1alpha1.DatasourceTypeOSS:
		l, err := NewLoaderOSS(ctx, c, endpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to new a loader with %w", err)
		}
		worker.l = l
	default:
		return nil, fmt.Errorf("datasource %s with type %s not supported in worker", d.Name, d.Spec.Type())
	}

	// init runner
	switch w.Spec.Type {
	case arcadiav1alpha1.WorkerTypeFastchatVLLM:
		r, err := NewRunnerFastchatVLLM(c, w.DeepCopy())
		if err != nil {
			return nil, fmt.Errorf("failed to new a runner with %w", err)
		}
		worker.r = r
	case arcadiav1alpha1.WorkerTypeFastchatNormal:
		r, err := NewRunnerFastchat(c, w.DeepCopy())
		if err != nil {
			return nil, fmt.Errorf("failed to new a runner with %w", err)
		}
		worker.r = r
	default:
		return nil, fmt.Errorf("worker %s with type %s not supported in worker", w.Name, w.Spec.Type)
	}

	return worker, nil
}

// Model that this worker is running for
func (worker *PodWorker) Model() *arcadiav1alpha1.Model {
	return worker.m.DeepCopy()
}

func (worker *PodWorker) BeforeStart(ctx context.Context) error {
	var err error

	// prepare pvc
	if worker.w.Spec.Storage != nil {
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: worker.Namespace,
				Name:      worker.SuffixedName(),
			},
			Spec: *worker.w.Spec.Storage.DeepCopy(),
		}
		err = controllerutil.SetControllerReference(worker.w, pvc, worker.s)
		if err != nil {
			return fmt.Errorf("failed to set owner reference with %w", err)
		}

		err = worker.c.Get(ctx, types.NamespacedName{Namespace: pvc.Namespace, Name: pvc.Name}, &corev1.PersistentVolumeClaim{})
		switch ActionOnError(err) {
		case Panic:
			return err
		case Update:
			if err = worker.c.Update(ctx, pvc); err != nil {
				return err
			}
		case Create:
			err = worker.c.Create(ctx, pvc)
			if err != nil {
				return err
			}
		}
		worker.storage = corev1.Volume{
			Name: "models",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: worker.SuffixedName(),
				},
			},
		}
	}

	// prepare svc
	svc := worker.service.DeepCopy()
	err = controllerutil.SetControllerReference(worker.w, svc, worker.s)
	if err != nil {
		return err
	}
	err = worker.c.Get(ctx, types.NamespacedName{Namespace: svc.Namespace, Name: svc.Name}, &corev1.Service{})
	switch ActionOnError(err) {
	case Panic:
		return err
	case Update:
		if err := worker.c.Update(ctx, svc); err != nil {
			return err
		}
	case Create:
		if err := worker.c.Create(ctx, svc); err != nil {
			return err
		}
	}

	return nil
}

func (worker *PodWorker) Start(ctx context.Context) error {
	var err error

	// define the way to load model
	loader, err := worker.l.Build(ctx, &arcadiav1alpha1.TypedObjectReference{Namespace: &worker.m.Namespace, Name: worker.m.Name})
	if err != nil {
		return fmt.Errorf("failed to build loader with %w", err)
	}
	conLoader, _ := loader.(*corev1.Container)

	// define the way to run model
	runner, err := worker.r.Build(ctx, &arcadiav1alpha1.TypedObjectReference{Namespace: &worker.m.Namespace, Name: worker.m.Name})
	if err != nil {
		return fmt.Errorf("failed to build runner with %w", err)
	}
	conRunner, _ := runner.(*corev1.Container)

	// initialize deployment
	deployment := worker.deployment.DeepCopy()
	deployment.Spec.Template = corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"app.kubernetes.io/name": worker.SuffixedName(),
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy:  corev1.RestartPolicyAlways,
			InitContainers: []corev1.Container{*conLoader},
			Containers:     []corev1.Container{*conRunner},
			Volumes:        []corev1.Volume{worker.storage},
		},
	}
	err = controllerutil.SetControllerReference(worker.w, deployment, worker.s)
	if err != nil {
		return fmt.Errorf("failed to set owner reference with %w", err)
	}

	currDeployment := &appsv1.Deployment{}
	err = worker.c.Get(ctx, types.NamespacedName{Namespace: deployment.Namespace, Name: deployment.Name}, currDeployment)
	switch ActionOnError(err) {
	case Panic:
		return err
	case Update:
		// TODO: check to decide whethere
		// err = worker.c.Update(ctx, deployment)
		// if err != nil {
		// 	return fmt.Errorf("failed to create deployment with %w", err)
		// }
	case Create:
		err = worker.c.Create(ctx, deployment)
		if err != nil {
			return fmt.Errorf("failed to create deployment with %w", err)
		}
	}

	return nil
}

// TODO: BeforeStop
func (worker *PodWorker) BeforeStop(ctx context.Context) error {
	return nil
}

// TODO: Stop
func (worker *PodWorker) Stop(ctx context.Context) error {
	return nil
}

// State of this worker
func (worker *PodWorker) State(ctx context.Context) (any, error) {
	podList := &corev1.PodList{}
	err := worker.c.List(ctx, podList, &client.ListOptions{
		LabelSelector: labels.Set{
			"app.kubernetes.io/name": worker.SuffixedName(),
		}.AsSelector(),
	})

	if err != nil {
		return nil, err
	}

	if len(podList.Items) != 1 {
		return nil, fmt.Errorf("expected 1 but got %d worker pods", len(podList.Items))
	}

	return &podList.Items[0].Status, nil
}
