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

type Search struct {
	ConversationID *string
	MessageID      *string
	AppName        *string
	AppNamespace   *string
	User           *string
	Debug          *bool
}

type SearchOption func(options *Search)

func NewSearchOptions(conversationID *string) *Search {
	return &Search{ConversationID: conversationID}
}

func applyOptions(conversationID *string, opts ...SearchOption) *Search {
	o := NewSearchOptions(conversationID)
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// WithConversationID returns a Search for setting the ConversationID.
func WithConversationID(id string) SearchOption {
	if id == "" {
		return func(o *Search) {}
	}
	return func(o *Search) {
		o.ConversationID = &id
	}
}

// WithMessageID returns a Search for setting the MessageID.
func WithMessageID(id string) SearchOption {
	if id == "" {
		return func(o *Search) {}
	}
	return func(o *Search) {
		o.MessageID = &id
	}
}

// WithAppName returns a Search for setting the AppName.
func WithAppName(name string) SearchOption {
	if name == "" {
		return func(o *Search) {}
	}
	return func(o *Search) {
		o.AppName = &name
	}
}

// WithAppNamespace returns a Search for setting the AppNamespace.
func WithAppNamespace(name string) SearchOption {
	if name == "" {
		return func(o *Search) {}
	}
	return func(o *Search) {
		o.AppNamespace = &name
	}
}

// WithUser returns a Search for setting the User.
func WithUser(name string) SearchOption {
	if name == "" {
		return func(o *Search) {}
	}
	return func(o *Search) {
		o.User = &name
	}
}

// WithDebug returns a Search for setting the Debug.
func WithDebug(debug bool) SearchOption {
	return func(o *Search) {
		o.Debug = &debug
	}
}
