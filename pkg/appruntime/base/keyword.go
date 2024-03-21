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

const (
	InputQuestionKeyInArg                 = "question"
	InputIsNeedStreamKeyInArg             = "_need_stream"
	LangchaingoChatMessageHistoryKeyInArg = "_history"
	OutputAnserKeyInArg                   = "_answer"
	AgentOutputInArg                      = "_agent_answer"
	MapReduceDocumentOutputInArg          = "_mapreduce_document_answer"
	OutputAnserStreamChanKeyInArg         = "_answer_stream"
	RuntimeRetrieverReferencesKeyInArg    = "_references"
	LangchaingoRetrieverKeyInArg          = "retriever"
	LangchaingoLLMKeyInArg                = "llm"
	LangchaingoPromptKeyInArg             = "prompt"
	APPDocNullReturn                      = "_app_doc_null_return"
	ConversationKnowledgeBaseInArg        = "_conversation_knowledgebase" // the conversation Knowledgebase cr in args, status has ready
)
