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

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	llmzhipuai "github.com/kubeagi/arcadia/pkg/llms/zhipuai"
)

// log is for logging in this package.
var promptlog = logf.Log.WithName("prompt-resource")

func (p *Prompt) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(p).
		WithDefaulter(p).
		WithValidator(p).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-arcadia-kubeagi-k8s-com-cn-v1alpha1-prompt,mutating=true,failurePolicy=fail,sideEffects=None,groups=arcadia.kubeagi.k8s.com.cn,resources=portals,verbs=create;update,versions=v1alpha1,name=mprompt.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &Prompt{}

func (p *Prompt) Default(ctx context.Context, obj runtime.Object) error {
	promptlog.Info("default", "name", p.Name)

	// Override p.Spec.ZhiPuAIParams with default values if not nil
	if p.Spec.ZhiPuAIParams != nil {
		merged := llmzhipuai.MergeParams(*p.Spec.ZhiPuAIParams, llmzhipuai.DefaultModelParams())
		p.Spec.ZhiPuAIParams = &merged
	}

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-arcadia-kubeagi-k8s-com-cn-v1alpha1-prompt,mutating=false,failurePolicy=fail,sideEffects=None,groups=arcadia.kubeagi.k8s.com.cn,resources=prompts,verbs=create;update;delete,versions=v1alpha1,name=vprompt.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &Prompt{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (p *Prompt) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	promptlog.Info("validate create", "name", p.Name)

	if p.Spec.ZhiPuAIParams != nil {
		if err := llmzhipuai.ValidateModelParams(*p.Spec.ZhiPuAIParams); err != nil {
			promptlog.Error(err, "validate model params")
			return err
		}
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (p *Prompt) ValidateUpdate(ctx context.Context, oldObj runtime.Object, newObj runtime.Object) error {
	promptlog.Info("validate update", "name", p.Name)

	if p.Spec.ZhiPuAIParams != nil {
		if err := llmzhipuai.ValidateModelParams(*p.Spec.ZhiPuAIParams); err != nil {
			promptlog.Error(err, "validate model params")
			return err
		}
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (p *Prompt) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	promptlog.Info("validate delete", "name", p.Name)
	return nil
}
