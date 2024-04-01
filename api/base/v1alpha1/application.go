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

package v1alpha1

import "fmt"

const (
	InputNode  = "Input"
	OutputNode = "Output"

	// AppCategoryLabelKey app category, the value is the id of the category.
	// here are multiple categories separated by commas. For example 1,2,3
	AppCategoryLabelKey = Group + "/app-category"

	// AppPublicLabelKey will add to app which is public
	AppPublicLabelKey = Group + "/app-is-public"

	// AppRecommendedLabelKey will add to app which is recommended
	AppRecommendedLabelKey = Group + "/app-is-recommended"

	DefaultChatTimeoutSeconds = 60
)

// ConversationFilePath is the path in system storage for file within a conversation
func ConversationFilePath(appName string, conversationID string, fileName string) string {
	return fmt.Sprintf("application/%s/conversation/%s/%s", appName, conversationID, fileName)
}
