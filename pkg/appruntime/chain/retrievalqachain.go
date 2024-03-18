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

package chain

import (
	"context"
	"errors"
	"fmt"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
	langchainschema "github.com/tmc/langchaingo/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/app-node/chain/v1alpha1"
	apiretriever "github.com/kubeagi/arcadia/api/app-node/retriever/v1alpha1"
	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/appruntime/base"
	"github.com/kubeagi/arcadia/pkg/appruntime/log"
	appruntimeretriever "github.com/kubeagi/arcadia/pkg/appruntime/retriever"
)

type RetrievalQAChain struct {
	chains.ConversationalRetrievalQA
	base.BaseNode
	Instance *v1alpha1.RetrievalQAChain
}

func NewRetrievalQAChain(baseNode base.BaseNode) *RetrievalQAChain {
	return &RetrievalQAChain{
		ConversationalRetrievalQA: chains.ConversationalRetrievalQA{},
		BaseNode:                  baseNode,
	}
}

func (l *RetrievalQAChain) Init(ctx context.Context, cli client.Client, _ map[string]any) error {
	instance := &v1alpha1.RetrievalQAChain{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: l.RefNamespace(), Name: l.Ref.Name}, instance); err != nil {
		return fmt.Errorf("can't find the chain in cluster: %w", err)
	}
	l.Instance = instance
	return nil
}

func (l *RetrievalQAChain) Run(ctx context.Context, cli client.Client, args map[string]any) (outArgs map[string]any, err error) {
	v1, ok := args[base.LangchaingoLLMKeyInArg]
	if !ok {
		return args, errors.New("no llm")
	}
	llm, ok := v1.(llms.Model)
	if !ok {
		return args, errors.New("llm not llms.Model")
	}
	v2, ok := args["prompt"]
	if !ok {
		return args, errors.New("no prompt")
	}
	prompt, ok := v2.(prompts.FormatPrompter)
	if !ok {
		return args, errors.New("prompt not prompts.FormatPrompter")
	}
	vkb, ok := args[base.ConversationKnowledgeBaseInArg]
	if ok {
		kb, ok := vkb.(*arcadiav1alpha1.KnowledgeBase)
		if !ok {
			return args, errors.New("knowledgebase not arcadiav1alpha1.KnowledgeBase")
		}
		args, finish, err := appruntimeretriever.GenerateKnowledgebaseRetriever(ctx, cli, kb.Name, kb.Namespace, apiretriever.CommonRetrieverConfig{ScoreThreshold: pointer.Float32(apiretriever.DefaultScoreThreshold), NumDocuments: 20}, args)
		if err != nil {
			return args, err
		}
		defer finish()
	}

	v3, ok := args["retriever"]
	if !ok {
		return args, errors.New("no retriever")
	}
	retriever, ok := v3.(langchainschema.Retriever)
	if !ok {
		return args, errors.New("retriever not schema.Retriever")
	}

	v4, ok := args[base.LangchaingoChatMessageHistoryKeyInArg]
	if !ok {
		return args, errors.New("no history")
	}
	history, ok := v4.(langchainschema.ChatMessageHistory)
	if !ok {
		return args, errors.New("history not memory.ChatMessageHistory")
	}

	instance := l.Instance
	options := GetChainOptions(instance.Spec.CommonChainConfig)

	// Check if have files as input
	v5, ok := args["documents"]
	if ok {
		docs, ok := v5.([]langchainschema.Document)
		if ok && len(docs) != 0 {
			args["max_number_of_conccurent"] = instance.Spec.MaxNumberOfConccurent
			mpChain := NewMapReduceChain(l.BaseNode, options...)
			err = mpChain.Init(ctx, nil, args)
			if err != nil {
				return args, err
			}
			_, err = mpChain.Run(ctx, nil, args)
			if err != nil {
				return args, err
			}
			// TODO:save out as a reference of following answer
		}
	}

	/*
		Note: we can only add context to retriever's documents not args["context"] if we use ConversationalRetrievalQA chain
		see https://github.com/kubeagi/langchaingo/blob/ca2f549e8d91788fd76a9f8706afcaae617275c5/chains/stuff_documents.go#L54-L69
		`args` are reset multiple times, and each Call uses different `args`.
		chains.Predict ---> ConversationalRetrievalQA.Call ---> CondenseQuestionChain.Call ---> Retriever.GetRelevantDocuments ---> combineDocumentsChain.Call
	*/
	// Add the agent output to qachain context
	if args[base.AgentOutputInArg] != nil {
		klog.FromContext(ctx).V(5).Info(fmt.Sprintf("get context from agent: %s", args[base.AgentOutputInArg]))
		docs, err := retriever.GetRelevantDocuments(ctx, args["question"].(string))
		if err != nil {
			return args, fmt.Errorf("can't get doc from retriever: %w", err)
		}
		doc := langchainschema.Document{PageContent: fmt.Sprintf("Tool Output of %s is: %s", args["question"].(string), args[base.AgentOutputInArg].(string))}
		docs = append(docs, doc)
		retriever = &appruntimeretriever.Fakeretriever{Docs: docs, Name: "AddAgentOutputRetriever"}
	}
	// Add the mapReduceDocument output to the context if it's not empty
	if args[base.MapReduceDocumentOutputInArg] != nil {
		// Note: Now args[base.MapReduceDocumentOutputInArg] will not be null only for the first "summarize content of uploaded document" Chat request
		// after uploading the file, and we don't need to bring in the content of the knowledgebase at this point.
		// see https://github.com/kubeagi/arcadia/pull/887#discussion_r1531469465
		klog.FromContext(ctx).V(5).Info(fmt.Sprintf("get context from mapReduceDocument: %s", args[base.MapReduceDocumentOutputInArg]))
		args[base.RuntimeRetrieverReferencesKeyInArg] = nil
		doc := langchainschema.Document{PageContent: args[base.MapReduceDocumentOutputInArg].(string)}
		retriever = &appruntimeretriever.Fakeretriever{Docs: []langchainschema.Document{doc}, Name: "AddMapReduceOutputRetriever"}
	}

	llmChain := chains.NewLLMChain(llm, prompt)
	if history != nil {
		llmChain.Memory = GetMemory(llm, instance.Spec.Memory, history, "", "")
	}
	llmChain.CallbacksHandler = log.KLogHandler{LogLevel: 3}
	condenseQustionGenerator := chains.LoadCondenseQuestionGenerator(llm)
	condenseQustionGenerator.CallbacksHandler = log.KLogHandler{LogLevel: 3}
	chain := chains.NewConversationalRetrievalQA(chains.NewStuffDocuments(llmChain), condenseQustionGenerator, retriever, GetMemory(llm, instance.Spec.Memory, history, "", ""))
	chain.RephraseQuestion = false
	l.ConversationalRetrievalQA = chain
	args["query"] = args["question"]
	var out string
	needStream := false
	needStream, ok = args[base.InputIsNeedStreamKeyInArg].(bool)
	if ok && needStream {
		options = append(options, chains.WithStreamingFunc(stream(args)))
		out, err = chains.Predict(ctx, l.ConversationalRetrievalQA, args, options...)
	} else {
		if len(options) > 0 {
			out, err = chains.Predict(ctx, l.ConversationalRetrievalQA, args, options...)
		} else {
			out, err = chains.Predict(ctx, l.ConversationalRetrievalQA, args)
		}
	}

	out, err = handleNoErrNoOut(ctx, needStream, out, err, l.ConversationalRetrievalQA, args, options)
	klog.FromContext(ctx).V(5).Info("use retrievalqachain, blocking out:" + out)
	if err == nil {
		args[base.OutputAnserKeyInArg] = out
		return args, nil
	}
	return args, fmt.Errorf("retrievalqachain run error: %w", err)
}

func (l *RetrievalQAChain) Ready() (isReady bool, msg string) {
	return l.Instance.Status.IsReadyOrGetReadyMessage()
}
