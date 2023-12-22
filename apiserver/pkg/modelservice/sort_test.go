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
	"testing"
	"time"

	"github.com/kubeagi/arcadia/apiserver/graph/generated"
)

type input struct {
	start, end int
	exp        []string
}

func initData() []*generated.ModelService {
	seconds := []int{
		0, -2, 4, -6, 8, -10, 12, -14, 16, -18, 20, -22, 24, 26, 28, 30,
	}
	// sorted order: 30,28,26,24,33, 66, 5, 4, 7, 3, 1, 2, 6, 10, 55, 88
	source := []*generated.ModelService{
		{Name: "3"}, {Name: "1"}, {Name: "7"},
		{Name: "2"}, {Name: "4"}, {Name: "6"},
		{Name: "5"}, {Name: "10"}, {Name: "66"},
		{Name: "55"}, {Name: "33"}, {Name: "88"},
		{Name: "24"}, {Name: "26"}, {Name: "28"},
		{Name: "30"},
	}
	now := time.Now()
	for idx := range source {
		tmp := now.Add(time.Second * time.Duration(seconds[idx]))
		source[idx].CreationTimestamp = &tmp
	}
	return source
}

func TestPagedModelService(t *testing.T) {
	for _, tc := range []input{
		{0, 3, []string{"30", "28", "26"}},
		{3, 6, []string{"24", "33", "66"}},
		{6, 9, []string{"5", "4", "7"}},
		{9, 12, []string{"3", "1", "2"}},
		{12, 15, []string{"6", "10", "55"}},
		{15, 18, []string{"88"}},
		{0, 7, []string{"30", "28", "26", "24", "33", "66", "5"}},
		{7, 14, []string{"4", "7", "3", "1", "2", "6", "10"}},
		{14, 18, []string{"55", "88"}},
		{0, 10, []string{"30", "28", "26", "24", "33", "66", "5", "4", "7", "3"}},
		{10, 20, []string{"1", "2", "6", "10", "55", "88"}},
	} {
		source := initData()
		r := pageModelService(tc.start, tc.end, &source)
		if len(tc.exp) != len(r) {
			t.Fatalf("expect %d items, got %d", len(tc.exp), len(r))
		}
		for idx := range r {
			if r[idx].Name != tc.exp[idx] {
				t.Fatalf("[%d - %d] expects the i-th to be %s but actually is %s", tc.start, tc.end, tc.exp[idx], r[idx].Name)
			}
		}
	}
}
