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
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/apiserver/graph/generated"
)

func filterByKeyWord(keyword string) func(object client.Object) bool {
	return func(u client.Object) bool {
		return strings.Contains(u.GetName(), keyword)
	}
}

func converter(u client.Object) (generated.PageNode, error) {
	return &generated.F{Path: u.GetName()}, nil
}

func initResource() []client.Object {
	// The reverse order should be
	// name3, name7, name1, name5, name4, name2, name6,name9, name8
	timeStrings := []string{
		"2024-01-10T11:57:26Z",
		"2024-01-05T11:57:26Z",
		"2024-01-12T11:57:26Z",
		"2024-01-08T11:57:26Z",
		"2024-01-09T11:57:26Z",
		"2024-01-02T11:57:26Z",
		"2024-01-11T11:57:26Z",
		"2023-01-12T11:57:26Z",
		"2023-02-12T11:57:26Z",
	}
	l := make([]client.Object, len(timeStrings))
	for i, t := range timeStrings {
		tv, _ := time.Parse(time.RFC3339, t)
		l[i] = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              fmt.Sprintf("name%d", i+1),
				CreationTimestamp: metav1.NewTime(tv),
			},
		}
	}
	return l
}

func toF(data []generated.PageNode) []string {
	r := make([]string, 0)
	for _, d := range data {
		r = append(r, d.(*generated.F).Path)
	}
	return r
}

func TestListResources(t *testing.T) {
	type input struct {
		keyword        string
		page, pageSize int
		exp            []string
	}
	for _, tc := range []input{
		{
			keyword:  "abc",
			page:     1,
			pageSize: 1,
			exp:      []string{},
		},
		{
			keyword:  "abc",
			page:     1,
			pageSize: 2,
			exp:      []string{},
		},
		{
			keyword:  "1",
			page:     1,
			pageSize: 1,
			exp:      []string{"name1"},
		},
		{
			keyword:  "1",
			page:     1,
			pageSize: 2,
			exp:      []string{"name1"},
		},
		{
			keyword:  "1",
			page:     2,
			pageSize: 2,
			exp:      []string{},
		},
		{
			keyword:  "name",
			page:     1,
			pageSize: 2,
			exp:      []string{"name3", "name7"},
		},
		{
			keyword:  "name",
			page:     2,
			pageSize: 2,
			exp:      []string{"name1", "name5"},
		},
		{
			keyword:  "name",
			page:     3,
			pageSize: 2,
			exp:      []string{"name4", "name2"},
		},
		{
			keyword:  "name",
			page:     4,
			pageSize: 2,
			exp:      []string{"name6", "name9"},
		},
		{
			keyword:  "name",
			page:     5,
			pageSize: 2,
			exp:      []string{"name8"},
		},
		{
			keyword:  "name",
			page:     6,
			pageSize: 2,
			exp:      []string{},
		},
		{
			keyword:  "name",
			page:     1,
			pageSize: 9,
			exp:      []string{"name3", "name7", "name1", "name5", "name4", "name2", "name6", "name9", "name8"},
		},
		{
			keyword:  "name1",
			page:     1,
			pageSize: 9,
			exp:      []string{"name1"},
		},
		{
			keyword:  "name",
			page:     1,
			pageSize: 8,
			exp:      []string{"name3", "name7", "name1", "name5", "name4", "name2", "name6", "name9"},
		},
		{
			keyword:  "name",
			page:     2,
			pageSize: 8,
			exp:      []string{"name8"},
		},
		{
			keyword:  "name",
			page:     2,
			pageSize: -1,
			exp:      []string{"name3", "name7", "name1", "name5", "name4", "name2", "name6", "name9", "name8"},
		},
		{
			keyword:  "name",
			page:     3,
			pageSize: 8,
			exp:      []string{},
		},
	} {
		// name3, name7, name1, name5, name4, name2, name6,name9, name8
		data := initResource()
		r, _ := ListReources(data, tc.page, tc.pageSize, converter, filterByKeyWord(tc.keyword))
		r1 := toF(r.Nodes)
		if !reflect.DeepEqual(tc.exp, r1) {
			t.Fatalf("with keyword %s, page: %d, pageSize: %d , expect %v get %v", tc.keyword, tc.page, tc.pageSize, tc.exp, r1)
		}
	}
}
