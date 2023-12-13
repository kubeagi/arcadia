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

	"github.com/KawashiroNitori/butcher/v2"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/config"
	"github.com/kubeagi/arcadia/pkg/datasource"
)

type Scheduler struct {
	ctx    context.Context
	cancel context.CancelFunc
	runner butcher.Butcher
	client client.Client

	ds     *v1alpha1.VersionedDataset
	remove bool
}

func NewScheduler(ctx context.Context, c client.Client, instance *v1alpha1.VersionedDataset, fileStatus []v1alpha1.FileStatus, remove bool) (*Scheduler, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx1, cancel := context.WithCancel(ctx)

	// TODO: Currently, we think there is only one default minio environment,
	// so we get the minio client directly through the configuration.
	systemDatasource, err := config.GetSystemDatasource(ctx1, c, nil)
	if err != nil {
		klog.Errorf("generate new scheduler error %s", err)
		cancel()
		return nil, err
	}
	endpoint := systemDatasource.Spec.Enpoint.DeepCopy()
	if endpoint.AuthSecret != nil && endpoint.AuthSecret.Namespace == nil {
		endpoint.AuthSecret.WithNameSpace(systemDatasource.Namespace)
	}
	oss, err := datasource.NewOSS(ctx1, c, nil, endpoint)
	if err != nil {
		cancel()
		klog.Errorf("generate new minio client error %s", err)
		return nil, err
	}

	s := &Scheduler{ctx: ctx1, cancel: cancel, ds: instance, client: c, remove: remove}
	exectuor, err := butcher.NewButcher[JobPayload](newExecutor(ctx1, c, oss, instance, fileStatus, remove), butcher.BufferSize(bufSize), butcher.MaxWorker(maxWorkers))
	if err != nil {
		cancel()
		return nil, err
	}
	s.runner = exectuor
	return s, nil
}

func (s *Scheduler) Start() error {
	if err := s.runner.Run(s.ctx); err != nil {
		klog.Errorf("versionDataset %s/%s run failed with err %s.", s.ds.Namespace, s.ds.Name, err)
		return err
	}

	// Only when there are no errors, the latest CR is fetched to check if the resource has changed.
	ds := &v1alpha1.VersionedDataset{}
	if err := s.client.Get(s.ctx, types.NamespacedName{Namespace: s.ds.Namespace, Name: s.ds.Name}, ds); err != nil {
		klog.Errorf("versionDataset %s/%s get failed. err %s", s.ds.Namespace, s.ds.Name, err)
		return err
	}

	if ds.DeletionTimestamp != nil {
		ds.Finalizers = nil
		klog.Infof("versionDataset %s/%s is being deleted, so we need to update his finalizers to allow the deletion to complete smoothly", ds.Namespace, ds.Name)
		return s.client.Update(s.ctx, ds)
	}
	if s.remove {
		return nil
	}

	if ds.ResourceVersion == s.ds.ResourceVersion {
		syncCond := true
		for _, cond := range ds.Status.Conditions {
			if cond.Type == v1alpha1.TypeReady && cond.Status == corev1.ConditionTrue && cond.Reason == v1alpha1.ReasonFileSuncSuccess {
				syncCond = false
			}
		}
		deepCopy := ds.DeepCopy()
		deepCopy.Status.Files = s.ds.Status.Files
		if syncCond {
			condition := v1alpha1.Condition{
				Type:               v1alpha1.TypeReady,
				Status:             corev1.ConditionTrue,
				LastTransitionTime: v1.Now(),
				Reason:             v1alpha1.ReasonFileSuncSuccess,
				Message:            "",
			}
			for _, checker := range s.ds.Status.Files {
				shouldBreak := false
				for _, f := range checker.Status {
					if f.Phase != v1alpha1.FileProcessPhaseSucceeded {
						condition.Status = corev1.ConditionFalse
						condition.Reason = v1alpha1.ReasonFileSyncFailed
						condition.Message = fmt.Sprintf("%s sync failed, %s", f.Path, f.ErrMessage)
						shouldBreak = true
					}
				}
				if shouldBreak {
					break
				}
			}
			deepCopy.Status.ConditionedStatus.SetConditions(condition)
		}

		return s.client.Status().Patch(s.ctx, deepCopy, client.MergeFrom(ds))
	}

	klog.Infof("current resourceversion: %s, previous resourceversion: %s", ds.ResourceVersion, s.ds.ResourceVersion)
	s.Stop()
	return nil
}

func (s *Scheduler) Stop() {
	s.cancel()
}
