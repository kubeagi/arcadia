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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/minio/minio-go/v7"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/pkg/auth"
	"github.com/kubeagi/arcadia/apiserver/pkg/chat/storage"
	pkgclient "github.com/kubeagi/arcadia/apiserver/pkg/client"
	pkgconfig "github.com/kubeagi/arcadia/pkg/config"
)

// ReceiveConversationDocs receive and process docs for a conversation
func (cs *ChatServer) ReceiveConversationFile(ctx context.Context, messageID string, req ConversationFilesReqBody, file *multipart.FileHeader) (*ChatRespBody, error) {
	if messageID == "" {
		messageID = string(uuid.NewUUID())
	}

	var conversation *storage.Conversation
	var err error
	currentUser, _ := ctx.Value(auth.UserNameContextKey).(string)
	if !req.NewChat {
		search := []storage.SearchOption{
			storage.WithAppName(req.APPName),
			storage.WithAppNamespace(req.AppNamespace),
		}
		if currentUser != "" {
			search = append(search, storage.WithUser(currentUser))
		}
		conversation, err = cs.Storage().FindExistingConversation(req.ConversationID, search...)
		if err != nil {
			return nil, err
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
		// create before upload documents
		if err := cs.Storage().UpdateConversation(conversation); err != nil {
			return nil, err
		}
	}

	// upload files to system datasource
	ds, err := pkgconfig.GetSystemDatasourceOSS(ctx)
	if err != nil {
		klog.Errorf("no storage service found with err %s", err)
		return nil, fmt.Errorf("no storage service found with err %s", err.Error())
	}

	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()
	data, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}
	// use sha256 as the object name so we can avoid overwrite files with same name but different content
	hash := sha256.Sum256(data)
	objectName := hex.EncodeToString(hash[:])
	objectPath := arcadiav1alpha1.ConversationFilePath(req.APPName, req.ConversationID, fmt.Sprintf("%s%s", objectName, filepath.Ext(file.Filename)))
	_, err = ds.Client.PutObject(
		ctx, req.AppNamespace,
		objectPath,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{
			// UserTags: map[string]string{
			// 	"FILE_NAME": file.Filename,
			// },
		})
	if err != nil {
		klog.Errorf("failed to store file %s with error %s", file.Filename, err.Error())
		return nil, fmt.Errorf("failed to store file %s with error %s", file.Filename, err.Error())
	}

	document := storage.Document{
		ID:             string(uuid.NewUUID()),
		MessageID:      messageID,
		ConversationID: req.ConversationID,
		Name:           file.Filename,
		Object:         objectPath,
	}

	// build/update conversation knowledgebase
	err = cs.BuildConversationKnowledgeBase(ctx, req, document)
	if err != nil {
		// only log error
		klog.Errorf("failed to build conversation knowledgebase %s with error %s", req.ConversationID, err.Error())
	}

	// process document with map-reduce
	message := storage.Message{
		ID:        messageID,
		Action:    "UPLOAD",
		Query:     "UPLOAD",
		Answer:    "DONE",
		Latency:   int64(time.Since(req.StartTime).Milliseconds()),
		Documents: []storage.Document{document},
	}

	// update conversat ion
	conversation.Messages = append(conversation.Messages, message)
	conversation.UpdatedAt = time.Now()
	// update the conversation with new message
	if err := cs.Storage().UpdateConversation(conversation); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if !ok {
			return nil, err
		}
		// 42P10 means confilict happens on object(primary key in pg)
		if pgErr.Code != "42P10" {
			return nil, err
		}
	}

	return &ChatRespBody{
		ConversationID: req.ConversationID,
		CreatedAt:      time.Now(),
		MessageID:      messageID,
		Action:         "UPLOAD",
		Message:        "Done",
		Latency:        message.Latency,
		Document: DocumentRespBody{
			ID:     document.ID,
			Name:   document.Name,
			Object: document.Object,
		},
	}, nil
}

// BuildConversationKnowledgeBase create/updates knowledgebase for this conversation.
// Conversation ID will be the knowledgebase name and document will be placed unde filegroup
// Knoweledgebase will embed the document into vectorstore which can be used in this conversation as references(similarity search)
func (cs *ChatServer) BuildConversationKnowledgeBase(ctx context.Context, req ConversationFilesReqBody, document storage.Document) error {
	// get system embedding suite
	embedder, vs, err := pkgconfig.GetSystemEmbeddingSuite(ctx)
	if err != nil {
		return err
	}

	// new knowledgebase
	kb := &arcadiav1alpha1.KnowledgeBase{
		ObjectMeta: v1.ObjectMeta{
			Name:      req.ConversationID,
			Namespace: req.AppNamespace,
			Labels: map[string]string{
				arcadiav1alpha1.LabelKnowledgeBaseType: string(arcadiav1alpha1.KnowledgeBaseTypeConversation),
			},
		},
		Spec: arcadiav1alpha1.KnowledgeBaseSpec{
			CommonSpec: arcadiav1alpha1.CommonSpec{
				DisplayName: "Conversation",
				Description: "Knowledgebase built for conversation",
			},
			Type:        arcadiav1alpha1.KnowledgeBaseTypeConversation,
			Embedder:    embedder.TypedObjectReference(),
			VectorStore: vs.TypedObjectReference(),
			FileGroups:  make([]arcadiav1alpha1.FileGroup, 0),
		},
	}

	// app as ownerreference
	app, _, err := cs.GetApp(ctx, req.APPName, req.AppNamespace)
	if err != nil {
		return err
	}
	// systemDatasource which stores the document
	systemDatasource, err := pkgconfig.GetSystemDatasource(ctx)
	if err != nil {
		return err
	}
	// create or update the conversation knowledgebase
	_, err = controllerutil.CreateOrUpdate(ctx, cs.cli, kb, func() error {
		if err := controllerutil.SetControllerReference(app, kb, pkgclient.Scheme); err != nil {
			return err
		}
		// append document path
		kb.Spec.FileGroups = append(kb.Spec.FileGroups, arcadiav1alpha1.FileGroup{
			Source: &arcadiav1alpha1.TypedObjectReference{
				APIGroup:  &arcadiav1alpha1.GroupVersion.Group,
				Kind:      "Datasource",
				Name:      systemDatasource.Name,
				Namespace: &systemDatasource.Namespace,
			},
			Files: []arcadiav1alpha1.FileWithVersion{{Path: document.Object}},
		})
		return nil
	})

	return err
}
