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

package ray

import (
	"context"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/pkg/config"
)

func ListRayClusters(ctx context.Context, c client.Client, input generated.ListCommonInput) (*generated.PaginatedResult, error) {
	clusters, err := config.GetRayClusters(ctx, c)
	if err != nil {
		return nil, err
	}

	var results = make([]generated.PageNode, 0, len(clusters))
	for index, cluster := range clusters {
		cluster := cluster
		// skip if keyword not in cluster name
		if input.Keyword != nil {
			if !strings.Contains(cluster.Name, *input.Keyword) {
				continue
			}
		}

		results = append(results, &generated.RayCluster{
			Index:         index,
			Name:          cluster.Name,
			HeadAddress:   &cluster.HeadAddress,
			DashboardHost: &cluster.DashboardHost,
			PythonVersion: &cluster.PythonVersion,
		})
	}

	return &generated.PaginatedResult{
		TotalCount: len(results),
		Nodes:      results,
	}, nil
}
