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

package retriever

import (
	"context"

	langchaingoschema "github.com/tmc/langchaingo/schema"
)

var _ langchaingoschema.Retriever = &Fakeretriever{}

type Fakeretriever struct {
	Docs []langchaingoschema.Document
}

func (f *Fakeretriever) GetRelevantDocuments(context.Context, string) ([]langchaingoschema.Document, error) {
	return f.Docs, nil
}
