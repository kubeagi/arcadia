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

package storage

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"

	"github.com/kubeagi/arcadia/pkg/appruntime/retriever"
)

func (r *References) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value:%#v", value)
	}

	result := make([]retriever.Reference, 0)
	err := json.Unmarshal(bytes, &result)
	if err != nil {
		return err
	}
	*r = result
	return nil
}

func (r References) Value() (driver.Value, error) {
	if r == nil || len([]retriever.Reference(r)) == 0 {
		return nil, nil
	}
	// return nil, nil
	return json.Marshal(r)
}

var _ Storage = (*PostgreSQLStorage)(nil)

type PostgreSQLStorage struct {
	db *gorm.DB
}

func (p *PostgreSQLStorage) CountMessages(appName, appNamespace string) (int64, error) {
	conversationQuery := Conversation{AppNamespace: appNamespace, AppName: appName}
	conversation := make([]Conversation, 0)
	tx := p.db.Select("id").Find(&conversation, conversationQuery)
	if tx.Error != nil {
		return 0, tx.Error
	}
	conversationIDs := make([]string, len(conversation))
	for i := range conversation {
		conversationIDs[i] = conversation[i].ID
	}
	var count int64
	tx = p.db.Model(&Message{}).Where("conversation_id IN ?", conversationIDs).Count(&count)
	if tx.Error != nil {
		return 0, tx.Error
	}
	return count, nil
}

func (p *PostgreSQLStorage) ListConversations(opts ...SearchOption) ([]Conversation, error) {
	searchOpt := applyOptions(nil, opts...)
	conversationQuery := Conversation{}
	if searchOpt.ConversationID != nil {
		conversationQuery.ID = *searchOpt.ConversationID
	}
	if searchOpt.Debug != nil {
		conversationQuery.Debug = *searchOpt.Debug
	}
	if searchOpt.User != nil {
		conversationQuery.User = *searchOpt.User
	}
	if searchOpt.AppName != nil {
		conversationQuery.AppName = *searchOpt.AppName
	}
	if searchOpt.AppNamespace != nil {
		conversationQuery.AppNamespace = *searchOpt.AppNamespace
	}
	conversationQuery.Debug = false
	conversationQuery.DeletedAt.Valid = false
	res := make([]Conversation, 0)
	tx := p.db.Preload("Messages.Documents").Order("updated_at DESC").Find(&res, conversationQuery)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return res, nil
}

func (p *PostgreSQLStorage) UpdateConversation(conversation *Conversation) error {
	tx := p.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(conversation)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func NewPostgreSQLStorage(conn *pgx.Conn) (*PostgreSQLStorage, error) {
	connPool := stdlib.OpenDB(*conn.Config())
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: connPool}), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&Conversation{}, &Message{}, &Document{}); err != nil {
		return nil, err
	}
	customLogger := logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             100 * time.Millisecond,
		LogLevel:                  logger.Info,
		IgnoreRecordNotFoundError: false,
		Colorful:                  true,
	})
	db.Logger = customLogger
	return &PostgreSQLStorage{
		db: db,
	}, nil
}

func (p *PostgreSQLStorage) FindExistingConversation(conversationID string, opts ...SearchOption) (*Conversation, error) {
	searchOpt := applyOptions(&conversationID, opts...)
	conversationQuery := Conversation{ID: conversationID}
	if searchOpt.Debug != nil {
		conversationQuery.Debug = *searchOpt.Debug
	}
	if searchOpt.User != nil {
		conversationQuery.User = *searchOpt.User
	}
	if searchOpt.AppName != nil {
		conversationQuery.AppName = *searchOpt.AppName
	}
	if searchOpt.AppNamespace != nil {
		conversationQuery.AppNamespace = *searchOpt.AppNamespace
	}
	conversationQuery.Debug = false
	conversationQuery.DeletedAt.Valid = false
	res := &Conversation{}
	tx := p.db.Preload("Messages.Documents").First(res, conversationQuery)
	if tx.Error != nil {
		return nil, tx.Error
	}

	for index, message := range res.Messages {
		// search document info based on object which is also a primary key in Document
		if message.Action != "UPLOAD" && message.Files != nil && len(message.Files) > 0 {
			documents, err := p.findMessageRelevantDocuments(message)
			if err == nil {
				message.Documents = documents
				res.Messages[index] = message
			}
		}
	}

	return res, nil
}

func (p *PostgreSQLStorage) findMessageRelevantDocuments(message Message) ([]Document, error) {
	var documents []Document
	err := p.db.Where("object IN ?", message.Files).Find(&documents).Error
	if err != nil {
		return nil, err
	}
	return documents, nil
}

func (p *PostgreSQLStorage) Delete(opts ...SearchOption) error {
	searchOpt := applyOptions(nil, opts...)
	c := &Conversation{}
	if searchOpt.ConversationID != nil {
		c.ID = *searchOpt.ConversationID
	}
	if searchOpt.User != nil {
		c.User = *searchOpt.User
	}
	if searchOpt.AppName != nil {
		c.AppName = *searchOpt.AppName
	}
	if searchOpt.AppNamespace != nil {
		c.AppNamespace = *searchOpt.AppNamespace
	}
	if searchOpt.Debug != nil {
		c.Debug = *searchOpt.Debug
	}
	tx := p.db.Select("Messages").Select("Documents").Delete(c)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}

func (p *PostgreSQLStorage) FindExistingMessage(conversationID string, messageID string, opts ...SearchOption) (*Message, error) {
	searchOpt := applyOptions(&conversationID, opts...)
	conversationQuery := Conversation{ID: conversationID}
	if searchOpt.Debug != nil {
		conversationQuery.Debug = *searchOpt.Debug
	}
	if searchOpt.User != nil {
		conversationQuery.User = *searchOpt.User
	}
	if searchOpt.AppName != nil {
		conversationQuery.AppName = *searchOpt.AppName
	}
	if searchOpt.AppNamespace != nil {
		conversationQuery.AppNamespace = *searchOpt.AppNamespace
	}
	conversationQuery.Debug = false
	conversationQuery.DeletedAt.Valid = false
	conversation := &Conversation{}
	message := &Message{}
	tx := p.db.Preload("Documents").First(message, Message{ID: messageID})
	if tx.Error != nil {
		return nil, tx.Error
	}
	tx = p.db.First(conversation, conversationQuery)
	if tx.Error != nil {
		return nil, tx.Error
	}
	association := p.db.Model(conversation).Association("Messages")
	if association.Error != nil {
		return nil, association.Error
	}
	if err := association.Find(message, Message{ID: messageID}); err != nil {
		return nil, err
	}
	return message, nil
}

func (p *PostgreSQLStorage) FindExistingDocument(conversationID, messageID string, documentID string, opts ...SearchOption) (*Document, error) {
	messageQuery := Message{ID: messageID}
	message := &Message{}
	document := &Document{}
	tx := p.db.First(message, messageQuery)
	if tx.Error != nil {
		return nil, tx.Error
	}
	association := p.db.Model(message).Association("Documents")
	if association.Error != nil {
		return nil, association.Error
	}
	if err := association.Find(document, Document{ID: documentID}); err != nil {
		return nil, err
	}
	return document, nil
}
