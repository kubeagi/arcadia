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
package common

import (
	"container/heap"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/apiserver/graph/generated"
)

// ListReources filtering resources based on conditions will modify the original array,
// so if you want to preserve the original data, make a backup before calling the function
func ListReources(items []client.Object, page, pageSize int, converter ResourceConverter, options ...ResourceFilter) (*generated.PaginatedResult, error) {
	index, optIndex := 0, 0
	for i := range items {
		for optIndex = 0; optIndex < len(options); optIndex++ {
			if !options[optIndex](items[i]) {
				break
			}
		}
		if optIndex == len(options) {
			items[index] = items[i]
			index++
		}
	}
	items = items[:index]

	total := len(items)
	start, end := PagePosition(page, pageSize, total)
	result := &generated.PaginatedResult{
		TotalCount:  total,
		HasNextPage: end < total,
		Nodes:       make([]generated.PageNode, 0),
	}

	if start >= total {
		return result, nil
	}

	h := PageNodeSorter(items)
	heap.Init(&h)
	for cur := 0; h.Len() > 0 && cur < end; cur++ {
		top := heap.Pop(&h).(client.Object)
		if cur >= start {
			node, err := converter(top)
			if err != nil {
				return nil, err
			}
			result.Nodes = append(result.Nodes, node)
		}
	}

	return result, nil
}
