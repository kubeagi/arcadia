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
package modelservice

import (
	"container/heap"

	"github.com/kubeagi/arcadia/apiserver/graph/generated"
)

// Because here the merger of llm and embedders two kinds of data,
// so the amount of data than one kind of quite a bit more,
// so the direct sort may not be an optimal solution,
// we can use the heap, and start, end the two parameters do not arrange all the data, and then find the data to be paged.
type ModelServiceList []*generated.ModelService

func (m *ModelServiceList) Len() int {
	return len(*m)
}

func (m *ModelServiceList) Less(i, j int) bool {
	a := (*m)[i]
	b := (*m)[j]
	if a.CreationTimestamp.Equal(*b.CreationTimestamp) {
		return a.Name < b.Name
	}
	return a.CreationTimestamp.After(*b.CreationTimestamp)
}

func (m *ModelServiceList) Swap(i, j int) {
	(*m)[i], (*m)[j] = (*m)[j], (*m)[i]
}

func (m *ModelServiceList) Push(x any) {
	*m = append(*m, x.(*generated.ModelService))
}

func (m *ModelServiceList) Pop() any {
	old := *m
	l := len(old)
	x := old[l-1]
	*m = old[:l-1]
	return x
}

func pageModelService(start, end int, list *[]*generated.ModelService) []*generated.ModelService {
	l := ModelServiceList(*list)
	heap.Init(&l)

	r := make([]*generated.ModelService, 0)
	for cur := 0; cur < end && l.Len() > 0; cur++ {
		top := heap.Pop(&l)
		if cur < start {
			continue
		}
		r = append(r, top.(*generated.ModelService))
	}
	return r
}
