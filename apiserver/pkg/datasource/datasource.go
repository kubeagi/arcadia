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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/graph/generated"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	graphqlutils "github.com/kubeagi/arcadia/apiserver/pkg/utils"
	"github.com/kubeagi/arcadia/pkg/datasource"
)

func datasource2modelConverter(obj client.Object) (generated.PageNode, error) {
	ds, ok := obj.(*v1alpha1.Datasource)
	if !ok {
		return nil, errors.New("can't convert object to Datasource")
	}
	return datasource2model(ds)
}

func datasource2model(datasource *v1alpha1.Datasource) (*generated.Datasource, error) {
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
	web := generated.Web{}
	if datasource.Spec.Web != nil {
		web.RecommendIntervalTime = &datasource.Spec.Web.RecommendIntervalTime
	}

	md := generated.Datasource{
		ID:                &id,
		Name:              datasource.Name,
		Namespace:         datasource.Namespace,
		Creator:           pointer.String(datasource.Spec.Creator),
		Labels:            graphqlutils.MapStr2Any(datasource.GetLabels()),
		Annotations:       graphqlutils.MapStr2Any(datasource.GetAnnotations()),
		DisplayName:       &datasource.Spec.DisplayName,
		Description:       &datasource.Spec.Description,
		Endpoint:          &endpoint,
		Type:              string(datasource.Spec.Type()),
		Oss:               &oss,
		Web:               &web,
		Status:            &status,
		Message:           &message,
		CreationTimestamp: &creationtimestamp,
		UpdateTimestamp:   &updateTime,
	}
	return &md, nil
}

func CreateDatasource(ctx context.Context, c client.Client, input generated.CreateDatasourceInput) (*generated.Datasource, error) {
	// create datasource
	datasource := &v1alpha1.Datasource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      input.Name,
			Namespace: input.Namespace,
		},
	}
	common.SetCreator(ctx, &datasource.Spec.CommonSpec)
	datasource.Spec.DisplayName = pointer.StringDeref(input.DisplayName, datasource.Spec.DisplayName)
	datasource.Spec.Description = pointer.StringDeref(input.Description, datasource.Spec.Description)

	// make endpoint
	endpoint, err := common.MakeEndpoint(ctx, c, datasource, input.Endpointinput)
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

	if input.Webinput != nil {
		datasource.Spec.Web = &v1alpha1.Web{
			RecommendIntervalTime: input.Webinput.RecommendIntervalTime,
		}
	}

	err = c.Create(ctx, datasource)
	if err != nil {
		return nil, err
	}

	// update auth secret with owner reference
	if input.Endpointinput.Auth != nil {
		// user obj as the owner
		err := common.MakeAuthSecret(ctx, c, input.Namespace, common.GenerateAuthSecretName(datasource.Name, "datasource"), input.Endpointinput.Auth, datasource)
		if err != nil {
			return nil, err
		}
	}
	return datasource2model(datasource)
}

func UpdateDatasource(ctx context.Context, c client.Client, input *generated.UpdateDatasourceInput) (*generated.Datasource, error) {
	datasource := &v1alpha1.Datasource{}
	err := c.Get(ctx, types.NamespacedName{Namespace: input.Namespace, Name: input.Name}, datasource)
	if err != nil {
		return nil, err
	}

	datasource.SetLabels(graphqlutils.MapAny2Str(input.Labels))
	datasource.SetAnnotations(graphqlutils.MapAny2Str(input.Annotations))
	datasource.Spec.DisplayName = pointer.StringDeref(input.DisplayName, datasource.Spec.DisplayName)
	datasource.Spec.Description = pointer.StringDeref(input.Description, datasource.Spec.Description)

	// Update endpoint
	if input.Endpointinput != nil {
		endpoint, err := common.MakeEndpoint(ctx, c, datasource, *input.Endpointinput)
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

	// Update webinput
	if input.Webinput != nil {
		datasource.Spec.Web = &v1alpha1.Web{
			RecommendIntervalTime: input.Webinput.RecommendIntervalTime,
		}
	}

	err = c.Update(ctx, datasource)
	if err != nil {
		return nil, err
	}
	return datasource2model(datasource)
}

func DeleteDatasources(ctx context.Context, c client.Client, input *generated.DeleteCommonInput) (*string, error) {
	opts, err := common.DeleteAllOptions(input)
	if err != nil {
		return nil, err
	}
	err = c.DeleteAllOf(ctx, &v1alpha1.Datasource{}, opts...)
	return nil, err
}

func ListDatasources(ctx context.Context, c client.Client, input generated.ListCommonInput) (*generated.PaginatedResult, error) {
	keyword := ""
	page, pageSize := 1, 10
	if input.Keyword != nil {
		keyword = *input.Keyword
	}
	if input.Page != nil && *input.Page > 0 {
		page = *input.Page
	}
	if input.PageSize != nil && *input.PageSize > 0 {
		pageSize = *input.PageSize
	}

	opts, err := common.NewListOptions(input)
	if err != nil {
		return nil, err
	}
	datasList := &v1alpha1.DatasourceList{}
	err = c.List(ctx, datasList, opts...)
	if err != nil {
		return nil, err
	}

	filter := make([]common.ResourceFilter, 0)
	if keyword != "" {
		filter = append(filter, common.FilterDatasourceByKeyword(keyword))
	}
	items := make([]client.Object, len(datasList.Items))
	for i := range datasList.Items {
		items[i] = &datasList.Items[i]
	}
	return common.ListReources(items, page, pageSize, datasource2modelConverter, filter...)
}

func ReadDatasource(ctx context.Context, c client.Client, name, namespace string) (*generated.Datasource, error) {
	u := &v1alpha1.Datasource{}
	err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, u)
	if err != nil {
		return nil, err
	}
	return datasource2model(u)
}

// CheckDatasource
func CheckDatasource(ctx context.Context, _ client.Client, input generated.CreateDatasourceInput) (*generated.Datasource, error) {
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
