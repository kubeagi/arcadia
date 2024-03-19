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

type ModelTagBranch struct {
	Name         string `json:"name"`
	Ref          string `json:"ref"`
	TargetCommit string `json:"targetCommit"`
}

type ModelRevision struct {
	Tags     []ModelTagBranch `json:"tags"`
	Branches []ModelTagBranch `json:"branches"`
}

type Sibling struct {
	RFileName string `json:"rfilename"`
}
type Model struct {
	ID       string    `json:"id"`
	ModelID  string    `json:"modelId"`
	Author   string    `json:"author"`
	SHA      string    `json:"sha"`
	Siblings []Sibling `json:"siblings"`
}
