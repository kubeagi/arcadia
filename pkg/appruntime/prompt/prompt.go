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

package prompt

import (
	"context"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/prompts"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/api/app-node/prompt/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/appruntime/base"
)

const (
	promptContextPlaceholder = "{{.context}}"
)

type Prompt struct {
	base.BaseNode
	prompts.ChatPromptTemplate
	Instance *v1alpha1.Prompt
}

func NewPrompt(baseNode base.BaseNode) *Prompt {
	return &Prompt{
		BaseNode:           baseNode,
		ChatPromptTemplate: prompts.ChatPromptTemplate{},
	}
}

func (p *Prompt) Init(ctx context.Context, cli client.Client, _ map[string]any) error {
	instance := &v1alpha1.Prompt{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: p.RefNamespace(), Name: p.Ref.Name}, instance); err != nil {
		return fmt.Errorf("can't find the prompt in cluster: %w", err)
	}
	p.Instance = instance
	return nil
}

func (p *Prompt) Run(ctx context.Context, cli client.Client, args map[string]any) (map[string]any, error) {
	instance := p.Instance
	ps := make([]prompts.MessageFormatter, 0)
	if instance.Spec.SystemMessage != "" {
		ps = append(ps, prompts.NewSystemMessagePromptTemplate(instance.Spec.SystemMessage, []string{}))
	}
	if instance.Spec.UserMessage != "" {
		if !strings.Contains(instance.Spec.UserMessage, promptContextPlaceholder) {
			// Add the context by default if it does not exist, and leave it empty
			// so we can add more contexts as needed in all agents/chains
			instance.Spec.UserMessage = fmt.Sprintf("%s\n%s", promptContextPlaceholder, instance.Spec.UserMessage)
		}
		ps = append(ps, prompts.NewHumanMessagePromptTemplate(instance.Spec.UserMessage, []string{"question"}))
	}
	template := prompts.NewChatPromptTemplate(ps)
	// todo format
	p.ChatPromptTemplate = template
	args["prompt"] = p
	return args, nil
}

func (p *Prompt) Ready() (isReady bool, msg string) {
	return p.Instance.Status.IsReadyOrGetReadyMessage()
}
