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

package retriever

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/chains"
	langchaingoschema "github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"

	apiretriever "github.com/kubeagi/arcadia/api/app-node/retriever/v1alpha1"
	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/appruntime/base"
	"github.com/kubeagi/arcadia/pkg/documentloaders"
	"github.com/kubeagi/arcadia/pkg/langchainwrap"
	pkgvectorstore "github.com/kubeagi/arcadia/pkg/vectorstore"
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
}

func (reference Reference) String() string {
	bytes, err := json.Marshal(&reference)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func AddReferencesToArgs(args map[string]any, refs []Reference) map[string]any {
	if len(refs) == 0 {
		return args
	}
	old, exist := args["_references"]
	if exist {
		oldRefs := old.([]Reference)
		args["_references"] = append(oldRefs, refs...)
		return args
	}
	args["_references"] = refs
	return args
}

type KnowledgeBaseRetriever struct {
	langchaingoschema.Retriever
	base.BaseNode
	DocNullReturn string
}

func NewKnowledgeBaseRetriever(baseNode base.BaseNode) *KnowledgeBaseRetriever {
	return &KnowledgeBaseRetriever{
		nil,
		baseNode,
		"",
	}
}

func (l *KnowledgeBaseRetriever) Run(ctx context.Context, cli dynamic.Interface, args map[string]any) (map[string]any, error) {
	ns := base.GetAppNamespace(ctx)
	instance := &apiretriever.KnowledgeBaseRetriever{}
	obj, err := cli.Resource(schema.GroupVersionResource{Group: apiretriever.GroupVersion.Group, Version: apiretriever.GroupVersion.Version, Resource: "knowledgebaseretrievers"}).
		Namespace(l.BaseNode.Ref.GetNamespace(ns)).Get(ctx, l.BaseNode.Ref.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("can't find the retriever in cluster: %w", err)
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), instance)
	if err != nil {
		return nil, fmt.Errorf("can't convert the retriever in cluster: %w", err)
	}
	l.DocNullReturn = instance.Spec.DocNullReturn

	var knowledgebaseName, knowledgebaseNamespace string
	for _, n := range l.BaseNode.GetPrevNode() {
		if n.Kind() == "knowledgebase" {
			knowledgebaseName = n.RefName()
			knowledgebaseNamespace = n.RefNamespace()
			break
		}
	}
	if knowledgebaseName == "" || knowledgebaseNamespace == "" {
		return nil, fmt.Errorf("knowledgebase is not setting")
	}

	knowledgebase := &v1alpha1.KnowledgeBase{}
	obj, err = cli.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "knowledgebases"}).
		Namespace(knowledgebaseNamespace).Get(ctx, knowledgebaseName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("can't find the knowledgebase in cluster: %w", err)
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), knowledgebase)
	if err != nil {
		return nil, fmt.Errorf("can't convert the knowledgebase in cluster: %w", err)
	}

	embedderReq := knowledgebase.Spec.Embedder
	vectorStoreReq := knowledgebase.Spec.VectorStore
	if embedderReq == nil || vectorStoreReq == nil {
		return nil, fmt.Errorf("knowledgebase %s: embedder or vectorstore or filegroups is not setting", knowledgebaseName)
	}

	embedder := &v1alpha1.Embedder{}
	obj, err = cli.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "embedders"}).
		Namespace(embedderReq.GetNamespace(ns)).Get(ctx, embedderReq.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("can't find the embedder in cluster: %w", err)
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), embedder)
	if err != nil {
		return nil, fmt.Errorf("can't convert the embedder in cluster: %w", err)
	}
	em, err := langchainwrap.GetLangchainEmbedder(ctx, embedder, nil, cli)
	if err != nil {
		return nil, fmt.Errorf("can't convert to langchain embedder: %w", err)
	}
	vectorStore := &v1alpha1.VectorStore{}
	obj, err = cli.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "vectorstores"}).
		Namespace(vectorStoreReq.GetNamespace(ns)).Get(ctx, vectorStoreReq.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("can't find the vectorstore in cluster: %w", err)
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), vectorStore)
	if err != nil {
		return nil, fmt.Errorf("can't convert the vectorstore in cluster: %w", err)
	}
	var s vectorstores.VectorStore
	s, _, err = pkgvectorstore.NewVectorStore(ctx, vectorStore, em, knowledgebase.VectorStoreCollectionName(), nil, cli)
	if err != nil {
		return nil, err
	}
	l.Retriever = vectorstores.ToRetriever(s, instance.Spec.NumDocuments, vectorstores.WithScoreThreshold(instance.Spec.ScoreThreshold))
	args["retriever"] = l
	return args, nil
}

// KnowledgeBaseStuffDocuments is similar to chains.StuffDocuments but with new joinDocuments method
type KnowledgeBaseStuffDocuments struct {
	chains.StuffDocuments
	isDocNullReturn bool
	DocNullReturn   string
	callbacks.SimpleHandler
	References []Reference
}

func (c *KnowledgeBaseStuffDocuments) GetCallbackHandler() callbacks.Handler {
	return c
}

var (
	_ chains.Chain           = &KnowledgeBaseStuffDocuments{}
	_ callbacks.Handler      = &KnowledgeBaseStuffDocuments{}
	_ callbacks.HandlerHaver = &KnowledgeBaseStuffDocuments{}
)

func (c *KnowledgeBaseStuffDocuments) joinDocuments(ctx context.Context, docs []langchaingoschema.Document) string {
	logger := klog.FromContext(ctx)
	var text string
	docLen := len(docs)
	for k, doc := range docs {
		logger.V(3).Info(fmt.Sprintf("KnowledgeBaseRetriever: related doc[%d] raw text: %s, raw score: %f\n", k, doc.PageContent, doc.Score))
		for key, v := range doc.Metadata {
			if str, ok := v.([]byte); ok {
				logger.V(3).Info(fmt.Sprintf("KnowledgeBaseRetriever: related doc[%d] metadata[%s]: %s\n", k, key, string(str)))
			} else {
				logger.V(3).Info(fmt.Sprintf("KnowledgeBaseRetriever: related doc[%d] metadata[%s]: %#v\n", k, key, v))
			}
		}
		// chroma will get []byte, pgvector will get string...
		answer, ok := doc.Metadata[documentloaders.AnswerCol].(string)
		if !ok {
			if a, ok := doc.Metadata[documentloaders.AnswerCol].([]byte); ok {
				answer = strings.TrimPrefix(strings.TrimSuffix(string(a), "\""), "\"")
			}
		}

		text += doc.PageContent
		if len(answer) != 0 {
			text = text + "\na: " + answer
		}
		if k != docLen-1 {
			text += c.Separator
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
		c.References = append(c.References, Reference{
			Question:     doc.PageContent,
			Answer:       answer,
			Score:        1 - doc.Score, // for pgvector
			QAFilePath:   qafilepath,
			QALineNumber: line,
			FileName:     filename,
			PageNumber:   page,
			Content:      content,
		})
	}
	logger.V(3).Info(fmt.Sprintf("KnowledgeBaseRetriever: finally get related text: %s\n", text))
	if len(text) == 0 {
		c.isDocNullReturn = true
	}
	return text
}

func NewStuffDocuments(llmChain *chains.LLMChain, docNullReturn string) *KnowledgeBaseStuffDocuments {
	return &KnowledgeBaseStuffDocuments{
		StuffDocuments: chains.NewStuffDocuments(llmChain),
		DocNullReturn:  docNullReturn,
		References:     make([]Reference, 0, 5),
	}
}

func (c *KnowledgeBaseStuffDocuments) Call(ctx context.Context, values map[string]any, options ...chains.ChainCallOption) (map[string]any, error) {
	docs, ok := values[c.InputKey].([]langchaingoschema.Document)
	if !ok {
		return nil, fmt.Errorf("%w: %w", chains.ErrInvalidInputValues, chains.ErrInputValuesWrongType)
	}

	inputValues := make(map[string]any)
	for key, value := range values {
		inputValues[key] = value
	}

	inputValues[c.DocumentVariableName] = c.joinDocuments(ctx, docs)
	return chains.Call(ctx, c.LLMChain, inputValues, options...)
}

func (c KnowledgeBaseStuffDocuments) GetMemory() langchaingoschema.Memory {
	return c.StuffDocuments.GetMemory()
}

func (c KnowledgeBaseStuffDocuments) GetInputKeys() []string {
	return c.StuffDocuments.GetInputKeys()
}

func (c KnowledgeBaseStuffDocuments) GetOutputKeys() []string {
	return c.StuffDocuments.GetOutputKeys()
}

func (c KnowledgeBaseStuffDocuments) HandleChainEnd(ctx context.Context, outputValues map[string]any) {
	if !c.isDocNullReturn {
		return
	}
	klog.FromContext(ctx).Info(fmt.Sprintf("raw llmChain output: %s, but there is no doc return, so set output to %s\n", outputValues[c.LLMChain.OutputKey], c.DocNullReturn))
	outputValues[c.LLMChain.OutputKey] = c.DocNullReturn
}
