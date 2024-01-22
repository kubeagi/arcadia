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

import "sync"

var _ Storage = (*MemoryStorage)(nil)

type MemoryStorage struct {
	mu            sync.Mutex
	conversations map[string]Conversation
}

// NewMemoryStorage creates a new MemoryStorage instance.
//
// No parameters.
// Returns a pointer to MemoryStorage.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		conversations: make(map[string]Conversation),
	}
}

// ListConversations retrieves conversations from MemoryStorage based on the provided options.
// It takes in optional SearchOption(s) and returns a slice of Conversation and an error.
func (m *MemoryStorage) ListConversations(opts ...SearchOption) (conversations []Conversation, err error) {
	searchOpt := applyOptions(nil, opts...)
	m.mu.Lock()
	for _, c := range m.conversations {
		if searchOpt.ConversationID != nil && c.ID != *searchOpt.ConversationID {
			continue
		}
		if searchOpt.AppName != nil && c.AppName != *searchOpt.AppName {
			continue
		}
		if searchOpt.AppNamespace != nil && c.AppNamespace != *searchOpt.AppNamespace {
			continue
		}
		if searchOpt.User != nil && c.User != *searchOpt.User {
			continue
		}
		if searchOpt.Debug != nil && c.Debug != *searchOpt.Debug {
			continue
		}
		conversations = append(conversations, c)
	}
	m.mu.Unlock()
	return conversations, nil
}

// UpdateConversation updates a conversation in the MemoryStorage.
//
// It takes a pointer to a Conversation as a parameter and returns an error.
func (m *MemoryStorage) UpdateConversation(conversation *Conversation) error {
	m.mu.Lock()
	m.conversations[conversation.ID] = *conversation
	m.mu.Unlock()
	return nil
}

func (m *MemoryStorage) FindExistingMessage(conversationID string, messageID string, opts ...SearchOption) (*Message, error) {
	conversation, err := m.FindExistingConversation(conversationID, opts...)
	if err != nil {
		return nil, err
	}
	for _, v := range conversation.Messages {
		v := v
		if v.ID == messageID {
			return &v, nil
		}
	}
	return nil, nil
}

// Delete deletes a conversation from MemoryStorage based on the provided options.
//
// Parameter(s): opts ...SearchOption
// Return type(s): error
func (m *MemoryStorage) Delete(opts ...SearchOption) (err error) {
	searchOpt := applyOptions(nil, opts...)
	var c *Conversation
	if searchOpt.ConversationID != nil {
		con, ok := m.conversations[*searchOpt.ConversationID]
		if !ok {
			return nil
		}
		c = &con
	} else {
		c, err = m.FindExistingConversation("", opts...)
		if err != nil {
			return err
		}
		if c == nil {
			return
		}
	}
	if searchOpt.User != nil && c.User != *searchOpt.User {
		return
	}
	if searchOpt.AppName != nil && c.AppName != *searchOpt.AppName {
		return
	}
	if searchOpt.AppNamespace != nil && c.AppNamespace != *searchOpt.AppNamespace {
		return
	}
	if searchOpt.Debug != nil && c.Debug != *searchOpt.Debug {
		return
	}
	m.mu.Lock()
	delete(m.conversations, c.ID)
	m.mu.Unlock()
	return nil
}

// FindExistingConversation searches for an existing conversation in MemoryStorage.
//
// ConversationID string, opt ...SearchOption. Returns *Conversation, error.
func (m *MemoryStorage) FindExistingConversation(conversationID string, opt ...SearchOption) (*Conversation, error) {
	searchOpt := applyOptions(&conversationID, opt...)
	m.mu.Lock()
	v, ok := m.conversations[*searchOpt.ConversationID]
	m.mu.Unlock()
	if !ok {
		return nil, ErrConversationNotFound
	}
	if searchOpt.Debug != nil && v.Debug != *searchOpt.Debug {
		return nil, ErrConversationNotFound
	}
	if searchOpt.AppName != nil && v.AppName != *searchOpt.AppName {
		return nil, ErrConversationNotFound
	}
	if searchOpt.AppNamespace != nil && v.AppNamespace != *searchOpt.AppNamespace {
		return nil, ErrConversationNotFound
	}
	if searchOpt.User != nil && v.User != *searchOpt.User {
		return nil, ErrConversationNotFound
	}
	return &v, nil
}
