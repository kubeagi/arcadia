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

type ModelScopeSummaryData struct {
	Name          string `json:"Name"`
	ReadMeContent string `json:"ReadMeContent"`
}
type ModelScopeSummary struct {
	Code int `json:"Code"`

	Data ModelScopeSummaryData `json:"Data"`

	Message string `json:"Message"`
	Success bool   `json:"Success"`
}

type EachRevision struct {
	Revision string `json:"Revision"`
}

type ModelScopeRevisionData struct {
	RevisionMap RevisionMap `json:"RevisionMap"`
}

type RevisionMap struct {
	Branches []EachRevision `json:"Branches"`
	Tags     []EachRevision `json:"Tags"`
}
type ModelScopeRevision struct {
	Code int `json:"Code"`

	Data ModelScopeRevisionData `json:"Data"`

	Message string `json:"Message"`
	Success bool   `json:"Success"`
}
