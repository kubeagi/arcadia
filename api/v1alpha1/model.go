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

package v1alpha1

import (
	"fmt"
)

const (
	// LabelModelTypes keeps the spec.types field
	LabelModelTypes    = Group + "/model-types"
	LabelModelFullPath = Group + "/full-path"
)

func (model Model) ModelTypes() string {
	if model.Spec.Types == "" {
		return "unknown"
	}
	return model.Spec.Types
}

// FullPath with bucket and object path
func (model Model) FullPath() string {
	return fmt.Sprintf("%s/%s", model.Namespace, model.ObjectPath())
}

// ObjectPath is the path where model stored at in a bucket
func (model Model) ObjectPath() string {
	return fmt.Sprintf("model/%s/", model.Name)
}
