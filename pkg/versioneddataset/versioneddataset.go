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
package versioneddataset

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/minio/minio-go/v7"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/datasource"
)

func generateInheritedFileStatus(oss *datasource.OSS, instance *v1alpha1.VersionedDataset) []v1alpha1.FileStatus {
	srcBucket := instance.Spec.Dataset.Namespace
	prefix := fmt.Sprintf("dataset/%s/%s/", instance.Spec.Dataset.Name, instance.Spec.InheritedFrom)
	name := v1alpha1.InheritedFromVersionName + instance.Spec.InheritedFrom
	phase := v1alpha1.FileProcessPhaseProcessing
	if instance.Spec.InheritedFrom == "" {
		prefix = fmt.Sprintf("dataset/%s/%s/", instance.Spec.Dataset.Name, instance.Spec.Version)
		name = v1alpha1.InheritedFromVersionName + instance.Spec.Version
		phase = v1alpha1.FileProcessPhaseSucceeded
	}

	status := make([]v1alpha1.FileDetails, 0)
	if instance.Spec.InheritedFrom != "" {
		anyFilePaths, _ := oss.ListObjects(context.TODO(), *srcBucket, minio.ListObjectsOptions{
			Prefix:    prefix,
			Recursive: true,
		})
		filePaths := anyFilePaths.([]minio.ObjectInfo)
		sort.Slice(filePaths, func(i, j int) bool {
			return filePaths[i].Key < filePaths[j].Key
		})
		for _, fp := range filePaths {
			status = append(status, v1alpha1.FileDetails{
				Path:  strings.TrimPrefix(fp.Key, prefix),
				Phase: phase,
			})
		}
	}

	return []v1alpha1.FileStatus{
		{
			TypedObjectReference: v1alpha1.TypedObjectReference{
				Name:      name,
				Namespace: &instance.Namespace,
				Kind:      "VersionedDataset",
			},
			Status: status,
		}}
}

func generateDatasourceFileStatus(instance *v1alpha1.VersionedDataset) []v1alpha1.FileStatus {
	// 2. Organize the contents of the fileGroup into this format: {"datasourceNamespace datasourceName": ["file1", "file2"]}
	fileGroup := make(map[string][]string)
	for _, fg := range instance.Spec.FileGroups {
		namespace := instance.Namespace
		if fg.Source.Namespace != nil {
			namespace = *fg.Source.Namespace
		}
		key := fmt.Sprintf("%s %s", namespace, fg.Source.Name)
		if _, ok := fileGroup[key]; !ok {
			fileGroup[key] = make([]string, 0)
		}
		for i := range fg.Files {
			fileGroup[key] = append(fileGroup[key], fg.Files[i].Path)
		}
	}

	// 3. Convert fileGroup to []DatasourceFileStatus format
	targetDatasourceFileStatus := make([]v1alpha1.FileStatus, 0)
	var namespace, name string
	for datasource, filePaths := range fileGroup {
		_, _ = fmt.Sscanf(datasource, "%s %s", &namespace, &name)
		item := v1alpha1.FileStatus{
			TypedObjectReference: v1alpha1.TypedObjectReference{
				Name:      name,
				Namespace: &namespace,
				Kind:      "Datasource",
			},
			Status: []v1alpha1.FileDetails{},
		}
		for _, fp := range filePaths {
			item.Status = append(item.Status, v1alpha1.FileDetails{
				Path:  fp,
				Phase: v1alpha1.FileProcessPhaseProcessing,
			})
		}
		sort.Slice(item.Status, func(i, j int) bool {
			return item.Status[i].Path < item.Status[j].Path
		})

		targetDatasourceFileStatus = append(targetDatasourceFileStatus, item)
	}
	return targetDatasourceFileStatus
}

// CopiedFileGroup2Status the function will eventually return, whether there are new files added. and a list of files that were deleted.
func CopiedFileGroup2Status(oss *datasource.OSS, instance *v1alpha1.VersionedDataset) (bool, []v1alpha1.FileStatus) {
	if instance.DeletionTimestamp != nil {
		source := instance.Status.Files
		instance.Status.Files = nil
		return false, source
	}

	// 1. First store the information about the status of the file that has been saved in the current status.
	oldDatasourceFiles := make(map[string]map[string]v1alpha1.FileDetails)
	for _, fileStatus := range instance.Status.Files {
		key := fmt.Sprintf("%s %s", *fileStatus.Namespace, fileStatus.Name)
		if _, ok := oldDatasourceFiles[key]; !ok {
			oldDatasourceFiles[key] = make(map[string]v1alpha1.FileDetails)
		}
		for _, item := range fileStatus.Status {
			oldDatasourceFiles[key][item.Path] = item
		}
	}

	targetDatasourceFileStatus := generateDatasourceFileStatus(instance)
	targetDatasourceFileStatus = append(targetDatasourceFileStatus, generateInheritedFileStatus(oss, instance)...)

	// 4. If a file from a data source is found to exist in oldDatasourceFiles,
	// replace it with the book inside oldDatasourceFiles.
	// Otherwise set the file as being processed.
	update := false
	deletedFiles := make([]v1alpha1.FileStatus, 0)
	for idx := range targetDatasourceFileStatus {
		item := targetDatasourceFileStatus[idx]
		key := fmt.Sprintf("%s %s", *item.Namespace, item.Name)

		// if the datasource itself is not in status, then it is a new series of files added.
		datasourceFiles, ok := oldDatasourceFiles[key]
		if !ok {
			update = true
			continue
		}

		// We need to check if the file under spec has existed in status, if so, how to update its status, otherwise it is a new file.
		for i, status := range item.Status {
			oldFileStatus, ok := datasourceFiles[status.Path]
			if !ok {
				update = true
				continue
			}
			item.Status[i] = oldFileStatus

			// do the deletion here and the last data that still exists in the map then is the file that needs to be deleted.
			delete(datasourceFiles, status.Path)
		}
		if len(datasourceFiles) > 0 {
			ds := v1alpha1.FileStatus{
				TypedObjectReference: v1alpha1.TypedObjectReference{
					Name:      item.Name,
					Namespace: item.Namespace,
				},
				Status: make([]v1alpha1.FileDetails, 0),
			}
			for _, r := range datasourceFiles {
				ds.Status = append(ds.Status, r)
			}
			deletedFiles = append(deletedFiles, ds)
		}
		targetDatasourceFileStatus[idx] = item
		delete(oldDatasourceFiles, key)
	}

	for key, item := range oldDatasourceFiles {
		var namespace, name string
		fmt.Sscanf(key, "%s %s", &namespace, &name)
		ds := v1alpha1.FileStatus{
			TypedObjectReference: v1alpha1.TypedObjectReference{
				Name:      name,
				Namespace: &namespace,
			},
			Status: make([]v1alpha1.FileDetails, 0),
		}
		for _, r := range item {
			ds.Status = append(ds.Status, r)
		}
		deletedFiles = append(deletedFiles, ds)
	}

	sort.Slice(targetDatasourceFileStatus, func(i, j int) bool {
		return targetDatasourceFileStatus[i].Name < targetDatasourceFileStatus[j].Name
	})

	index := -1
	for idx, item := range instance.Status.Conditions {
		if item.Type == v1alpha1.TypeReady {
			if item.Status != corev1.ConditionTrue {
				index = idx
			}
			break
		}
	}
	if len(instance.Status.Conditions) == 0 || index != -1 {
		message := "sync files."
		if index != -1 {
			message = "file synchronization failed, try again"
		}
		cond := v1alpha1.Condition{
			Type:               v1alpha1.TypeReady,
			Status:             corev1.ConditionFalse,
			Reason:             v1alpha1.ReasonFileSyncing,
			Message:            message,
			LastTransitionTime: v1.Now(),
		}
		instance.Status.ConditionedStatus.SetConditions(cond)
	}
	klog.V(4).Infof("[Debug] delete filestatus %+v\n", deletedFiles)

	instance.Status.Files = targetDatasourceFileStatus
	// update condition to sync
	return update, deletedFiles
}

func UpdateFileStatus(ctx context.Context, instance *v1alpha1.VersionedDataset, datasource, srcPath string, syncStatus v1alpha1.FileProcessPhase, errMsg string) error {
	datasourceFileLen := len(instance.Status.Files)
	datasourceIndex := sort.Search(datasourceFileLen, func(i int) bool {
		return instance.Status.Files[i].Name >= datasource
	})
	if datasourceIndex == datasourceFileLen {
		return fmt.Errorf("not found datasource %s in %s/%s.status", datasource, instance.Namespace, instance.Name)
	}

	filePathLen := len(instance.Status.Files[datasourceIndex].Status)
	fileIndex := sort.Search(filePathLen, func(i int) bool {
		return instance.Status.Files[datasourceIndex].Status[i].Path >= srcPath
	})
	if fileIndex == filePathLen {
		return fmt.Errorf("not found srcPath %s in datasource %s", srcPath, datasource)
	}

	// Only this state transfer is allowed
	curPhase := instance.Status.Files[datasourceIndex].Status[fileIndex].Phase
	if curPhase == v1alpha1.FileProcessPhaseProcessing && (syncStatus == v1alpha1.FileProcessPhaseSucceeded || syncStatus == v1alpha1.FileProcessPhaseFailed) {
		instance.Status.Files[datasourceIndex].Status[fileIndex].Phase = syncStatus
		if syncStatus == v1alpha1.FileProcessPhaseFailed {
			instance.Status.Files[datasourceIndex].Status[fileIndex].ErrMessage = errMsg
		}
		if syncStatus == v1alpha1.FileProcessPhaseSucceeded {
			instance.Status.Files[datasourceIndex].Status[fileIndex].LastUpdateTime = v1.Now()
		}
		return nil
	}

	return fmt.Errorf("wrong state. from %s to %s", curPhase, syncStatus)
}
