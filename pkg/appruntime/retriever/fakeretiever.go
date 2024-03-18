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
	"fmt"

	langchaingoschema "github.com/tmc/langchaingo/schema"
	"k8s.io/klog/v2"
)

var _ langchaingoschema.Retriever = &Fakeretriever{}

type Fakeretriever struct {
	Name string
	Docs []langchaingoschema.Document
}

func (f *Fakeretriever) GetRelevantDocuments(ctx context.Context, query string) ([]langchaingoschema.Document, error) {
	logger := klog.FromContext(ctx)
	logger.V(3).Info(fmt.Sprintf("GetReleavantDocuments from %s, query: %s", f.Name, query))
	logger.V(5).Info(fmt.Sprintf("Doc: %#v", f.Docs))
	return f.Docs, nil
}
