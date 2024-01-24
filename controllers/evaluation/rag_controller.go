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

package evaluationarcadia

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	evaluationarcadiav1alpha1 "github.com/kubeagi/arcadia/api/evaluation/v1alpha1"
	"github.com/kubeagi/arcadia/pkg/evaluation"
)

var errJobNotDone = errors.New("wait for the job to complete, go to the next step")

// RAGReconciler reconciles a RAG object
type RAGReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=evaluation.arcadia.kubeagi.k8s.com.cn,resources=rags,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=evaluation.arcadia.kubeagi.k8s.com.cn,resources=rags/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=evaluation.arcadia.kubeagi.k8s.com.cn,resources=rags/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=*
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=*

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RAG object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *RAGReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.V(5).Info("Start RAG Reconcile")

	// TODO(user): your logic here
	instance := &evaluationarcadiav1alpha1.RAG{}
	if err := r.Client.Get(ctx, req.NamespacedName, instance); err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.V(1).Info("failed to get rag")
		return ctrl.Result{}, err
	}
	if instance.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	if instance.Labels == nil {
		instance.Labels = make(map[string]string)
	}
	if app, ok := instance.Labels[evaluationarcadiav1alpha1.EvaluationApplicationLabel]; !ok || app != instance.Spec.Application.Name {
		instance.Labels[evaluationarcadiav1alpha1.EvaluationApplicationLabel] = instance.Spec.Application.Name
		err := r.Client.Update(ctx, instance)
		if err != nil {
			logger.Error(err, "failed to add application name label")
		}
		return ctrl.Result{}, err
	}
	return r.phaseHandler(ctx, instance)
}

// SetupWithManager sets up the controller with the Manager.
func (r *RAGReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&evaluationarcadiav1alpha1.RAG{}, builder.WithPredicates(predicate.Funcs{
			UpdateFunc: func(ue event.UpdateEvent) bool {
				n := ue.ObjectNew.(*evaluationarcadiav1alpha1.RAG)
				o := ue.ObjectOld.(*evaluationarcadiav1alpha1.RAG)
				if !reflect.DeepEqual(n.Spec, o.Spec) {
					// If the spec portion of the RAG changes, the process needs to be re-executed
					if evaluationarcadiav1alpha1.RAGSpecChanged(n.Spec, o.Spec) {
						_ = r.DeleteJobsAndPvc(context.TODO(), n)
						return false
					}
					return true
				}
				if evaluationarcadiav1alpha1.RagStatusChanged(n.Status, o.Status) {
					return true
				}
				if !reflect.DeepEqual(n.Labels, o.Labels) {
					return true
				}
				return false
			},
		})).
		Watches(&source.Kind{
			Type: &corev1.PersistentVolumeClaim{},
		}, handler.Funcs{
			DeleteFunc: func(de event.DeleteEvent, rli workqueue.RateLimitingInterface) {
				pvc := de.Object.(*corev1.PersistentVolumeClaim)
				r.WhenPVCDeleted(pvc)
			},
		}).
		Watches(&source.Kind{
			Type: &batchv1.Job{},
		}, handler.Funcs{
			UpdateFunc: func(ue event.UpdateEvent, rli workqueue.RateLimitingInterface) {
				job := ue.ObjectNew.(*batchv1.Job)
				old := ue.ObjectOld.(*batchv1.Job)
				if !reflect.DeepEqual(job.Status.Conditions, old.Status.Conditions) {
					r.WhenJobChanged(job)
				}
			},
		}).
		Complete(r)
}

func (r *RAGReconciler) DeleteJobsAndPvc(ctx context.Context, instance *evaluationarcadiav1alpha1.RAG) error {
	logger := log.FromContext(ctx)
	selector := labels.NewSelector()
	requirtment, _ := labels.NewRequirement(evaluationarcadiav1alpha1.EvaluationJobLabels, selection.Equals, []string{instance.Name})
	selector = selector.Add(*requirtment)

	m := metav1.DeletePropagationForeground
	job := &batchv1.Job{}
	err := r.Client.DeleteAllOf(ctx, job, &client.DeleteAllOfOptions{
		DeleteOptions: client.DeleteOptions{
			PropagationPolicy: &m,
		},
		ListOptions: client.ListOptions{
			Namespace:     instance.Namespace,
			LabelSelector: selector,
		},
	})
	if err != nil && !k8serrors.IsNotFound(err) {
		logger.Error(err, "sepc changed, failed to delete rag associated job.")
		return err
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
	}

	err = r.Client.Delete(ctx, pvc, &client.DeleteOptions{
		PropagationPolicy: &m,
	})
	if err != nil && !k8serrors.IsNotFound(err) {
		logger.Error(err, "spec changed, failed to delete pvc", "PvcName", pvc.Name)
		return err
	}

	deepCopyInstance := instance.DeepCopy()
	deepCopyInstance.Status.Conditions = nil
	deepCopyInstance.Status.Phase = ""
	logger.Info("spec changes, delete all related resources")
	return r.Client.Status().Patch(ctx, deepCopyInstance, client.MergeFrom(instance))
}

func (r *RAGReconciler) phaseHandler(ctx context.Context, instance *evaluationarcadiav1alpha1.RAG) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	curPhase := instance.Status.Phase
	switch curPhase {
	case "":
		deepCopyInstance := instance.DeepCopy()
		deepCopyInstance.Status.Phase = evaluationarcadiav1alpha1.InitPvcPhase
		deepCopyInstance.Status.Conditions = []batchv1.JobCondition{
			{
				Type:    batchv1.JobComplete,
				Status:  corev1.ConditionFalse,
				Message: "need to create pvc",
			},
		}
		err := r.Client.Status().Patch(ctx, deepCopyInstance, client.MergeFrom(instance))
		if err != nil {
			logger.Error(err, "failed to initialize RAG state")
		}
		return ctrl.Result{}, err
	case evaluationarcadiav1alpha1.InitPvcPhase:
		err := r.initPVC(ctx, instance)
		return ctrl.Result{}, err
	case evaluationarcadiav1alpha1.DownloadFilesPhase:
		err := r.JobGenerator(ctx, instance, curPhase, evaluationarcadiav1alpha1.GenerateTestFilesPhase, evaluation.DownloadJob)
		if err != nil && err != errJobNotDone {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	case evaluationarcadiav1alpha1.GenerateTestFilesPhase:
		err := r.JobGenerator(ctx, instance, curPhase, evaluationarcadiav1alpha1.JudgeLLMPhase, evaluation.GenTestDataJob)
		if err != nil && err != errJobNotDone {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	case evaluationarcadiav1alpha1.JudgeLLMPhase:
		err := r.JobGenerator(ctx, instance, curPhase, evaluationarcadiav1alpha1.UploadFilesPhase, evaluation.JudgeJobGenerator(ctx, r.Client))
		if err != nil && err != errJobNotDone {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	case evaluationarcadiav1alpha1.UploadFilesPhase:
		err := r.JobGenerator(ctx, instance, curPhase, evaluationarcadiav1alpha1.CompletePhase, evaluation.UploadJobGenerator(ctx, r.Client))
		if err != nil && err != errJobNotDone {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	case evaluationarcadiav1alpha1.CompletePhase:
		logger.Info("evaluation process complete, end reconcile")
	}
	return ctrl.Result{}, nil
}

func (r *RAGReconciler) initPVC(ctx context.Context, instance *evaluationarcadiav1alpha1.RAG) error {
	logger := log.FromContext(ctx)
	deepCopyInstance := instance.DeepCopy()
	for _, cond := range instance.Status.Conditions {
		if cond.Type == batchv1.JobComplete && cond.Status == corev1.ConditionTrue {
			// next phase
			deepCopyInstance.Status.Phase = evaluationarcadiav1alpha1.DownloadFilesPhase
			deepCopyInstance.Status.Conditions = []batchv1.JobCondition{
				{
					Type:    batchv1.JobComplete,
					Status:  corev1.ConditionFalse,
					Message: "pvc creation complete, create download file job",
				},
			}
			err := r.Client.Status().Patch(ctx, deepCopyInstance, client.MergeFrom(instance))
			if err != nil {
				logger.Error(err, "update the status of the rag to start downloading the file failed.")
			}
			return err
		}
	}

	pvc := corev1.PersistentVolumeClaim{}
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: instance.Namespace, Name: instance.Name}, &pvc); err != nil {
		if !k8serrors.IsNotFound(err) {
			logger.Error(err, "failed to get pvc", "PVCName", instance.Name)
			return err
		}
		pvc.Name = instance.Name
		pvc.Namespace = instance.Namespace
		pvc.Spec = *instance.Spec.Storage
		_ = controllerutil.SetOwnerReference(instance, &pvc, r.Scheme)
		err = r.Client.Create(ctx, &pvc)
		if err != nil {
			logger.Error(err, "failed to create pvc", "PVCName", pvc.Name)
			deepCopyInstance.Status.Conditions = []batchv1.JobCondition{
				{
					Type:               batchv1.JobFailed,
					Status:             corev1.ConditionTrue,
					Message:            fmt.Sprintf("pvc creation failure. %s", err),
					LastTransitionTime: metav1.Now(),
				},
			}
			return r.Client.Status().Patch(ctx, deepCopyInstance, client.MergeFrom(instance))
		}
	}
	if pvc.DeletionTimestamp != nil {
		logger.Info("pvc is being deleted, need to wait for next process", "PVCname", pvc.Name)
		return errors.New("pvc is being deleted, need to wait for next process")
	}
	deepCopyInstance.Status.Conditions = []batchv1.JobCondition{
		{
			Type:               batchv1.JobComplete,
			Status:             corev1.ConditionTrue,
			Message:            "pvc created successfully",
			LastTransitionTime: metav1.Now(),
		},
	}

	logger.Info("pvc already exists", "PVCName", pvc.Name, "Phase", pvc.Status.Phase)
	return r.Client.Status().Patch(ctx, deepCopyInstance, client.MergeFrom(instance))
}

func (r *RAGReconciler) JobGenerator(
	ctx context.Context,
	instance *evaluationarcadiav1alpha1.RAG,
	curPhase, nextPhse evaluationarcadiav1alpha1.RAGPhase,
	genJob func(*evaluationarcadiav1alpha1.RAG) (*batchv1.Job, error),
) error {
	logger := log.FromContext(ctx)
	deepCopyInstance := instance.DeepCopy()
	for _, cond := range deepCopyInstance.Status.Conditions {
		if cond.Type == batchv1.JobComplete && cond.Status == corev1.ConditionTrue {
			deepCopyInstance.Status.Phase = nextPhse
			d := batchv1.JobCondition{
				Type:               batchv1.JobComplete,
				Status:             corev1.ConditionFalse,
				Message:            fmt.Sprintf("the %s phase execution is complete, opening the next %s phase.", curPhase, nextPhse),
				LastTransitionTime: metav1.Now(),
			}
			if nextPhse == evaluationarcadiav1alpha1.CompletePhase {
				d.Status = corev1.ConditionTrue
				d.Message = "evaluation process completed"
				deepCopyInstance.Status.CompletionTime = &d.LastTransitionTime
			}
			deepCopyInstance.Status.Conditions = []batchv1.JobCondition{d}
			err := r.Client.Status().Patch(ctx, deepCopyInstance, client.MergeFrom(instance))
			if err != nil {
				logger.Error(err, "failed to update rag status")
			}
			return err
		}
	}
	job := &batchv1.Job{}
	jobName := evaluation.PhaseJobName(instance, curPhase)
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: instance.Namespace, Name: jobName}, job); err != nil {
		if !k8serrors.IsNotFound(err) {
			logger.Error(err, fmt.Sprintf("checking for the existence of jobs in the %s phase has failed.", curPhase), "jobName", jobName)
			return err
		}

		logger.Info(fmt.Sprintf("start creating %s phase job", curPhase), "jobName", jobName)
		job, err = genJob(instance)
		if err != nil {
			logger.Error(err, "faled to generated %s phase job", curPhase)
			return err
		}
		if err := controllerutil.SetOwnerReference(instance, job, r.Scheme); err != nil {
			logger.Error(err, "set the job's owner failed.", "jobName", jobName)
			return err
		}
		if err := r.Client.Create(ctx, job); err != nil {
			logger.Error(err, fmt.Sprintf("failed to create %s phase job", curPhase), "jobName", jobName)
			deepCopyInstance.Status.Conditions = []batchv1.JobCondition{
				{
					Type:          batchv1.JobFailed,
					Status:        corev1.ConditionTrue,
					Message:       fmt.Sprintf("failed to create %s phase job", curPhase),
					LastProbeTime: metav1.Now(),
				},
			}
			return r.Client.Status().Patch(ctx, deepCopyInstance, client.MergeFrom(instance))
		}
		// job变化比你来得更早?
		deepCopyInstance.Status.Conditions = []batchv1.JobCondition{
			{
				Type:               batchv1.JobComplete,
				Status:             corev1.ConditionFalse,
				Message:            fmt.Sprintf("the %s phase job has been created and is waiting for the job to complete", curPhase),
				LastTransitionTime: metav1.Now(),
			},
		}
		return r.Client.Status().Patch(ctx, deepCopyInstance, client.MergeFrom(instance))
	}

	if job.DeletionTimestamp != nil {
		logger.Info("pvc is being deleted, need to wait for next process", "jobName", jobName)
		return errors.New("job is being deleted, need to wait for next process")
	}
	if *job.Spec.Suspend != instance.Spec.Suspend {
		complete := false
		for _, cond := range job.Status.Conditions {
			if cond.Type == batchv1.JobComplete && cond.Status == corev1.ConditionTrue {
				complete = true
				break
			}
		}
		if !complete {
			logger.Info(fmt.Sprintf("job suspend state switch from %v to %v", *job.Spec.Suspend, instance.Spec.Suspend))
			*job.Spec.Suspend = instance.Spec.Suspend
			return r.Client.Update(ctx, job)
		}
	}

	return errJobNotDone
}

func (r *RAGReconciler) WhenPVCDeleted(pvc *corev1.PersistentVolumeClaim) {
	ctx := context.TODO()
	logger := log.FromContext(ctx, "PVC", pvc.Name, "Namespace", pvc.Namespace)
	for _, owner := range pvc.OwnerReferences {
		if owner.APIVersion == evaluationarcadiav1alpha1.GroupVersion.String() && owner.Kind == "RAG" {
			rag := &evaluationarcadiav1alpha1.RAG{}
			if err := r.Client.Get(ctx, types.NamespacedName{Name: owner.Name, Namespace: pvc.Namespace}, rag); err != nil {
				logger.Error(err, "failed to get rag", "RAG", owner.Name)
				return
			}
			// the pvc was removed and the evaluation process needs to be re-executed
			dp := rag.DeepCopy()
			dp.Status.Conditions = nil
			dp.Status.Phase = ""
			if err := r.Client.Status().Patch(ctx, dp, client.MergeFrom(rag)); err != nil {
				logger.Error(err, "update the status of the rag to initial status failed.", "RAG", owner.Name)
			}
		}
	}
}

func (r *RAGReconciler) WhenJobChanged(job *batchv1.Job) {
	ctx := context.TODO()
	logger := log.FromContext(ctx, "JOB", job.Name, "Namespace", job.Namespace)
	if len(job.Status.Conditions) == 0 {
		logger.Info("job currently has no status changes and does not do anything about it")
		return
	}

	for _, owner := range job.OwnerReferences {
		if owner.APIVersion == evaluationarcadiav1alpha1.GroupVersion.String() && owner.Kind == "RAG" {
			rag := &evaluationarcadiav1alpha1.RAG{}
			if err := r.Client.Get(ctx, types.NamespacedName{Name: owner.Name, Namespace: job.Namespace}, rag); err != nil {
				logger.Error(err, "failed to get rag", "RAG", owner.Name)
				return
			}
			dp := rag.DeepCopy()
			cur := job.Status.Conditions[0]
			for i := 1; i < len(job.Status.Conditions); i++ {
				if job.Status.Conditions[i].LastTransitionTime.After(cur.LastTransitionTime.Time) {
					cur = job.Status.Conditions[i]
				}
			}
			dp.Status.Conditions = []batchv1.JobCondition{cur}
			if err := r.Client.Status().Patch(ctx, dp, client.MergeFrom(rag)); err != nil {
				logger.Error(err, "set the status of a job to rag failure.", "RAG", owner.Name, "Condition", dp.Status.Conditions[0])
			}
		}
	}
}
