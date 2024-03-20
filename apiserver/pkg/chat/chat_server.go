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
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tmc/langchaingo/chains"
	langchainllms "github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/prompts"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/pkg/auth"
	"github.com/kubeagi/arcadia/apiserver/pkg/chat/storage"
	"github.com/kubeagi/arcadia/apiserver/pkg/client"
	"github.com/kubeagi/arcadia/pkg/appruntime"
	"github.com/kubeagi/arcadia/pkg/appruntime/base"
	appruntimechain "github.com/kubeagi/arcadia/pkg/appruntime/chain"
	"github.com/kubeagi/arcadia/pkg/appruntime/knowledgebase"
	"github.com/kubeagi/arcadia/pkg/appruntime/llm"
	"github.com/kubeagi/arcadia/pkg/appruntime/retriever"
	pkgconfig "github.com/kubeagi/arcadia/pkg/config"
	"github.com/kubeagi/arcadia/pkg/datasource"
	pkgdocumentloaders "github.com/kubeagi/arcadia/pkg/documentloaders"
)

type ChatServer struct {
	cli     runtimeclient.Client
	storage storage.Storage
	once    sync.Once
}

func NewChatServer(cli runtimeclient.Client) *ChatServer {
	return &ChatServer{
		cli: cli,
	}
}

func (cs *ChatServer) Storage() storage.Storage {
	if cs.storage == nil {
		cs.once.Do(func() {
			ctx := context.TODO()
			ds, err := pkgconfig.GetRelationalDatasource(ctx, cs.cli)
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

func (cs *ChatServer) AppRun(ctx context.Context, req ChatReqBody, respStream chan string, messageID string) (*ChatRespBody, error) {
	app, c, err := cs.getApp(ctx, req.APPName, req.AppNamespace)
	if err != nil {
		return nil, err
	}
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
	out, err := appRun.Run(ctx, c, respStream, appruntime.Input{Question: req.Query, Files: req.Files, NeedStream: req.ResponseMode.IsStreaming(), History: history})
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
	return cs.Storage().ListConversations(storage.WithAppNamespace(req.AppNamespace), storage.WithAppName(req.APPName), storage.WithUser(currentUser), storage.WithUser(currentUser))
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
	app, c, err := cs.getApp(ctx, req.APPName, req.AppNamespace)
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
	remains := limit
	if kb != nil {
		system, err := pkgconfig.GetSystemDatasource(ctx, c)
		if err != nil {
			return nil, err
		}
		endpoint := system.Spec.Endpoint.DeepCopy()
		if endpoint != nil && endpoint.AuthSecret != nil {
			endpoint.AuthSecret.WithNameSpace(system.Namespace)
		}
		ds, err := datasource.NewLocal(ctx, c, endpoint)
		if err != nil {
			return nil, err
		}
	Outer:
		for _, detail := range kb.Status.FileGroupDetail {
			if detail.Source == nil || detail.Source.Name == "" {
				continue
			}
			versionedDataset := &v1alpha1.VersionedDataset{}
			if err := c.Get(ctx, types.NamespacedName{Namespace: detail.Source.GetNamespace(kb.Namespace), Name: detail.Source.Name}, versionedDataset); err != nil {
				klog.Infof("failed to get versionedDataset: %s, try next one", err)
				continue
			}
			if !versionedDataset.Status.IsReady() {
				klog.Infof("versionedDataset is not ready, try next one")
				continue
			}
			info := &v1alpha1.OSS{Bucket: versionedDataset.Namespace}
			for _, fileDetails := range detail.FileDetails {
				info.Object = filepath.Join("dataset", versionedDataset.Spec.Dataset.Name, versionedDataset.Spec.Version, fileDetails.Path)
				file, err := ds.ReadFile(ctx, info)
				if err != nil {
					klog.Infof("failed to read file: %s, try next one", err)
					continue
				}
				defer file.Close()
				data, err := io.ReadAll(file)
				if err != nil {
					klog.Infof("failed to read file: %s, try next one", err)
					continue
				}
				dataReader := bytes.NewReader(data)
				doc, err := pkgdocumentloaders.NewQACSV(dataReader, "").Load(ctx)
				if err != nil {
					klog.Infof("failed to load doc file: %s, try next one", err)
					continue
				}
				for i := 0; i < remains && i < len(doc); i++ {
					content := strings.TrimPrefix(doc[i].PageContent, "q: ")
					promptStarters = append(promptStarters, content)
				}
				remains = limit - len(promptStarters)
				if remains == 0 {
					break Outer
				}
			}
		}
	} else {
		klog.V(3).Infoln("app has no knowlegebase, let llm generate some question")
		if model != nil {
			p := prompts.NewPromptTemplate(PromptForGeneratePromptStarters, []string{"limit", "displayName", "description"})
			var c *chains.LLMChain
			if len(chainOptions) > 0 {
				c = chains.NewLLMChain(model, p, chainOptions...)
			} else {
				c = chains.NewLLMChain(model, p)
			}
			result, err := chains.Predict(ctx, c,
				map[string]any{
					"limit":       limit,
					"displayName": app.Spec.DisplayName,
					"description": app.Spec.Description,
				},
			)
			if err != nil {
				return nil, err
			}
			res := strings.Split(result, "\n")
			for _, r := range res {
				promptStarters = append(promptStarters, strings.TrimSpace(r))
			}
		}
	}
	return promptStarters, nil
}

const PromptForGeneratePromptStarters = `You are the friendly and curious questioner, please ask {{.limit}} questions based on the name and description of this app below.

Requires language consistent with the name and description of the application, no restating of my words, questions only, one question per line, no subheadings.

The name of the application is: {{.displayName}}

The description of the application is: {{.description}}

The question you asked is:`

func (cs *ChatServer) getApp(ctx context.Context, appName, appNamespace string) (*v1alpha1.Application, runtimeclient.Client, error) {
	token := auth.ForOIDCToken(ctx)
	c, err := client.GetClient(token)
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
