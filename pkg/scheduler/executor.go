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
	"github.com/kubeagi/arcadia/pkg/datasource"
)

type executor struct {
	instance *v1alpha1.VersionedDataset
	client   client.Client

	deleteFileGroups []v1alpha1.DatasourceFileStatus
}

const (
	maxWorkers = 5
	bufSize    = 5
)

func newExecutor(ctx context.Context, c client.Client, instance *v1alpha1.VersionedDataset) butcher.Executor[JobPayload] {
	_, deleteFileGroups := v1alpha1.CopyedFileGroup2Status(instance)
	klog.V(4).Infof("[Debug] status is: %+v\ndelete filegroups: %+v\n", instance.Status.DatasourceFiles, deleteFileGroups)
	klog.V(4).Infof("[Debug] client is nil: %v\n", c == nil)
	return &executor{instance: instance, deleteFileGroups: deleteFileGroups, client: c}
}

func (e *executor) generateJob(ctx context.Context, jobCh chan<- JobPayload, datasourceFiles []v1alpha1.DatasourceFileStatus, removeAction bool) error {
	for _, fs := range datasourceFiles {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		ds := &v1alpha1.Datasource{}
		datasourceNamespace := fs.DatasourceNamespace
		datasetNamespace := e.instance.Namespace
		if e.instance.Spec.Dataset.Namespace != nil {
			datasetNamespace = *e.instance.Spec.Dataset.Namespace
		}
		if err := e.client.Get(ctx, types.NamespacedName{Namespace: fs.DatasourceNamespace, Name: fs.DatasourceName}, ds); err != nil {
			klog.Errorf("generateJob: failed to get datasource %s", err)
			return err
		}

		srcBucket := datasourceNamespace
		if ds.Spec.OSS != nil {
			srcBucket = ds.Spec.OSS.Bucket
		}
		dstBucket := datasetNamespace
		klog.V(4).Infof("[Debug] datasourceNamespace: %s, datasetNamespace: %s, srcBucket: %s, dstBucket: %s",
			datasourceNamespace, datasetNamespace, srcBucket, dstBucket)

		klog.V(5).Infof("[Debug] get datasource %+v\n", *ds)

		oss, err := datasource.NewOSS(ctx, e.client, ds.Spec.Enpoint)

		if err != nil {
			klog.Errorf("generateJob: get oss client error %s", err)
			return err
		}
		klog.V(4).Infof("[Debug] oss client is nil: %v", oss.Client == nil)

		bucketExists, err := oss.Client.BucketExists(ctx, dstBucket)
		if err != nil {
			klog.Errorf("generateJob: check for the presence of a bucket has failed %s.", err)
			return err
		}
		if !bucketExists {
			if err = oss.Client.MakeBucket(ctx, dstBucket, minio.MakeBucketOptions{}); err != nil {
				klog.Errorf("generateJob: failed to create bucket %s.", dstBucket)
				return err
			}
		}

		dst := fmt.Sprintf("dataset/%s/%s", e.instance.Spec.Dataset.Name, e.instance.Spec.Version)

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
				klog.Warningf("skip %s/%s, because it ends with /. this is not a legal object in oss.", fs.DatasourceName, fp.Path)
				continue
			}

			dstPath := fp.Path
			if !strings.HasPrefix(fp.Path, "/") {
				dstPath = "/" + fp.Path
			}

			payload := JobPayload{
				Src:            fp.Path,
				Dst:            dst + dstPath,
				DatasourceName: fs.DatasourceName,
				SrcBucket:      srcBucket,
				DstBucket:      dstBucket,
				Client:         oss.Client,
				Remove:         removeAction,
			}

			klog.V(4).Infof("[Debug] send %+v to jobch", payload)
			jobCh <- payload
		}
	}
	return nil
}
func (e *executor) GenerateJob(ctx context.Context, jobCh chan<- JobPayload) error {
	if err := e.generateJob(ctx, jobCh, e.instance.Status.DatasourceFiles, false); err != nil {
		klog.Errorf("GenerateJob: error %s", err)
		return err
	}
	return e.generateJob(ctx, jobCh, e.deleteFileGroups, true)
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
	klog.V(4).Infof("[Debug] change the status of file %s/%s to %s", job.DatasourceName, job.Src, syncStatus)
	if err = v1alpha1.UpdateFileStatus(ctx, e.instance, job.DatasourceName, job.Src, syncStatus, errMsg); err != nil {
		klog.Errorf("the job with payload %v completes, but updating the cr status fails %s.", job, err)
	}
}
