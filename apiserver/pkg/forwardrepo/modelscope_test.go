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
	"testing"
)

const (
	modelID = "qwen/Qwen1.5-0.5B"
)

var (
	Revisions = Revision{
		Branches: []BranchTag{
			{Name: "master"},
		},
	}
)

func TestModelScope(t *testing.T) {
	m := NewModelScope(WithModelID(modelID))
	ctx := context.Background()
	_, err := m.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = m.Revisions(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
