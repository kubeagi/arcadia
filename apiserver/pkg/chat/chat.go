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

package chat

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/tmc/langchaingo/memory"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/klog/v2"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/pkg/auth"
	"github.com/kubeagi/arcadia/apiserver/pkg/client"
	"github.com/kubeagi/arcadia/pkg/appruntime"
	"github.com/kubeagi/arcadia/pkg/appruntime/base"
	"github.com/kubeagi/arcadia/pkg/appruntime/retriever"
)

var (
	mu            sync.Mutex
	Conversations = map[string]Conversation{}
)

func AppRun(ctx context.Context, req ChatReqBody, respStream chan string) (*ChatRespBody, error) {
	token := auth.ForOIDCToken(ctx)
	c, err := client.GetClient(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get a dynamic client: %w", err)
	}
	obj, err := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "applications"}).
		Namespace(req.AppNamespace).Get(ctx, req.APPName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	}
	app := &v1alpha1.Application{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), app)
	if err != nil {
		return nil, fmt.Errorf("failed to convert application: %w", err)
	}
	if !app.Status.IsReady() {
		return nil, errors.New("application is not ready")
	}
	var conversation Conversation
	currentUser, _ := ctx.Value(auth.UserNameContextKey).(string)
	if !req.NewChat {
		var ok bool
		conversation, ok = Conversations[req.ConversationID]
		if !ok {
			return nil, errors.New("conversation is not found")
		}
		if currentUser != "" && currentUser != conversation.User {
			return nil, errors.New("conversation id not match with user")
		}
		if conversation.AppName != req.APPName || conversation.AppNamespce != req.AppNamespace {
			return nil, errors.New("conversation id not match with app info")
		}
		if conversation.Debug != req.Debug {
			return nil, errors.New("conversation id not match with debug")
		}
	} else {
		conversation = Conversation{
			ID:          req.ConversationID,
			AppName:     req.APPName,
			AppNamespce: req.AppNamespace,
			StartedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Messages:    make([]Message, 0),
			History:     memory.NewChatMessageHistory(),
			User:        currentUser,
			Debug:       req.Debug,
		}
	}
	messageID := string(uuid.NewUUID())
	conversation.Messages = append(conversation.Messages, Message{
		ID:     messageID,
		Query:  req.Query,
		Answer: "",
	})
	ctx = base.SetAppNamespace(ctx, req.AppNamespace)
	appRun, err := appruntime.NewAppOrGetFromCache(ctx, c, app)
	if err != nil {
		return nil, err
	}
	klog.FromContext(ctx).Info("begin to run application", "appName", req.APPName, "appNamespace", req.AppNamespace)
	out, err := appRun.Run(ctx, c, respStream, appruntime.Input{Question: req.Query, NeedStream: req.ResponseMode.IsStreaming(), History: conversation.History})
	if err != nil {
		return nil, err
	}

	conversation.UpdatedAt = time.Now()
	conversation.Messages[len(conversation.Messages)-1].Answer = out.Answer
	conversation.Messages[len(conversation.Messages)-1].References = out.References
	mu.Lock()
	Conversations[conversation.ID] = conversation
	mu.Unlock()
	return &ChatRespBody{
		ConversationID: conversation.ID,
		MessageID:      messageID,
		Message:        out.Answer,
		CreatedAt:      time.Now(),
		References:     out.References,
	}, nil
}

func ListConversations(ctx context.Context, req APPMetadata) ([]Conversation, error) {
	conversations := make([]Conversation, 0)
	currentUser, _ := ctx.Value(auth.UserNameContextKey).(string)
	mu.Lock()
	for _, c := range Conversations {
		if !c.Debug && c.AppName == req.APPName && c.AppNamespce == req.AppNamespace && (currentUser == "" || currentUser == c.User) {
			conversations = append(conversations, c)
		}
	}
	mu.Unlock()
	return conversations, nil
}

func DeleteConversation(ctx context.Context, conversationID string) error {
	currentUser, _ := ctx.Value(auth.UserNameContextKey).(string)
	mu.Lock()
	defer mu.Unlock()
	c, ok := Conversations[conversationID]
	if ok && (currentUser == "" || currentUser == c.User) {
		delete(Conversations, c.ID)
		return nil
	} else {
		return errors.New("conversation is not found")
	}
}

func ListMessages(ctx context.Context, req ConversationReqBody) (Conversation, error) {
	currentUser, _ := ctx.Value(auth.UserNameContextKey).(string)
	mu.Lock()
	defer mu.Unlock()
	for _, c := range Conversations {
		if c.AppName == req.APPName && c.AppNamespce == req.AppNamespace && req.ConversationID == c.ID && (currentUser == "" || currentUser == c.User) {
			return c, nil
		}
	}
	return Conversation{}, errors.New("conversation is not found")
}

func GetMessageReferences(ctx context.Context, req MessageReqBody) ([]retriever.Reference, error) {
	currentUser, _ := ctx.Value(auth.UserNameContextKey).(string)
	mu.Lock()
	defer mu.Unlock()
	for _, c := range Conversations {
		if c.AppName == req.APPName && c.AppNamespce == req.AppNamespace && c.ID == req.ConversationID && (currentUser == "" || currentUser == c.User) {
			for _, m := range c.Messages {
				if m.ID == req.MessageID {
					return m.References, nil
				}
			}
		}
	}
	return nil, errors.New("conversation or message is not found")
}

// todo Reuse the flow without having to rebuild req same, not finish, Flow doesn't start with/contain nodes that depend on incomingInput.question
