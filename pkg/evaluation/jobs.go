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
package evaluation

import (
	"context"
	"fmt"
	"path/filepath"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/env"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	evav1alpha1 "github.com/kubeagi/arcadia/api/evaluation/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/config"
	"github.com/kubeagi/arcadia/pkg/llms"
	"github.com/kubeagi/arcadia/pkg/utils"
)

const (
	defaultPVCMountPath = "/data/evaluations"
	defaultTestRagFile  = "ragas.csv"
	defaultMCImage      = "kubeagi/minio-mc:RELEASE.2023-01-28T20-29-38Z"
)

func PhaseJobName(instance *evav1alpha1.RAG, phase evav1alpha1.RAGPhase) string {
	return fmt.Sprintf("%s-phase-%s", instance.Name, phase)
}

func DownloadJob(instance *evav1alpha1.RAG) (*batchv1.Job, error) {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: instance.Namespace,
			Name:      PhaseJobName(instance, evav1alpha1.DownloadFilesPhase),
			Labels: map[string]string{
				evav1alpha1.EvaluationJobLabels: instance.Name,
			},
		},
		Spec: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					RestartPolicy:      v1.RestartPolicyNever,
					ServiceAccountName: instance.Spec.ServiceAccountName,
					Containers: []v1.Container{
						{
							Name:  "download-dataset-files",
							Image: "kubeagi/arcadia-eval",
							Command: []string{
								"arctl",
							},
							Args: []string{
								fmt.Sprintf("-n=%s", instance.Namespace),
								"eval", "download",
								fmt.Sprintf("--rag=%s", instance.Name),
								fmt.Sprintf("--application=%s", instance.Spec.Application.Name),
								fmt.Sprintf("--dir=%s", defaultPVCMountPath),
								fmt.Sprintf("--system-conf-namespace=%s", utils.GetCurrentNamespace()),
								fmt.Sprintf("--system-conf-name=%s", env.GetString(config.EnvConfigKey, config.EnvConfigDefaultValue)),
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "data",
									MountPath: defaultPVCMountPath,
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "data",
							VolumeSource: v1.VolumeSource{
								PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
									ClaimName: instance.Name,
									ReadOnly:  false,
								},
							},
						},
					},
				},
			},
			BackoffLimit: pointer.Int32(1),
			Completions:  pointer.Int32(1),
			Parallelism:  pointer.Int32(1),
			Suspend:      &instance.Spec.Suspend,
		},
	}
	return job, nil
}

func GenTestDataJob(instance *evav1alpha1.RAG) (*batchv1.Job, error) {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: instance.Namespace,
			Name:      PhaseJobName(instance, evav1alpha1.GenerateTestFilesPhase),
			Labels: map[string]string{
				evav1alpha1.EvaluationJobLabels: instance.Name,
			},
		},
		Spec: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					RestartPolicy:      v1.RestartPolicyNever,
					ServiceAccountName: instance.Spec.ServiceAccountName,
					Containers: []v1.Container{
						{
							Name:  "gen-test-files",
							Image: "kubeagi/arcadia-eval",
							Command: []string{
								"arctl",
							},
							Args: []string{
								fmt.Sprintf("-n=%s", instance.Namespace),
								"eval", "gen_test_dataset",
								fmt.Sprintf("--application=%s", instance.Spec.Application.Name),
								fmt.Sprintf("--input-dir=%s", defaultPVCMountPath),
								"--output=csv",
								"--merge=true",
								fmt.Sprintf("--merge-file=%s", filepath.Join(defaultPVCMountPath, defaultTestRagFile)),
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "data",
									MountPath: defaultPVCMountPath,
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "data",
							VolumeSource: v1.VolumeSource{
								PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
									ClaimName: instance.Name,
									ReadOnly:  false,
								},
							},
						},
					},
				},
			},
			BackoffLimit: pointer.Int32(1),
			Completions:  pointer.Int32(1),
			Parallelism:  pointer.Int32(1),
			Suspend:      &instance.Spec.Suspend,
		},
	}
	return job, nil
}

func JudgeJobGenerator(ctx context.Context, c client.Client) func(*evav1alpha1.RAG) (*batchv1.Job, error) {
	return func(instance *evav1alpha1.RAG) (*batchv1.Job, error) {
		var (
			apiBase, model, apiKey string
			err                    error
		)
		llm := v1alpha1.LLM{}
		ns := instance.Namespace
		if instance.Spec.JudgeLLM.Namespace != nil {
			ns = *instance.Spec.JudgeLLM.Namespace
		}
		if err = c.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: instance.Spec.JudgeLLM.Name}, &llm); err != nil {
			return nil, err
		}

		apiBase = llm.Get3rdPartyLLMBaseURL()
		apiKey, err = llm.AuthAPIKey(ctx, c, nil)
		if err != nil {
			return nil, err
		}

		switch llm.Spec.Type {
		case llms.OpenAI:
			model = "gtp4"
		case llms.ZhiPuAI:
			model = "glm-4"
		default:
			return nil, fmt.Errorf("not support type %s", llm.Spec.Type)
		}
		if r := llm.Get3rdPartyModels(); len(r) > 0 {
			model = r[0]
		}

		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: instance.Namespace,
				Name:      PhaseJobName(instance, evav1alpha1.JudgeLLMPhase),
				Labels: map[string]string{
					evav1alpha1.EvaluationJobLabels: instance.Name,
				},
			},

			Spec: batchv1.JobSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						RestartPolicy:      v1.RestartPolicyNever,
						ServiceAccountName: instance.Spec.ServiceAccountName,
						Containers: []v1.Container{
							{
								Name:       "judge-llm",
								Image:      "kubeagi/arcadia-eval:v0.1.0",
								WorkingDir: defaultPVCMountPath,
								Command: []string{
									"python3",
								},
								Args: []string{
									"-m",
									"ragas_once.cli",
									fmt.Sprintf("--apibase=%s", apiBase),
									fmt.Sprintf("--model=%s", model),
									fmt.Sprintf("--apikey=%s", apiKey),
									fmt.Sprintf("--dataset=%s", filepath.Join(defaultPVCMountPath, defaultTestRagFile)),
								},
								VolumeMounts: []v1.VolumeMount{
									{
										Name:      "data",
										MountPath: defaultPVCMountPath,
									},
								},
							},
						},
						Volumes: []v1.Volume{
							{
								Name: "data",
								VolumeSource: v1.VolumeSource{
									PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
										ClaimName: instance.Name,
										ReadOnly:  false,
									},
								},
							},
						},
					},
				},
				BackoffLimit: pointer.Int32(1),
				Completions:  pointer.Int32(1),
				Parallelism:  pointer.Int32(1),
				Suspend:      &instance.Spec.Suspend,
			},
		}
		return job, nil
	}
}

func UploadJobGenerator(ctx context.Context, client client.Client) func(*evav1alpha1.RAG) (*batchv1.Job, error) {
	return func(instance *evav1alpha1.RAG) (*batchv1.Job, error) {
		datasource, err := config.GetSystemDatasource(ctx, client, nil)
		if err != nil {
			return nil, err
		}
		url := datasource.Spec.Endpoint.URL
		if datasource.Spec.Endpoint.Insecure {
			url = "http://" + url
		} else {
			url = "https://" + url
		}
		ns := datasource.Namespace
		if datasource.Spec.Endpoint.AuthSecret.Namespace != nil {
			ns = *datasource.Spec.Endpoint.AuthSecret.Namespace
		}
		data, err := datasource.Spec.Endpoint.AuthData(ctx, ns, client, nil)
		if err != nil {
			return nil, err
		}

		accessKeyID := string(data["rootUser"])
		secretAccessKey := string(data["rootPassword"])

		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: instance.Namespace,
				Name:      PhaseJobName(instance, evav1alpha1.UploadFilesPhase),
				Labels: map[string]string{
					evav1alpha1.EvaluationJobLabels: instance.Name,
				},
			},
			Spec: batchv1.JobSpec{
				Template: v1.PodTemplateSpec{
					Spec: v1.PodSpec{
						RestartPolicy:      v1.RestartPolicyNever,
						ServiceAccountName: instance.Spec.ServiceAccountName,
						Containers: []v1.Container{
							{
								Name:  "upload-result",
								Image: defaultMCImage,
								Command: []string{
									"/bin/bash",
									"-c",
									fmt.Sprintf(`echo "upload result"
mc alias set oss $MINIO_ENDPOINT $MINIO_ACCESS_KEY $MINIO_SECRET_KEY --insecure
mc --insecure cp -r %s/ oss/%s/evals/%s/%s`, defaultPVCMountPath, instance.Namespace, instance.Spec.Application.Name, instance.Name),
								},
								VolumeMounts: []v1.VolumeMount{
									{
										Name:      "data",
										MountPath: defaultPVCMountPath,
									},
								},
								Env: []v1.EnvVar{
									{Name: "MINIO_ENDPOINT", Value: url},
									{Name: "MINIO_ACCESS_KEY", Value: accessKeyID},
									{Name: "MINIO_SECRET_KEY", Value: secretAccessKey},
								},
							},
						},
						Volumes: []v1.Volume{
							{
								Name: "data",
								VolumeSource: v1.VolumeSource{
									PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
										ClaimName: instance.Name,
										ReadOnly:  false,
									},
								},
							},
						},
					},
				},
				BackoffLimit: pointer.Int32(1),
				Completions:  pointer.Int32(1),
				Parallelism:  pointer.Int32(1),
				Suspend:      &instance.Spec.Suspend,
			},
		}
		return job, nil
	}
}
