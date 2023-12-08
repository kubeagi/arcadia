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
	"bytes"
	"context"
	"errors"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/uuid"

	"github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/graphql-server/go-server/pkg/auth"
	"github.com/kubeagi/arcadia/graphql-server/go-server/pkg/client"
	"github.com/kubeagi/arcadia/pkg/application"
)

var Conversions = map[string]Conversion{}

func AppRun(ctx context.Context, req ChatReqBody) (*ChatRespBody, chan ChatRespBody, error) {
	token := auth.ForOIDCToken(ctx)
	c, err := client.GetClient(token)
	if err != nil {
		return nil, nil, err
	}
	obj, err := c.Resource(schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: v1alpha1.GroupVersion.Version, Resource: "applications"}).
		Namespace(req.AppNamespace).Get(ctx, req.APPName, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}
	app := &v1alpha1.Application{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), app)
	if err != nil {
		return nil, nil, err
	}
	if !app.Status.IsReady() {
		return nil, nil, errors.New("application is not ready")
	}
	var conversion Conversion
	if req.ConversionID != "" {
		var ok bool
		conversion, ok = Conversions[req.ConversionID]
		if !ok {
			return nil, nil, errors.New("conversion is not found")
		}
		if conversion.AppName != req.APPName || conversion.AppNamespce != req.AppNamespace {
			return nil, nil, errors.New("conversion id not match with app info")
		}
	} else {
		conversion = Conversion{
			ID:          string(uuid.NewUUID()),
			AppName:     req.APPName,
			AppNamespce: req.AppNamespace,
			StartedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Messages:    make([]Message, 0),
		}
	}
	conversion.Messages = append(conversion.Messages, Message{
		ID:     string(uuid.NewUUID()),
		Query:  req.Query,
		Answer: "",
	})
	appRun, err := application.NewAppOrGetFromCache(ctx, app, c)
	if err != nil {
		return nil, nil, err
	}
	out, outStream, err := appRun.Run(ctx, c, application.Input{Question: req.Query, NeedStream: req.ResponseMode == Streaming})
	if err != nil {
		return nil, nil, err
	}
	respStream := make(chan ChatRespBody, 1000)
	go func() {
		defer close(respStream)
		var res bytes.Buffer
		for {
			select {
			case <-ctx.Done():
				return
			case m := <-outStream:
				res.WriteString(m)
				respStream <- ChatRespBody{
					ConversionID: conversion.ID,
					Message:      m,
					CreatedAt:    time.Now(),
				}
				if res.String() == out.Answer {
					return
				}
			case <-time.After(3 * time.Second):
				return
			}
		}
	}()
	conversion.UpdatedAt = time.Now()
	conversion.Messages[len(conversion.Messages)-1].Answer = out.Answer
	Conversions[conversion.ID] = conversion
	return &ChatRespBody{
		ConversionID: conversion.ID,
		Message:      out.Answer,
		CreatedAt:    time.Now(),
	}, respStream, nil
}

// todo Reuse the flow without having to rebuild req same, not finish, Flow doesn't start with/contain nodes that depend on incomingInput.question
