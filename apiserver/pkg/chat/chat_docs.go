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
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/klog/v2"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/base/v1alpha1"
	"github.com/kubeagi/arcadia/apiserver/pkg/auth"
	"github.com/kubeagi/arcadia/apiserver/pkg/chat/storage"
	"github.com/kubeagi/arcadia/apiserver/pkg/common"
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
	ds, err := common.SystemDatasourceOSS(ctx, cs.cli)
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
