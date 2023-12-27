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

package appnode

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	node "github.com/kubeagi/arcadia/api/app-node"
)

func CheckAndUpdateAnnotation(ctx context.Context, log logr.Logger, cli client.Client, instance client.Object) error {
	instanceCopy := instance.DeepCopyObject()
	n, _ := instanceCopy.(node.Node)
	n.SetRef()
	instanceUpdate, _ := n.(client.Object)
	if !reflect.DeepEqual(instanceUpdate.GetAnnotations(), instance.GetAnnotations()) {
		if err := cli.Patch(ctx, instanceUpdate, client.MergeFrom(instance)); err != nil {
			log.Error(err, "Failed to update instance rule annotation", "instance", instanceUpdate)
			return err
		}
	}
	return nil
}
