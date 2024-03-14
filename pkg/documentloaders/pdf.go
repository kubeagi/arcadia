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

package documentloaders

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"code.sajari.com/docconv/v2"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
)

type PDF struct {
	r io.Reader
}

func NewPDF(r io.Reader) *PDF {
	return &PDF{r: r}
}

func (p *PDF) Load(ctx context.Context) ([]schema.Document, error) {
	str, meta, err := docconv.ConvertPDF(p.r)
	if err != nil {
		return nil, err
	}
	/*
		map[Author:tutu CreatedDate:1706601645 CreationDate:Tue Jan 30 08:00:45 2024 UTC Creator:WPS 文字
		Custom Metadata:yes Encrypted:no File size:20637614 bytes Form:none JavaScript:no Keywords: Metadata Stream:no ModDate:Tue Jan 30 08:00:45 2024 UTC ModifiedDate:1706601645
		Optimized:no PDF version:1.7 Page rot:0
		Page size:783.85 x 841.9 pts Pages:351 Producer: Subject: Suspects:no Tagged:no Title:需求规格及概要设计 UserProperties:no]
	*/
	docs := make([]schema.Document, 0)
	from := 0
	pages, _ := strconv.Atoi(meta["Pages"])
	for page := 1; page <= pages; page++ {
		key := fmt.Sprintf("Page %d of %d", page, pages)
		idx := strings.Index(str, key)
		docs = append(docs, schema.Document{
			PageContent: str[from:idx],
			Metadata: map[string]any{
				"page":        page,
				"total_pages": pages,
			},
		})
		from = idx + len(key) + 1
	}
	return docs, nil
}

func (p *PDF) LoadAndSplit(ctx context.Context, splitter textsplitter.TextSplitter) ([]schema.Document, error) {
	docs, err := p.Load(ctx)
	if err != nil {
		return nil, err
	}
	return textsplitter.SplitDocuments(splitter, docs)
}
