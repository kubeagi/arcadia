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
	"crypto/tls"
	"net/http"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/utils/pointer"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	graphqlutils "github.com/kubeagi/arcadia/apiserver/pkg/utils"
	"github.com/kubeagi/arcadia/pkg/datasource"
	"github.com/kubeagi/arcadia/pkg/utils"
)

func datasource2modelConverter(obj *unstructured.Unstructured) (generated.PageNode, error) {
	return datasource2model(obj)
}

func datasource2model(obj *unstructured.Unstructured) (*generated.Datasource, error) {
	datasource := &v1alpha1.Datasource{}
	if err := utils.UnstructuredToStructured(obj, datasource); err != nil {
		return nil, err
	}

	id := string(datasource.GetUID())

	creationtimestamp := datasource.GetCreationTimestamp().Time

	// conditioned status
	condition := datasource.Status.GetCondition(v1alpha1.TypeReady)
	updateTime := condition.LastTransitionTime.Time
	message := string(condition.Message)
	status := common.GetObjStatus(datasource)

	// parse endpoint
	endpoint := generated.Endpoint{
		URL:      &datasource.Spec.Endpoint.URL,
		Insecure: &datasource.Spec.Endpoint.Insecure,
	}
	if datasource.Spec.Endpoint.AuthSecret != nil {
		endpoint.AuthSecret = &generated.TypedObjectReference{
			Kind:      "Secret",
			Name:      datasource.Spec.Endpoint.AuthSecret.Name,
			Namespace: datasource.Spec.Endpoint.AuthSecret.Namespace,
		}
	}

	// parse oss
	oss := generated.Oss{}
	if datasource.Spec.OSS != nil {
		oss.Bucket = &datasource.Spec.OSS.Bucket
		oss.Object = &datasource.Spec.OSS.Object
	}

	md := generated.Datasource{
		ID:                &id,
		Name:              datasource.Name,
		Namespace:         datasource.Namespace,
		Creator:           pointer.String(datasource.Spec.Creator),
		Labels:            graphqlutils.MapStr2Any(obj.GetLabels()),
		Annotations:       graphqlutils.MapStr2Any(obj.GetAnnotations()),
		DisplayName:       &datasource.Spec.DisplayName,
		Description:       &datasource.Spec.Description,
		Endpoint:          &endpoint,
		Oss:               &oss,
		Status:            &status,
		Message:           &message,
		CreationTimestamp: &creationtimestamp,
		UpdateTimestamp:   &updateTime,
	}
	return &md, nil
}

func CreateDatasource(ctx context.Context, c dynamic.Interface, input generated.CreateDatasourceInput) (*generated.Datasource, error) {
	var displayname, description string

	if input.Description != nil {
		description = *input.Description
	}
	if input.DisplayName != nil {
		displayname = *input.DisplayName
	}

	// create datasource
	datasource := &v1alpha1.Datasource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Datasource",
			APIVersion: v1alpha1.GroupVersion.String(),
		},
		Spec: v1alpha1.DatasourceSpec{
			CommonSpec: v1alpha1.CommonSpec{
				DisplayName: displayname,
				Description: description,
			},
		},
	}
	common.SetCreator(ctx, &datasource.Spec.CommonSpec)

	// make endpoint
	endpoint, err := common.MakeEndpoint(ctx, c, generated.TypedObjectReferenceInput{
		APIGroup:  &datasource.APIVersion,
		Kind:      datasource.Kind,
		Name:      datasource.Name,
		Namespace: &datasource.Namespace,
	}, input.Endpointinput)
	if err != nil {
		return nil, err
	}
	datasource.Spec.Endpoint = endpoint

	if input.Ossinput != nil {
		datasource.Spec.OSS = &v1alpha1.OSS{
			Bucket: input.Ossinput.Bucket,
		}
		if input.Ossinput.Object != nil {
			datasource.Spec.OSS.Object = *input.Ossinput.Object
		}
	}

	unstructuredDatasource, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&datasource)
	if err != nil {
		return nil, err
	}
	obj, err := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "datasources"}).
		Namespace(input.Namespace).Create(ctx, &unstructured.Unstructured{Object: unstructuredDatasource}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	// update auth secret with owner reference
	if input.Endpointinput.Auth != nil {
		// user obj as the owner
		err := common.MakeAuthSecret(ctx, c, generated.TypedObjectReferenceInput{
			APIGroup:  &common.CoreV1APIGroup,
			Kind:      "Secret",
			Name:      common.MakeAuthSecretName(datasource.Name, "datasource"),
			Namespace: &input.Namespace,
		}, input.Endpointinput.Auth, obj)
		if err != nil {
			return nil, err
		}
	}
	return datasource2model(obj)
}

func UpdateDatasource(ctx context.Context, c dynamic.Interface, input *generated.UpdateDatasourceInput) (*generated.Datasource, error) {
	obj, err := common.ResouceGet(ctx, c, generated.TypedObjectReferenceInput{
		APIGroup:  &common.ArcadiaAPIGroup,
		Kind:      "Datasource",
		Name:      input.Name,
		Namespace: &input.Namespace,
	}, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	datasource := &v1alpha1.Datasource{}
	if err := utils.UnstructuredToStructured(obj, datasource); err != nil {
		return nil, err
	}

	datasource.SetLabels(graphqlutils.MapAny2Str(input.Labels))
	datasource.SetAnnotations(graphqlutils.MapAny2Str(input.Annotations))

	if input.DisplayName != nil {
		datasource.Spec.DisplayName = *input.DisplayName
	}
	if input.Description != nil {
		datasource.Spec.Description = *input.Description
	}

	// Update endpoint
	if input.Endpointinput != nil {
		endpoint, err := common.MakeEndpoint(ctx, c, generated.TypedObjectReferenceInput{
			APIGroup:  &datasource.APIVersion,
			Kind:      datasource.Kind,
			Name:      datasource.Name,
			Namespace: &datasource.Namespace,
		}, *input.Endpointinput)
		if err != nil {
			return nil, err
		}
		datasource.Spec.Endpoint = endpoint
	}

	// Update ossinput
	if input.Ossinput != nil {
		oss := &v1alpha1.OSS{
			Bucket: input.Ossinput.Bucket,
		}
		if input.Ossinput.Object != nil {
			oss.Object = *input.Ossinput.Object
		}
		datasource.Spec.OSS = oss
	}

	unstructuredDatasource, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&datasource)
	if err != nil {
		return nil, err
	}

	updatedObject, err := common.ResouceUpdate(ctx, c, generated.TypedObjectReferenceInput{
		APIGroup:  &common.ArcadiaAPIGroup,
		Kind:      "Datasource",
		Namespace: &datasource.Namespace,
		Name:      datasource.Name,
	}, unstructuredDatasource, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return datasource2model(updatedObject)
}

func DeleteDatasources(ctx context.Context, c dynamic.Interface, input *generated.DeleteCommonInput) (*string, error) {
	name := ""
	labelSelector, fieldSelector := "", ""
	if input.Name != nil {
		name = *input.Name
	}
	if input.FieldSelector != nil {
		fieldSelector = *input.FieldSelector
	}
	if input.LabelSelector != nil {
		labelSelector = *input.LabelSelector
	}
	resource := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "datasources"})
	if name != "" {
		err := resource.Namespace(input.Namespace).Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil {
			return nil, err
		}
	} else {
		err := resource.Namespace(input.Namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
			LabelSelector: labelSelector,
			FieldSelector: fieldSelector,
		})
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}
func ListDatasources(ctx context.Context, c dynamic.Interface, input generated.ListCommonInput) (*generated.PaginatedResult, error) {
	keyword, labelSelector, fieldSelector := "", "", ""
	page, pageSize := 1, 10
	if input.Keyword != nil {
		keyword = *input.Keyword
	}
	if input.FieldSelector != nil {
		fieldSelector = *input.FieldSelector
	}
	if input.LabelSelector != nil {
		labelSelector = *input.LabelSelector
	}
	if input.Page != nil && *input.Page > 0 {
		page = *input.Page
	}
	if input.PageSize != nil && *input.PageSize > 0 {
		pageSize = *input.PageSize
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: fieldSelector,
	}

	datasList, err := c.Resource(common.SchemaOf(&common.ArcadiaAPIGroup, "Datasource")).Namespace(input.Namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}
	filter := make([]common.ResourceFilter, 0)
	if keyword != "" {
		filter = append(filter, common.FilterDatasourceByKeyword(keyword))
	}
	return common.ListReources(datasList, page, pageSize, datasource2modelConverter, filter...)
}

func ReadDatasource(ctx context.Context, c dynamic.Interface, name, namespace string) (*generated.Datasource, error) {
	u, err := common.ResouceGet(ctx, c, generated.TypedObjectReferenceInput{
		APIGroup:  &common.ArcadiaAPIGroup,
		Kind:      "Datasource",
		Name:      name,
		Namespace: &namespace,
	}, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return datasource2model(u)
}

// CheckDatasource
func CheckDatasource(ctx context.Context, c dynamic.Interface, input generated.CreateDatasourceInput) (*generated.Datasource, error) {
	if input.Ossinput != nil {
		var insecure bool
		if input.Endpointinput.Insecure != nil {
			insecure = !*input.Endpointinput.Insecure
		}
		mc, err := minio.New(input.Endpointinput.URL, &minio.Options{
			Creds:  credentials.NewStaticV4(input.Endpointinput.Auth["rootUser"].(string), input.Endpointinput.Auth["rootPassword"].(string), ""),
			Secure: insecure,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		})
		if err != nil {
			return nil, err
		}
		ok, err := mc.BucketExists(ctx, input.Ossinput.Bucket)
		if err != nil {
			return nil, errors.Wrap(err, "Check bucket")
		}
		if !ok {
			return nil, datasource.ErrOSSNoSuchBucket
		}
		return &generated.Datasource{
			Namespace: input.Namespace,
			Name:      input.Name,
			Status:    &common.StatusTrue,
		}, nil
	}
	return nil, datasource.ErrUnknowDatasourceType
}
