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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	langchaingoschema "github.com/tmc/langchaingo/schema"
	"k8s.io/klog/v2"

	"github.com/kubeagi/arcadia/pkg/appruntime/base"
	"github.com/kubeagi/arcadia/pkg/documentloaders"
)

type Reference struct {
	// Question row
	Question string `json:"question" example:"q: 旷工最小计算单位为多少天？"`
	// Answer row
	Answer string `json:"answer" example:"旷工最小计算单位为 0.5 天。"`
	// vector search score
	Score float32 `json:"score" example:"0.34"`
	// the qa file fullpath
	QAFilePath string `json:"qa_file_path" example:"dataset/dataset-playground/v1/qa.csv"`
	// line number in the qa file
	QALineNumber int `json:"qa_line_number" example:"7"`
	// source file name, only file name, not full path
	FileName string `json:"file_name" example:"员工考勤管理制度-2023.pdf"`
	// page number in the source file
	PageNumber int `json:"page_number" example:"1"`
	// related content in the source file or in webpage
	Content string `json:"content" example:"旷工最小计算单位为0.5天，不足0.5天以0.5天计算，超过0.5天不满1天以1天计算，以此类推。"`
	// Title of the webpage
	Title string `json:"title,omitempty" example:"开始使用 Microsoft 帐户 – Microsoft"`
	// URL of the webpage
	URL string `json:"url,omitempty" example:"https://www.microsoft.com/zh-cn/welcome"`
	// RerankScore
	RerankScore float32        `json:"rerank_score,omitempty" example:"0.58124"`
	Metadata    map[string]any `json:"-"`
}

const RerankScoreCol string = "rerank_score"

func (reference Reference) String() string {
	bytes, err := json.Marshal(&reference)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func (reference Reference) SimpleString() string {
	return fmt.Sprintf("%s %s", reference.Question, reference.Answer)
}

func AddReferencesToArgs(args map[string]any, refs []Reference) map[string]any {
	old, exist := args[base.RuntimeRetrieverReferencesKeyInArg]
	if exist {
		oldRefs := old.([]Reference)
		args[base.RuntimeRetrieverReferencesKeyInArg] = append(oldRefs, refs...)
		return args
	}
	args[base.RuntimeRetrieverReferencesKeyInArg] = refs
	return args
}

// ConvertDocuments, convert raw doc to what we want doc, for example, knowledgebase doc should add answer into page content
func ConvertDocuments(ctx context.Context, docs []langchaingoschema.Document, retrieverName string) (newDocs []langchaingoschema.Document, refs []Reference) {
	logger := klog.FromContext(ctx)
	docLen := len(docs)
	logger.V(3).Info(fmt.Sprintf("get data from retriever: %s, total numbers: %d\n", retrieverName, docLen))
	refs = make([]Reference, 0, docLen)
	for k, doc := range docs {
		logger.V(3).Info(fmt.Sprintf("related doc[%d] raw text: %s, raw score: %f\n", k, doc.PageContent, doc.Score))
		for key, v := range doc.Metadata {
			if str, ok := v.([]byte); ok {
				logger.V(3).Info(fmt.Sprintf("related doc[%d] metadata[%s]: %s\n", k, key, string(str)))
			} else {
				logger.V(3).Info(fmt.Sprintf("related doc[%d] metadata[%s]: %#v\n", k, key, v))
			}
		}
		// chroma will get []byte, pgvector will get string...
		answer, ok := doc.Metadata[documentloaders.AnswerCol].(string)
		if !ok {
			if a, ok := doc.Metadata[documentloaders.AnswerCol].([]byte); ok {
				answer = strings.TrimPrefix(strings.TrimSuffix(string(a), "\""), "\"")
			}
		}
		pageContent := doc.PageContent
		joinStr := "\na: "
		if retrieverName == "knowledgebase" {
			// qachain will only use doc.PageContent in prompt, so add ansewer here if exist
			if len(answer) != 0 {
				doc.PageContent = doc.PageContent + joinStr + answer
			}
		}
		if retrieverName == "multiquery" || retrieverName == "retrievalqachain" {
			// pageContent may have the answer in previous steps, and we want this field only has question in reference output
			pageContent = strings.TrimSuffix(pageContent, joinStr+answer)
		}

		qafilepath, ok := doc.Metadata[documentloaders.QAFileName].(string)
		if !ok {
			if a, ok := doc.Metadata[documentloaders.QAFileName].([]byte); ok {
				qafilepath = strings.TrimPrefix(strings.TrimSuffix(string(a), "\""), "\"")
			}
		}
		lineNumber, ok := doc.Metadata[documentloaders.LineNumber].(string)
		if !ok {
			if a, ok := doc.Metadata[documentloaders.LineNumber].([]byte); ok {
				lineNumber = strings.TrimPrefix(strings.TrimSuffix(string(a), "\""), "\"")
			}
		}
		line, _ := strconv.Atoi(lineNumber)
		filename, ok := doc.Metadata[documentloaders.FileNameCol].(string)
		if !ok {
			if a, ok := doc.Metadata[documentloaders.FileNameCol].([]byte); ok {
				filename = strings.TrimPrefix(strings.TrimSuffix(string(a), "\""), "\"")
			}
		}
		pageNumber, ok := doc.Metadata[documentloaders.PageNumberCol].(string)
		if !ok {
			if a, ok := doc.Metadata[documentloaders.PageNumberCol].([]byte); ok {
				pageNumber = strings.TrimPrefix(strings.TrimSuffix(string(a), "\""), "\"")
			}
		}
		page, _ := strconv.Atoi(pageNumber)
		content, ok := doc.Metadata[documentloaders.ChunkContentCol].(string)
		if !ok {
			if a, ok := doc.Metadata[documentloaders.ChunkContentCol].([]byte); ok {
				content = strings.TrimPrefix(strings.TrimSuffix(string(a), "\""), "\"")
			}
		}
		rerankScore, _ := doc.Metadata[RerankScoreCol].(float32)
		refs = append(refs, Reference{
			Question:     pageContent,
			Answer:       answer,
			Score:        doc.Score,
			QAFilePath:   qafilepath,
			QALineNumber: line,
			FileName:     filename,
			PageNumber:   page,
			Content:      content,
			Metadata:     doc.Metadata,
			RerankScore:  rerankScore,
		})
		docs[k] = doc
	}
	return docs, refs
}
