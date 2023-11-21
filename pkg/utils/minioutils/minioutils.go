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

package minioutils

import (
	"context"
	"strings"

	"github.com/minio/minio-go/v7"
)

func ListObjects(ctx context.Context, bucket, prefix string, client *minio.Client, maxDep int) []string {
	result := make([]string, 0)
	q := []string{prefix}
	depth := 0

	for len(q) > 0 && (maxDep <= 0 || depth < maxDep) {
		nq := make([]string, 0)
		for _, p := range q {
			for key := range client.ListObjects(ctx, bucket, minio.ListObjectsOptions{Prefix: p}) {
				if strings.HasSuffix(key.Key, "/") {
					nq = append(nq, key.Key)
					continue
				}
				result = append(result, key.Key)
			}
		}
		q = nq
		depth++
	}
	return result
}

func ListObjectCompleteInfo(ctx context.Context, bucket, prefix string, client *minio.Client, maxDep int) []minio.ObjectInfo {
	result := make([]minio.ObjectInfo, 0)
	q := []string{prefix}
	depth := 0
	for len(q) > 0 && (maxDep <= 0 || depth < maxDep) {
		nq := make([]string, 0)
		for _, p := range q {
			objList := client.ListObjects(ctx, bucket, minio.ListObjectsOptions{
				Prefix: p,
			})
			for key := range objList {
				if strings.HasSuffix(key.Key, "/") {
					nq = append(nq, key.Key)
					continue
				}
				result = append(result, key)
			}
		}
		q = nq
		depth++
	}
	return result
}
