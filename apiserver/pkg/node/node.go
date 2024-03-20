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

package node

import (
	"context"
	"errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	graphqlutils "github.com/kubeagi/arcadia/apiserver/pkg/utils"
)

// obj2nodeConverter converts k8s object to page node
func obj2nodeConverter(obj client.Object) (generated.PageNode, error) {
	node, ok := obj.(*corev1.Node)
	if !ok {
		return nil, errors.New("can't convert object to Model")
	}
	n := &generated.Node{
		Name:   node.Name,
		Labels: graphqlutils.MapStr2Any(node.GetLabels()),
	}

	return n, nil
}

// ListK8sNodes list k8s nodes by filter
func ListK8sNodes(ctx context.Context, c client.Client, input generated.ListNodeInput) (*generated.PaginatedResult, error) {
	page := pointer.IntDeref(input.Page, 1)
	pageSize := pointer.IntDeref(input.PageSize, -1)

	// list nodes
	nodes := &corev1.NodeList{}
	opts, err := common.NewListOptions(generated.ListCommonInput{
		LabelSelector: input.LabelSelector,
	})
	if err != nil {
		return nil, err
	}
	err = c.List(ctx, nodes, opts...)
	if err != nil {
		return nil, err
	}

	items := make([]client.Object, len(nodes.Items))
	for i := range nodes.Items {
		items[i] = &nodes.Items[i]
	}

	return common.ListReources(items, page, pageSize, obj2nodeConverter)
}
