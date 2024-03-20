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
package forwardrepo

import (
	"context"
	"net/http"
	"reflect"
	"testing"
)

const (
	expReadme = `---
license: apache-2.0
---

123


do


### xyz`
)

var (
	expFiles    = []string{".gitattributes", "README.md"}
	expRevision = Revision{
		Tags:     []BranchTag{{Name: "tr", TargetCommit: "84ecd934b88eac3389265c93bc4c1d5712a19817"}},
		Branches: []BranchTag{{Name: "main", TargetCommit: "d78a3bd719100068671996f543c1879f4e39381d"}, {Name: "cdc", TargetCommit: "84ecd934b88eac3389265c93bc4c1d5712a19817"}},
	}
)

func TestHuggingFace(t *testing.T) {
	modelID := "uint64/abc"
	tp := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}

	ctx := context.Background()
	h := NewHuggingFace(WithTransport(tp), WithModelID(modelID))
	r, err := h.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if r != expReadme {
		t.Fatalf("expect %s get %s", expReadme, r)
	}

	r1, err := h.Files(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expFiles, r1) {
		t.Fatalf("expect %v get %v", expFiles, r1)
	}

	r2, err := h.Revisions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r2, expRevision) {
		t.Fatalf("expect %v get %v", expReadme, r2)
	}
}
