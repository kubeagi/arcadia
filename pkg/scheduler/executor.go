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
package scheduler

import (
	"context"
	"fmt"
	"strings"

	"github.com/KawashiroNitori/butcher/v2"
	"github.com/minio/minio-go/v7"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/v1alpha1"
)

type executor struct {
	instance    *v1alpha1.VersionedDataset
	client      client.Client
	minioClient *minio.Client

	fileStatus []v1alpha1.FileStatus
	remove     bool
}

const (
	maxWorkers = 5
	bufSize    = 5
)

func newExecutor(ctx context.Context, c client.Client, minioClient *minio.Client, instance *v1alpha1.VersionedDataset, fileStatus []v1alpha1.FileStatus, remove bool) butcher.Executor[JobPayload] {
	klog.V(4).Infof("[Debug] client is nil: %v\n", c == nil)
	return &executor{instance: instance, fileStatus: fileStatus, client: c, minioClient: minioClient, remove: remove}
}

func (e *executor) generateJob(ctx context.Context, jobCh chan<- JobPayload, datasourceFiles []v1alpha1.FileStatus, removeAction bool) error {
	for _, fs := range datasourceFiles {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		dstBucket := e.instance.Namespace
		dstPrefix := fmt.Sprintf("dataset/%s/%s/", e.instance.Spec.Dataset.Name, e.instance.Spec.Version)

		var srcBucket, srcPrefix string
		if !removeAction {
			switch fs.Kind {
			case "Datasource":
				// Since the data source can be configured with different minio addresses,
				// it may involve copying of data from different minio,
				// which may result in the operator memory increasing to be OOM.
				// so currently it is considered that all operations are in the same minio.
				ds := &v1alpha1.Datasource{}
				if err := e.client.Get(ctx, types.NamespacedName{Namespace: *fs.Namespace, Name: fs.Name}, ds); err != nil {
					klog.Errorf("generateJob: failed to get datasource %s", err)
					return err
				}
				srcBucket = *fs.Namespace
				if ds.Spec.OSS != nil {
					srcBucket = ds.Spec.OSS.Bucket
				}
			case "VersionedDataset":
				srcVersion := fs.Name[len(v1alpha1.InheritedFromVersionName):]
				srcBucket = e.instance.Namespace
				srcPrefix = fmt.Sprintf("dataset/%s/%s/", e.instance.Spec.Dataset.Name, srcVersion)
			default:
				klog.Errorf("currently, copying data from a data source of the type %s is not supported", fs.Kind)
				continue
			}
		}

		bucketExists, err := e.minioClient.BucketExists(ctx, dstBucket)
		if err != nil {
			klog.Errorf("generateJob: check for the presence of a bucket has failed %s.", err)
			return err
		}
		if !bucketExists {
			if err = e.minioClient.MakeBucket(ctx, dstBucket, minio.MakeBucketOptions{}); err != nil {
				klog.Errorf("generateJob: failed to create bucket %s.", dstBucket)
				return err
			}
		}

		for _, fp := range fs.Status {
			select {
			case <-ctx.Done():
				return nil
			default:
			}

			if !removeAction && fp.Phase != v1alpha1.FileProcessPhaseProcessing {
				klog.V(4).Infof("[Bebug] copy object: %v, curPhase: %s, skip", removeAction, fp.Phase)
				continue
			}

			if strings.HasSuffix(fp.Path, "/") {
				klog.Warningf("skip %s/%s, because it ends with /. this is not a legal object in oss.", fs.Name, fp.Path)
				continue
			}

			targetPath := strings.TrimPrefix(fp.Path, "/")

			payload := JobPayload{
				Src:        srcPrefix + targetPath,
				Dst:        dstPrefix + targetPath,
				SourceName: fs.Name,
				SrcBucket:  srcBucket,
				DstBucket:  dstBucket,
				Client:     e.minioClient,
				Remove:     removeAction,
			}

			klog.V(4).Infof("[Debug] send %+v to jobch", payload)
			jobCh <- payload
		}
	}
	return nil
}

func (e *executor) GenerateJob(ctx context.Context, jobCh chan<- JobPayload) error {
	if e.remove {
		return e.generateJob(ctx, jobCh, e.fileStatus, true)
	}
	return e.generateJob(ctx, jobCh, e.instance.Status.Files, false)
}

func (e *executor) Task(ctx context.Context, job JobPayload) error {
	if !job.Remove {
		klog.V(4).Infof("[Debug] copyObject task from %s/%s to %s/%s", job.SrcBucket, job.Src, job.DstBucket, job.Dst)
		_, err := job.Client.CopyObject(ctx, minio.CopyDestOptions{
			Bucket: job.DstBucket,
			Object: job.Dst,
		}, minio.CopySrcOptions{
			Bucket: job.SrcBucket,
			Object: job.Src,
		})
		klog.V(4).Infof("[Debug] copy object from %s to %s, result: %s", job.Src, job.Dst, err)
		return err
	}

	err := job.Client.RemoveObject(ctx, job.DstBucket, job.Dst, minio.RemoveObjectOptions{})
	klog.V(4).Infof("[Debug] removeObject %s/%s result %s", job.DstBucket, job.Dst, err)

	return err
}

func (e *executor) OnFinish(ctx context.Context, job JobPayload, err error) {
	if job.Remove {
		klog.V(4).Infof("[Debug] OnFinish, removeObject done, don't need to updated file status, result %s", err)
		return
	}

	syncStatus := v1alpha1.FileProcessPhaseSucceeded
	errMsg := ""
	if err != nil {
		syncStatus = v1alpha1.FileProcessPhaseFailed
		errMsg = err.Error()
	}
	src := job.Src
	if strings.HasPrefix(job.SourceName, v1alpha1.InheritedFromVersionName) {
		p := fmt.Sprintf("dataset/%s/%s/", e.instance.Spec.Dataset.Name, e.instance.Spec.InheritedFrom)
		src = strings.TrimPrefix(src, p)
	}
	klog.V(4).Infof("[Debug] change the status of file %s/%s to %s", job.SourceName, src, syncStatus)

	if err = v1alpha1.UpdateFileStatus(ctx, e.instance, job.SourceName, src, syncStatus, errMsg); err != nil {
		klog.Errorf("the job with payload %v completes, but updating the cr status fails %s.", job, err)
	}
}
