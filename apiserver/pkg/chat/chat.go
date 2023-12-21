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
)

var Conversions = map[string]Conversion{}

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
	if req.ConversionID != "" {
		var ok bool
		conversion, ok = Conversions[req.ConversionID]
		if !ok {
			return nil, errors.New("conversion is not found")
		}
		if conversion.AppName != req.APPName || conversion.AppNamespce != req.AppNamespace {
			return nil, errors.New("conversion id not match with app info")
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
		}
	}
	conversion.Messages = append(conversion.Messages, Message{
		ID:     string(uuid.NewUUID()),
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
	Conversions[conversion.ID] = conversion
	return &ChatRespBody{
		ConversionID: conversion.ID,
		Message:      out.Answer,
		CreatedAt:    time.Now(),
	}, nil
}

// todo Reuse the flow without having to rebuild req same, not finish, Flow doesn't start with/contain nodes that depend on incomingInput.question
