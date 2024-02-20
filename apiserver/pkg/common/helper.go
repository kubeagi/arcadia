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
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/apiserver/config"
)

type PageNodeSorter []client.Object

func (p *PageNodeSorter) Len() int {
	return len(*p)
}

func (p *PageNodeSorter) Less(i, j int) bool {
	sysmtemName := config.GetConfig().SystemNamespace

	nsa := (*p)[i].GetNamespace()
	nsb := (*p)[j].GetNamespace()
	a := (*p)[i].GetCreationTimestamp()
	b := (*p)[j].GetCreationTimestamp()

	if nsa == nsb {
		return a.After(b.Time)
	}

	// this is for model ordering and requires the system model to come first
	if nsa == sysmtemName {
		return true
	}
	if nsb == sysmtemName {
		return false
	}

	return a.After(b.Time)
}

func (p *PageNodeSorter) Swap(i, j int) {
	(*p)[i], (*p)[j] = (*p)[j], (*p)[i]
}

func (p *PageNodeSorter) Push(x any) {
	*p = append(*p, x.(client.Object))
}

func (p *PageNodeSorter) Pop() any {
	old := *p
	l := len(old)
	x := old[l-1]
	*p = old[:l-1]
	return x
}
