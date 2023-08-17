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

package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	arcadiav1alpha1 "github.com/kubeagi/arcadia/api/v1alpha1"
)

// LLMReconciler reconciles a LLM object
type LLMReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=llms,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=llms/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=arcadia.kubeagi.k8s.com.cn,resources=llms/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the LLM object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *LLMReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling LLM resource")

	// Fetch the LLM instance
	instance := &arcadiav1alpha1.LLM{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// LLM instance has been deleted.
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !instance.DeletionTimestamp.IsZero() {
		// Instance is being deleted
		logger.Info("Instance is being deleted")
		err = r.DeleteLLM(ctx, logger, instance)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	err = r.UpdateLLM(ctx, logger, instance)
	if err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("Instance is updated and synchronized")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LLMReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&arcadiav1alpha1.LLM{}).
		Complete(r)
}

// UpdateLLM updates new LLM instance.
func (r *LLMReconciler) UpdateLLM(ctx context.Context, logger logr.Logger, instance *arcadiav1alpha1.LLM) error {
	logger.Info("Updating LLM instance")
	// Check if URL/Auth has changed
	if instance.Spec.URL != instance.Status.URL || instance.Spec.Auth != instance.Status.Auth {
		// Check new URL/Auth availability
		err := r.TestLLMAvailability(instance, logger)
		if err != nil {
			// Set status to unavailable
			instance.Status.SetConditions(arcadiav1alpha1.Condition{
				Type:               arcadiav1alpha1.TypeUnavailable,
				Status:             corev1.ConditionFalse,
				Reason:             arcadiav1alpha1.ReasonUnavailable,
				Message:            err.Error(),
				LastTransitionTime: metav1.Now(),
			})
		} else {
			// Set status to available
			instance.Status.SetConditions(arcadiav1alpha1.Condition{
				Type:               arcadiav1alpha1.TypeReady,
				Status:             corev1.ConditionTrue,
				Reason:             arcadiav1alpha1.ReasonAvailable,
				Message:            "Available",
				LastTransitionTime: metav1.Now(),
				LastSuccessfulTime: metav1.Now(),
			})
		}
		// Update URL/Auth
		instance.Status.URL = instance.Spec.URL
		instance.Status.Auth = instance.Spec.Auth
	}
	return r.Client.Update(ctx, instance)
}

// DeleteLLM deletes LLM instance.
func (r *LLMReconciler) DeleteLLM(ctx context.Context, logger logr.Logger, instance *arcadiav1alpha1.LLM) error {
	logger.Info("Deleting LLM instance")
	return r.Client.Delete(ctx, instance)
}

// TestLLMAvailability tests LLM availability.
func (r *LLMReconciler) TestLLMAvailability(instance *arcadiav1alpha1.LLM, logger logr.Logger) error {
	logger.Info("Testing LLM availability")
	testMethod := "GET"
	testURL := "https://open.bigmodel.cn"
	authKey := ""

	if instance.Spec.Type == arcadiav1alpha1.OpenAI {
		testMethod = "GET"
		testURL = instance.Spec.URL + "/v1/models"
		authKey = instance.Spec.Auth
	} else if instance.Spec.Type == arcadiav1alpha1.ZhiPuAI {
		testMethod = "POST"
		testURL = instance.Spec.URL + "/api/paas/v3/model-api/chatglm_lite/async-invoke"
		authKey = instance.Spec.Auth
	}

	err := SendTestRequest(testMethod, testURL, authKey)
	if err != nil {
		return err
	}

	return nil
}

func SendTestRequest(method string, url string, auth string) error {

	// Construct test prompt data
	testPrompt := []map[string]string{
		{"role": "user", "content": "你好"},
		{"role": "assistant", "content": "我是人工智能助手"},
		{"role": "user", "content": "你叫什么名字"},
		{"role": "assistant", "content": "我叫chatGLM"},
		{"role": "user", "content": "你都可以做些什么事"},
	}
	promptBytes, _ := json.Marshal(testPrompt)

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", "application/json")

	if method == "POST" {
		// Add prompt into body
		reqBody := req.Body
		newBody := bytes.NewBuffer(promptBytes)
		req.Body = io.NopCloser(newBody)
		defer reqBody.Close()
	}

	cli := &http.Client{}
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("returns unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
