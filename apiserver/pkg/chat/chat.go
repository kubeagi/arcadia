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
	"github.com/kubeagi/arcadia/pkg/application"
	"github.com/kubeagi/arcadia/pkg/application/base"
	"github.com/kubeagi/arcadia/pkg/application/retriever"
)

var (
	mu          sync.Mutex
	Conversions = map[string]Conversion{}
)

func AppRun(ctx context.Context, req ChatReqBody, respStream chan string) (*ChatRespBody, error) {
	token := auth.ForOIDCToken(ctx)
	c, err := client.GetClient(token)
	if err != nil {
		return nil, err
	}
	obj, err := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "applications"}).
		Namespace(req.AppNamespace).Get(ctx, req.APPName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	app := &v1alpha1.Application{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), app)
	if err != nil {
		return nil, err
	}
	if !app.Status.IsReady() {
		return nil, errors.New("application is not ready")
	}
	var conversion Conversion
	currentUser, _ := ctx.Value(auth.UserNameContextKey).(string)
	if req.ConversionID != "" {
		var ok bool
		conversion, ok = Conversions[req.ConversionID]
		if !ok {
			return nil, errors.New("conversion is not found")
		}
		if currentUser != "" && currentUser != conversion.User {
			return nil, errors.New("conversion id not match with user")
		}
		if conversion.AppName != req.APPName || conversion.AppNamespce != req.AppNamespace {
			return nil, errors.New("conversion id not match with app info")
		}
		if conversion.Debug != req.Debug {
			return nil, errors.New("conversion id not match with debug")
		}
	} else {
		conversion = Conversion{
			ID:          string(uuid.NewUUID()),
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
	conversion.Messages = append(conversion.Messages, Message{
		ID:     messageID,
		Query:  req.Query,
		Answer: "",
	})
	ctx = base.SetAppNamespace(ctx, req.AppNamespace)
	appRun, err := application.NewAppOrGetFromCache(ctx, app, c)
	if err != nil {
		return nil, err
	}
	klog.Infoln("begin to run application", obj.GetName())
	out, err := appRun.Run(ctx, c, respStream, application.Input{Question: req.Query, NeedStream: req.ResponseMode == Streaming, History: conversion.History})
	if err != nil {
		return nil, err
	}

	conversion.UpdatedAt = time.Now()
	conversion.Messages[len(conversion.Messages)-1].Answer = out.Answer
	conversion.Messages[len(conversion.Messages)-1].References = out.References
	mu.Lock()
	Conversions[conversion.ID] = conversion
	mu.Unlock()
	return &ChatRespBody{
		ConversionID: conversion.ID,
		MessageID:    messageID,
		Message:      out.Answer,
		CreatedAt:    time.Now(),
		References:   out.References,
	}, nil
}

func ListConversations(ctx context.Context, req APPMetadata) ([]Conversion, error) {
	conversations := make([]Conversion, 0)
	currentUser, _ := ctx.Value(auth.UserNameContextKey).(string)
	mu.Lock()
	for _, c := range Conversions {
		if !c.Debug && c.AppName == req.APPName && c.AppNamespce == req.AppNamespace && (currentUser == "" || currentUser == c.User) {
			conversations = append(conversations, c)
		}
	}
	mu.Unlock()
	return conversations, nil
}

func DeleteConversation(ctx context.Context, conversionID string) error {
	currentUser, _ := ctx.Value(auth.UserNameContextKey).(string)
	mu.Lock()
	defer mu.Unlock()
	c, ok := Conversions[conversionID]
	if ok && (currentUser == "" || currentUser == c.User) {
		delete(Conversions, c.ID)
		return nil
	} else {
		return errors.New("conversion is not found")
	}
}

func ListMessages(ctx context.Context, req ConversionReqBody) (Conversion, error) {
	currentUser, _ := ctx.Value(auth.UserNameContextKey).(string)
	mu.Lock()
	defer mu.Unlock()
	for _, c := range Conversions {
		if c.AppName == req.APPName && c.AppNamespce == req.AppNamespace && req.ConversionID == c.ID && (currentUser == "" || currentUser == c.User) {
			return c, nil
		}
	}
	return Conversion{}, errors.New("conversion is not found")
}

func GetMessageReferences(ctx context.Context, req MessageReqBody) ([]retriever.Reference, error) {
	currentUser, _ := ctx.Value(auth.UserNameContextKey).(string)
	mu.Lock()
	defer mu.Unlock()
	for _, c := range Conversions {
		if c.AppName == req.APPName && c.AppNamespce == req.AppNamespace && c.ID == req.ConversionID && (currentUser == "" || currentUser == c.User) {
			for _, m := range c.Messages {
				if m.ID == req.MessageID {
					return m.References, nil
				}
			}
		}
	}
	return nil, errors.New("conversion or message is not found")
}

// todo Reuse the flow without having to rebuild req same, not finish, Flow doesn't start with/contain nodes that depend on incomingInput.question
