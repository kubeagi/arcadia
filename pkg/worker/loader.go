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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/datasource"
)

const (
	defaultOSSLoaderImage  = "kubeagi/minio-mc:RELEASE.2023-01-28T20-29-38Z"
	defaultRDMALoaderImage = "wetman2023/floo:23.12"
)

// ModelLoader load models for worker
type ModelLoader interface {
	Build(ctx context.Context, model *arcadiav1alpha1.TypedObjectReference) (any, error)
}

var _ ModelLoader = (*LoaderOSS)(nil)

// LoaderOSS defines the way to load model from oss
type LoaderOSS struct {
	c client.Client

	endpoint *arcadiav1alpha1.Endpoint
	oss      *datasource.OSS
	worker   *arcadiav1alpha1.Worker
}

func NewLoaderOSS(ctx context.Context, c client.Client, endpoint *arcadiav1alpha1.Endpoint, worker *arcadiav1alpha1.Worker) (ModelLoader, error) {
	if endpoint == nil {
		return nil, errors.New("nil oss endpoint")
	}

	oss, err := datasource.NewOSS(ctx, c, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to new oss client with %w", err)
	}

	return &LoaderOSS{
		c:        c,
		endpoint: endpoint,
		oss:      oss,
		worker:   worker,
	}, nil
}

// Load nothing inner go code
func (loader *LoaderOSS) Build(ctx context.Context, model *arcadiav1alpha1.TypedObjectReference) (any, error) {
	if model == nil || model.Namespace == nil {
		return nil, errors.New("nil model or nil model namespace")
	}
	err := loader.oss.Stat(ctx, &arcadiav1alpha1.OSS{
		Bucket: *model.Namespace,
		Object: fmt.Sprintf("model/%s/", model.Name),
	})
	if err != nil {
		return nil, err
	}

	var accessKeyID, secretAccessKey string
	if loader.endpoint.AuthSecret != nil {
		secret := corev1.Secret{}
		if err := loader.c.Get(ctx, types.NamespacedName{
			Namespace: *loader.endpoint.AuthSecret.Namespace,
			Name:      loader.endpoint.AuthSecret.Name,
		}, &secret); err != nil {
			return nil, err
		}
		accessKeyID = string(secret.Data["rootUser"])
		secretAccessKey = string(secret.Data["rootPassword"])

		// TODO: implement https(secure check)
		// if !endpoint.Insecure {
		// }
	}

	// user internal url if not empty
	url := loader.endpoint.SchemeURL()
	if loader.endpoint.InternalURL != "" {
		loader.endpoint.Insecure = true
		url = loader.endpoint.SchemeInternalURL()
	}

	img := defaultOSSLoaderImage
	if loader.worker.Spec.Loader.Image != "" {
		img = loader.worker.Spec.Loader.Image
	}
	container := &corev1.Container{
		Name:            "loader",
		Image:           img,
		ImagePullPolicy: loader.worker.Spec.Loader.ImagePullPolicy,
		Command: []string{
			"/bin/bash",
			"-c",
			`echo "Load models"
mc alias set oss $MINIO_ENDPOINT $MINIO_ACCESS_KEY $MINIO_SECRET_KEY --insecure
mc --insecure cp -r oss/$MINIO_MODEL_BUCKET/model/$MINIO_MODEL_NAME /data/models`},
		Env: []corev1.EnvVar{
			{Name: "MINIO_ENDPOINT", Value: url},
			{Name: "MINIO_ACCESS_KEY", Value: accessKeyID},
			{Name: "MINIO_SECRET_KEY", Value: secretAccessKey},
			// Bucket should be the same as current namespace
			{Name: "MINIO_MODEL_BUCKET", Value: *model.Namespace},
			{Name: "MINIO_MODEL_NAME", Value: model.Name},
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "models", MountPath: "/data/models"},
		},
	}

	return container, nil
}

var _ ModelLoader = (*LoaderGit)(nil)

// LoaderGit defines the way to load model from git
type LoaderGit struct{}

func (loader *LoaderGit) Build(ctx context.Context, model *arcadiav1alpha1.TypedObjectReference) (any, error) {
	return nil, ErrNotImplementedYet
}

var _ ModelLoader = (*RDMALoader)(nil)

// RDMALoader Support for RDMA.
// Allow Pod to user hostpath and RDMA to pull models faster and start services
type RDMALoader struct {
	c client.Client

	modelName string
	// workerUID/modelname is the local model storage path
	workerUID string

	datasource *arcadiav1alpha1.Datasource
	worker     *arcadiav1alpha1.Worker
}

func NewRDMALoader(c client.Client, modelName, workerUID string, source *arcadiav1alpha1.Datasource, worker *arcadiav1alpha1.Worker) *RDMALoader {
	return &RDMALoader{c: c, modelName: modelName, workerUID: workerUID, datasource: source, worker: worker}
}

func (r *RDMALoader) Build(ctx context.Context, _ *arcadiav1alpha1.TypedObjectReference) (any, error) {
	rdmaEndpoint := r.datasource.Spec.Endpoint.URL
	remoteBaseSavePath := r.datasource.Spec.RDMA.Path

	img := defaultRDMALoaderImage
	if r.worker.Spec.Loader.Image != "" {
		img = r.worker.Spec.Loader.Image
	}
	container := &corev1.Container{
		Name:            "rdma-loader",
		Image:           img,
		ImagePullPolicy: r.worker.Spec.Loader.ImagePullPolicy,
		Command: []string{
			"/bin/bash",
			"-c",
			// pulls files from the service's 'rdmaEndpoint:/remoteBaseSavePath/modelName' directory to the local 'UID' directory.
			fmt.Sprintf("floo_get --from=%s --to=$TO --srv=%s --dir=%s%s", rdmaEndpoint, r.workerUID, remoteBaseSavePath, r.modelName),
		},
		Env: []corev1.EnvVar{
			{
				Name: "TO",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "status.hostIP",
					},
				},
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "tmp",
				MountPath: "/tmp",
			},
		},
	}
	return container, nil
}
