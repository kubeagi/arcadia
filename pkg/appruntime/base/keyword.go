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

package base

import (
	"errors"

	langchainschema "github.com/tmc/langchaingo/schema"
)

const (
	InputQuestionKeyInArg                 = "question"
	InputIsNeedStreamKeyInArg             = "_need_stream"
	LangchaingoChatMessageHistoryKeyInArg = "_history"
	OutputAnswerKeyInArg                  = "_answer"
	AgentOutputInArg                      = "_agent_answer"
	MapReduceDocumentOutputInArg          = "_mapreduce_document_answer"
	OutputAnswerStreamChanKeyInArg        = "_answer_stream"
	RuntimeRetrieverReferencesKeyInArg    = "_references"
	LangchaingoRetrieversKeyInArg         = "retrievers"
	LangchaingoLLMKeyInArg                = "llm"
	LangchaingoPromptKeyInArg             = "prompt"
	APPDocNullReturn                      = "_app_doc_null_return"
	ConversationKnowledgeBaseInArg        = "_conversation_knowledgebase" // the conversation Knowledgebase cr in args, status has ready
	ConversationIDInArg                   = "_conversation_id"
)

var (
	ErrNoQuestion   = errors.New("no question in args")
	ErrNoRetrievers = errors.New("no retrievers in args")
)

func GetInputQuestionFromArg(args map[string]any) (string, error) {
	q, ok := args[InputQuestionKeyInArg]
	if !ok {
		return "", ErrNoQuestion
	}
	query, ok := q.(string)
	if !ok || len(query) == 0 {
		return "", errors.New("empty question")
	}
	return query, nil
}

func GetRetrieversFromArg(args map[string]any) ([]langchainschema.Retriever, error) {
	v, ok := args[LangchaingoRetrieversKeyInArg]
	if !ok {
		return nil, ErrNoRetrievers
	}
	retrievers, ok := v.([]langchainschema.Retriever)
	if !ok {
		return nil, errors.New("retrievers not []schema.Retriever")
	}
	return retrievers, nil
}

func GetAPPDocNullReturnFromArg(args map[string]any) (string, error) {
	v, ok := args[APPDocNullReturn]
	if !ok {
		return "", nil
	}
	docNullReturn, ok := v.(string)
	if !ok {
		return "", errors.New("app doc null return not string type")
	}
	return docNullReturn, nil
}

// AddKnowledgebaseRetrieverToArg add knowledgebase retriever to args
// Note: only knowledgebase retriever will be appended, other components like qachain will only use the first retriever in args
func AddKnowledgebaseRetrieverToArg(args map[string]any, retriever langchainschema.Retriever) map[string]any {
	if _, exist := args[LangchaingoRetrieversKeyInArg]; !exist {
		args[LangchaingoRetrieversKeyInArg] = make([]langchainschema.Retriever, 0)
	}
	retrievers := args[LangchaingoRetrieversKeyInArg].([]langchainschema.Retriever)
	retrievers = append(retrievers, retriever)
	args[LangchaingoRetrieversKeyInArg] = retrievers
	return args
}
