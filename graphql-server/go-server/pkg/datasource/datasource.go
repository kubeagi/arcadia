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

package datasource

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeagi/arcadia/api/v1alpha1"
	"github.com/kubeagi/arcadia/graphql-server/go-server/graph/model"
	"github.com/kubeagi/arcadia/graphql-server/go-server/pkg/client"
)

func datasource2model(obj *unstructured.Unstructured) *model.Datasource {
	labels := make(map[string]interface{})
	for k, v := range obj.GetLabels() {
		labels[k] = v
	}
	annotations := make(map[string]interface{})
	for k, v := range obj.GetAnnotations() {
		annotations[k] = v
	}
	url, _, _ := unstructured.NestedString(obj.Object, "spec", "url")
	authsecret, _, _ := unstructured.NestedString(obj.Object, "spec", "atuhsecret")
	spec := model.DatasourceSpec{
		URL:        &url,
		Authsecret: &authsecret,
	}
	md := model.Datasource{
		Kind:              obj.GetKind(),
		APIVersion:        obj.GetAPIVersion(),
		Name:              obj.GetName(),
		Namespace:         obj.GetNamespace(),
		UID:               string(obj.GetUID()),
		ResourceVersion:   obj.GetResourceVersion(),
		Generation:        int(obj.GetGeneration()),
		CreationTimestamp: obj.GetCreationTimestamp().Time,
		Spec:              &spec,
	}
	return &md
}

func CreateDatasource(ctx context.Context, name, namespace, url, authsecret string) (*model.Datasource, error) {
	c := client.GetClient()
	datasource := v1alpha1.Datasource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Datasource",
			APIVersion: v1alpha1.GroupVersion.String(),
		},
		Spec: v1alpha1.DatasourceSpec{
			URL:        url,
			AuthSecret: authsecret,
		},
	}

	unstructuredDatasource, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&datasource)
	if err != nil {
		return nil, err
	}
	obj, err := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "datasources"}).
		Namespace(namespace).Create(ctx, &unstructured.Unstructured{Object: unstructuredDatasource}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	ds := datasource2model(obj)
	return ds, nil
}

func DatasourceList(ctx context.Context, name, namespace, labelSelector, fieldSelector string) ([]*model.Datasource, error) {
	dsSchema := schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "datasources"}
	c := client.GetClient()
	if name != "" {
		u, err := c.Resource(dsSchema).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return []*model.Datasource{datasource2model(u)}, nil
	}
	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	}
	us, err := c.Resource(dsSchema).Namespace(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}
	result := make([]*model.Datasource, len(us.Items))
	for idx, u := range us.Items {
		result[idx] = datasource2model(&u)
	}
	return result, nil
}
