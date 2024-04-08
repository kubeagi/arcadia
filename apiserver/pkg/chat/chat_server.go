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

package chat

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tmc/langchaingo/chains"
	langchainllms "github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/prompts"
	langchainschema "github.com/tmc/langchaingo/schema"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	apiretriever "github.com/kubeagi/arcadia/api/app-node/retriever/v1alpha1"
	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/config"
	"github.com/kubeagi/arcadia/apiserver/pkg/auth"
	"github.com/kubeagi/arcadia/apiserver/pkg/chat/storage"
	"github.com/kubeagi/arcadia/apiserver/pkg/client"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
	"github.com/kubeagi/arcadia/pkg/appruntime"
	"github.com/kubeagi/arcadia/pkg/appruntime/base"
	appruntimechain "github.com/kubeagi/arcadia/pkg/appruntime/chain"
	"github.com/kubeagi/arcadia/pkg/appruntime/knowledgebase"
	"github.com/kubeagi/arcadia/pkg/appruntime/llm"
	"github.com/kubeagi/arcadia/pkg/appruntime/retriever"
	pkgconfig "github.com/kubeagi/arcadia/pkg/config"
	"github.com/kubeagi/arcadia/pkg/datasource"
	"github.com/kubeagi/arcadia/pkg/documentloaders"
)

type ChatServer struct {
	cli     runtimeclient.Client
	storage storage.Storage
	once    sync.Once
	isGpts  bool
}

func NewChatServer(cli runtimeclient.Client, isGpts bool) *ChatServer {
	return &ChatServer{
		cli:    cli,
		isGpts: isGpts,
	}
}

func (cs *ChatServer) Storage() storage.Storage {
	if cs.storage == nil {
		cs.once.Do(func() {
			ctx := context.TODO()
			ds, err := pkgconfig.GetRelationalDatasource(ctx)
			if err != nil || ds == nil {
				if err != nil {
					klog.Infof("get relational datasource failed: %s, use memory storage for chat", err.Error())
				} else if ds == nil {
					klog.Infoln("no relational datasource found, use memory storage for chat")
				}
				cs.storage = storage.NewMemoryStorage()
				return
			}
			pg, err := datasource.GetPostgreSQLPool(ctx, cs.cli, ds)
			if err != nil {
				klog.Errorf("get postgresql pool failed : %s", err.Error())
				cs.storage = storage.NewMemoryStorage()
				return
			}
			conn, err := pg.Pool.Acquire(ctx)
			if err != nil {
				klog.Errorf("postgresql pool acquire failed : %s", err.Error())
				cs.storage = storage.NewMemoryStorage()
				return
			}
			db, err := storage.NewPostgreSQLStorage(conn.Conn())
			if err != nil {
				klog.Errorf("storage.NewPostgreSQLStorage failed : %s", err.Error())
				cs.storage = storage.NewMemoryStorage()
				return
			}
			klog.Infoln("use pg as chat storage.")
			cs.storage = db
		})
	}
	return cs.storage
}

func (cs *ChatServer) AppRun(ctx context.Context, req ChatReqBody, respStream chan string, messageID string, timeout *float64) (*ChatRespBody, error) {
	app, c, err := cs.GetApp(ctx, req.APPName, req.AppNamespace)
	if err != nil {
		return nil, err
	}
	*timeout = app.Spec.ChatTimeoutSecond
	var conversation *storage.Conversation
	history := memory.NewChatMessageHistory()
	currentUser, _ := ctx.Value(auth.UserNameContextKey).(string)
	if !req.NewChat {
		search := []storage.SearchOption{
			storage.WithAppName(req.APPName),
			storage.WithAppNamespace(req.AppNamespace),
			storage.WithDebug(req.Debug),
		}
		if currentUser != "" {
			search = append(search, storage.WithUser(currentUser))
		}
		conversation, err = cs.Storage().FindExistingConversation(req.ConversationID, search...)
		if err != nil {
			return nil, err
		}
		for _, v := range conversation.Messages {
			_ = history.AddUserMessage(ctx, v.Query)
			_ = history.AddAIMessage(ctx, v.Answer)
		}
	} else {
		conversation = &storage.Conversation{
			ID:           req.ConversationID,
			AppName:      req.APPName,
			AppNamespace: req.AppNamespace,
			StartedAt:    req.StartTime,
			Messages:     make([]storage.Message, 0),
			User:         currentUser,
			Debug:        req.Debug,
		}
		// create before do AppRun
		if err := cs.Storage().UpdateConversation(conversation); err != nil {
			return nil, err
		}
	}
	conversation.Messages = append(conversation.Messages, storage.Message{
		ID:     messageID,
		Action: "CHAT",
		Query:  req.Query,
		Answer: "",
	})
	appRun, err := appruntime.NewAppOrGetFromCache(ctx, c, app)
	if err != nil {
		return nil, err
	}
	klog.FromContext(ctx).Info("begin to run application", "appName", req.APPName, "appNamespace", req.AppNamespace)
	out, err := appRun.Run(ctx, c, respStream, appruntime.Input{Question: req.Query, Files: req.Files, NeedStream: req.ResponseMode.IsStreaming(), History: history, ConversationID: req.ConversationID})
	if err != nil {
		return nil, err
	}

	conversation.UpdatedAt = req.StartTime
	conversation.Messages[len(conversation.Messages)-1].Answer = out.Answer
	conversation.Messages[len(conversation.Messages)-1].References = out.References
	conversation.Messages[len(conversation.Messages)-1].Latency = time.Since(req.StartTime).Milliseconds()
	if req.Files != nil && len(req.Files) > 0 {
		conversation.Messages[len(conversation.Messages)-1].RawFiles = strings.Join(req.Files, ",")
	}

	if err := cs.Storage().UpdateConversation(conversation); err != nil {
		return nil, err
	}
	return &ChatRespBody{
		ConversationID: conversation.ID,
		MessageID:      messageID,
		Action:         "CHAT",
		Message:        out.Answer,
		CreatedAt:      time.Now(),
		References:     out.References,
	}, nil
}

func (cs *ChatServer) ListConversations(ctx context.Context, req APPMetadata) ([]storage.Conversation, error) {
	currentUser, _ := ctx.Value(auth.UserNameContextKey).(string)
	return cs.Storage().ListConversations(storage.WithAppNamespace(req.AppNamespace), storage.WithAppName(req.APPName), storage.WithUser(currentUser))
}

func (cs *ChatServer) DeleteConversation(ctx context.Context, conversationID string) error {
	currentUser, _ := ctx.Value(auth.UserNameContextKey).(string)
	// Note: in pg table, this data is marked as deleted, deleted_at column is not null. the pdf in minio is not deleted. we only delete the conversation knowledgebase.
	// delete conversation knowledgebase if it exists
	token := auth.ForOIDCToken(ctx)
	c, err := client.GetClient(token)
	if err != nil {
		return fmt.Errorf("failed to get a client: %w", err)
	}
	kbList := &v1alpha1.KnowledgeBaseList{}
	if err = runtimeclient.IgnoreNotFound(c.List(ctx, kbList, runtimeclient.MatchingFields(map[string]string{"metadata.name": conversationID}))); err != nil {
		return err
	}
	if len(kbList.Items) == 1 {
		kb := &kbList.Items[0]
		if err = c.Delete(ctx, kb); err != nil {
			return err
		}
	} else if len(kbList.Items) > 1 {
		return fmt.Errorf("multiple conversation knowledgebases found")
	}
	return cs.Storage().Delete(storage.WithConversationID(conversationID), storage.WithUser(currentUser))
}

func (cs *ChatServer) ListMessages(ctx context.Context, req ConversationReqBody) (storage.Conversation, error) {
	currentUser, _ := ctx.Value(auth.UserNameContextKey).(string)
	c, err := cs.Storage().FindExistingConversation(req.ConversationID, storage.WithAppNamespace(req.AppNamespace), storage.WithAppName(req.APPName), storage.WithAppNamespace(req.AppNamespace), storage.WithUser(currentUser))
	if err != nil {
		return storage.Conversation{}, err
	}
	if c != nil {
		return *c, nil
	}
	return storage.Conversation{}, errors.New("conversation is not found")
}

func (cs *ChatServer) GetMessageReferences(ctx context.Context, req MessageReqBody) ([]retriever.Reference, error) {
	currentUser, _ := ctx.Value(auth.UserNameContextKey).(string)
	m, err := cs.Storage().FindExistingMessage(req.ConversationID, req.MessageID, storage.WithAppNamespace(req.AppNamespace), storage.WithAppName(req.APPName), storage.WithAppNamespace(req.AppNamespace), storage.WithUser(currentUser))
	if err != nil {
		return nil, err
	}
	if m != nil && m.References != nil {
		return m.References, nil
	}
	return nil, errors.New("conversation or message is not found")
}

// ListPromptStarters PromptStarter are examples for users to help them get up and running with the application quickly. We use same name with chatgpt
func (cs *ChatServer) ListPromptStarters(ctx context.Context, req APPMetadata, limit int) (promptStarters []string, err error) {
	app, c, err := cs.GetApp(ctx, req.APPName, req.AppNamespace)
	if err != nil {
		return nil, err
	}
	var kb *v1alpha1.KnowledgeBase
	var chainOptions []chains.ChainCallOption
	var model langchainllms.Model
	for _, n := range app.Spec.Nodes {
		baseNode := base.NewBaseNode(app.Namespace, n.Name, *n.Ref)
		switch baseNode.Group() {
		case "chain":
			switch baseNode.Kind() {
			case "llmchain":
				ch := appruntimechain.NewLLMChain(baseNode)
				if err := ch.Init(ctx, c, nil); err != nil {
					klog.Infof("init llmchain err:%s, will use empty chain config", err)
				}
				chainOptions = appruntimechain.GetChainOptions(ch.Instance.Spec.CommonChainConfig)
			case "retrievalqachain":
				ch := appruntimechain.NewRetrievalQAChain(baseNode)
				if err := ch.Init(ctx, c, nil); err != nil {
					klog.Infof("init retrievalqachain err:%s, will use empty chain config", err)
				}
				chainOptions = appruntimechain.GetChainOptions(ch.Instance.Spec.CommonChainConfig)
			case "apichain":
				ch := appruntimechain.NewAPIChain(baseNode)
				if err := ch.Init(ctx, c, nil); err != nil {
					klog.Infof("init apichain err:%s, will use empty chain config", err)
				}
				chainOptions = appruntimechain.GetChainOptions(ch.Instance.Spec.CommonChainConfig)
			default:
				klog.Infoln("can't find chain config in app, use empty chain config")
			}
		case "":
			switch baseNode.Kind() {
			case "llm":
				l := llm.NewLLM(baseNode)
				if err := l.Init(ctx, c, nil); err != nil {
					klog.Infof("init llm err:%s, abort", err)
					return nil, err
				}
				model = l.Model
			case "knowledgebase":
				k := knowledgebase.NewKnowledgebase(baseNode)
				if err := k.Init(ctx, c, nil); err != nil {
					klog.Infof("init knowledgebase err:%s, abort", err)
					return nil, err
				}
				kb = k.Instance
			}
		}
	}
	promptStarters = make([]string, 0, limit)
	content := bytes.Buffer{}
	// if there is a knowledgebase, use it to generate prompt starter
	if kb != nil {
		outArg, finish, err := retriever.GenerateKnowledgebaseRetriever(ctx, c, kb.Name, kb.Namespace, apiretriever.CommonRetrieverConfig{NumDocuments: limit * 2}, map[string]any{"question": "开始"})
		if err != nil {
			return nil, err
		}
		if finish != nil {
			defer finish()
		}
		v, ok := outArg[base.LangchaingoRetrieverKeyInArg]
		if ok {
			r, ok := v.(langchainschema.Retriever)
			if ok {
				doc, err := r.GetRelevantDocuments(ctx, "")
				if err != nil {
					return nil, err
				}
				for _, d := range doc {
					hasAnswer := false
					// has answer, means qa.csv, just return the question
					v, ok := d.Metadata[documentloaders.AnswerCol]
					if ok {
						answer, ok := v.(string)
						if ok && answer != "" {
							question := strings.TrimSuffix(d.PageContent, "\na: "+answer)
							promptStarters = append(promptStarters, strings.TrimPrefix(question, "q: "))
							hasAnswer = true
							if len(promptStarters) == limit {
								break
							}
						}
					}
					if !hasAnswer {
						content.WriteString(d.PageContent + "\n")
					}
				}
			}
		}
	}
	if len(promptStarters) == limit {
		klog.V(3).Infoln("app has knowlegebase with qa.csv, just read some question")
		return promptStarters, nil
	}
	if model == nil {
		return nil, fmt.Errorf("can't find model in app")
	}
	var p prompts.PromptTemplate
	predictArg := map[string]any{"limit": limit}
	if content.Len() > 0 {
		klog.V(3).Infoln("app has knowlegebase with chunk information, let llm generate some question")
		p = prompts.NewPromptTemplate(PromptForGeneratePromptStartersByChunk, []string{"limit", "information"})
		contentStr := content.String()
		// if content is too long, may cause llm error
		if len(contentStr) > 500 {
			contentStr = contentStr[0:500]
		}
		predictArg["information"] = contentStr
	} else {
		klog.V(3).Infoln("app has no knowlegebase, let llm generate some question")
		p = prompts.NewPromptTemplate(PromptForGeneratePromptStartersByAppInfo, []string{"limit", "displayName", "description"})
		predictArg["displayName"] = app.Spec.DisplayName
		predictArg["description"] = app.Spec.Description
	}
	var llmchain *chains.LLMChain
	if len(chainOptions) > 0 {
		llmchain = chains.NewLLMChain(model, p, chainOptions...)
	} else {
		llmchain = chains.NewLLMChain(model, p)
	}
	result, err := chains.Predict(ctx, llmchain, predictArg)
	if err != nil {
		return nil, err
	}
	res := strings.Split(result, "\n")
	for _, r := range res {
		promptStarters = append(promptStarters, strings.TrimSpace(r))
	}
	return promptStarters, nil
}

const PromptForGeneratePromptStartersByAppInfo = `You are the friendly and curious questioner, please ask {{.limit}} questions based on the name and description of this app below.

Requires language consistent with the name and description of the application, no restating of my words, questions only, one question per line, no subheadings.

The name of the application is: {{.displayName}}

The description of the application is: {{.description}}

The question you asked is:`

const PromptForGeneratePromptStartersByChunk = `You are the friendly and curious questioner, please ask {{.limit}} questions based on the information below.

Requires language consistent with the information, no restating of my words, questions only, one question per line, no subheadings.
---
{{.information}}
---
The question you asked is:`

func (cs *ChatServer) GetApp(ctx context.Context, appName, appNamespace string) (*v1alpha1.Application, runtimeclient.Client, error) {
	token := auth.ForOIDCToken(ctx)
	var c runtimeclient.Client
	var err error
	if !cs.isGpts {
		c, err = client.GetClient(token)
	} else {
		c = cs.cli
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get a client: %w", err)
	}
	app := &v1alpha1.Application{}
	if err := c.Get(ctx, types.NamespacedName{Namespace: appNamespace, Name: appName}, app); err != nil {
		return nil, c, fmt.Errorf("failed to get application: %w", err)
	}
	if !app.Status.IsReady() {
		return nil, c, fmt.Errorf("application not ready: %s", app.Status.GetCondition(v1alpha1.TypeReady).Message)
	}
	return app, c, nil
}

// todo Reuse the flow without having to rebuild req same, not finish, Flow doesn't start with/contain nodes that depend on incomingInput.question

func (cs *ChatServer) FillAppIconToConversations(ctx context.Context, conversations *[]storage.Conversation) error {
	if conversations == nil {
		return nil
	}
	appMap := make(map[string]int, len(*conversations))
	i := 0
	for _, c := range *conversations {
		key := fmt.Sprintf("%s/%s", c.AppNamespace, c.AppName)
		if _, exist := appMap[key]; exist {
			continue
		}
		appMap[key] = i
		i++
	}
	result := make([]string, len(appMap))
	g, _ := errgroup.WithContext(ctx)
	g.SetLimit(10)
	for key, index := range appMap {
		key, index := key, index
		g.Go(func() error {
			app := &v1alpha1.Application{}
			ns, name, ok := strings.Cut(key, "/")
			if !ok {
				return nil
			}
			app.Name = name
			app.Namespace = ns
			link := common.AppIconLink(app, config.GetConfig().PlaygroundEndpointPrefix)
			result[index] = link
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	for i, c := range *conversations {
		c.Icon = result[appMap[fmt.Sprintf("%s/%s", c.AppNamespace, c.AppName)]]
		(*conversations)[i] = c
	}
	return nil
}
