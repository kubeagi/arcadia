/*
Copyright 2023 The KubeAGI Authors.

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

package streamlit

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubeagi/arcadia/pkg/config"
)

const (
	StreamlitInstalledAnnotation = "kubeagi.k8s.com.cn/streamlit.installed"
	StreamlitAppName             = "streamlit-app"
)

type StreamlitDeployer struct {
	ctx    context.Context
	client client.Client

	namespace *corev1.Namespace
}

func NewStreamlitDeployer(ctx context.Context, client client.Client, instance *corev1.Namespace) *StreamlitDeployer {
	return &StreamlitDeployer{ctx: ctx, client: client, namespace: instance}
}

func (st *StreamlitDeployer) Install() error {
	// Check if streamlit already installed
	exist, _ := st.streamlitInstalled()
	if exist {
		klog.Info(("Streamlit app already exists"))
		return nil
	}
	// begin to install
	replicas := int32(1)
	containerPort := int32(8501)
	appLabel := map[string]string{"app": StreamlitAppName}
	quantity, _ := resource.ParseQuantity("10Gi")

	namespace := st.namespace.Name
	// lookup streamlit image from config
	streamlitConfig, err := config.GetStreamlit(st.ctx, st.client)
	if err != nil {
		klog.Errorln("failed to get streamlit config", err)
		return err
	}

	// 1. Create the service first
	streamlitService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      StreamlitAppName,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: containerPort,
					TargetPort: intstr.IntOrString{
						Type:   0,
						IntVal: containerPort,
					},
				},
			},
			Selector: appLabel,
		},
	}
	err = st.client.Create(st.ctx, &streamlitService)
	if err != nil {
		klog.Errorln("failed to create streamlit service", err)
		return err
	}

	// 2. Create a PVC to hold all streamlit pages
	pvc := corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      StreamlitAppName,
			Namespace: namespace,
			Labels: map[string]string{
				"app": StreamlitAppName,
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: quantity,
				},
			},
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
		},
	}
	err = st.client.Create(st.ctx, &pvc)
	if err != nil {
		klog.Errorln("failed to create streamlit pvc", err)
		return err
	}
	// 3. Create the deployment
	streamlitImage := streamlitConfig.Image
	streamlitDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      StreamlitAppName,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: appLabel,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: appLabel,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  StreamlitAppName,
							Image: streamlitImage,
							Env: []corev1.EnvVar{
								{
									Name:  "STREAMLIT_UI_HIDE_SIDEBAR_NAV",
									Value: "true",
								},
								{
									Name:  "STREAMLIT_UI_HIDE_TOP_BAR",
									Value: "true",
								},
								{
									Name:  "STREAMLIT_SERVER_BASE_URL_PATH",
									Value: fmt.Sprintf("%s/%s", streamlitConfig.ContextPath, namespace),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "pages",
									MountPath: "/app/pages",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: containerPort,
									Protocol:      "TCP",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "pages",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: pvc.ObjectMeta.Name,
								},
							},
						},
					},
				},
			},
		},
	}
	err = st.client.Create(st.ctx, &streamlitDeployment)
	if err != nil {
		klog.Errorln("failed to create streamlit deployment", err)
		return err
	}

	// 4. Create the ingress to expose the streamlit app
	ingressClassName := &streamlitConfig.IngressClassName
	pathType := networkv1.PathTypePrefix

	streamlitIngress := networkv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      StreamlitAppName,
			Namespace: namespace,
		},
		Spec: networkv1.IngressSpec{
			IngressClassName: ingressClassName,
			Rules: []networkv1.IngressRule{
				{
					Host: streamlitConfig.Host,
					IngressRuleValue: networkv1.IngressRuleValue{
						HTTP: &networkv1.HTTPIngressRuleValue{
							Paths: []networkv1.HTTPIngressPath{
								{
									// Ingress path should be the same as streamlit context path
									Path:     fmt.Sprintf("%s/%s", streamlitConfig.ContextPath, namespace),
									PathType: &pathType,
									Backend: networkv1.IngressBackend{
										Service: &networkv1.IngressServiceBackend{
											Name: StreamlitAppName,
											Port: networkv1.ServiceBackendPort{
												Number: containerPort,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	err = st.client.Create(st.ctx, &streamlitIngress)
	if err != nil {
		klog.Errorln("failed to create streamlit ingress", err)
		return err
	}
	return nil
}

func (st *StreamlitDeployer) Uninstall() error {
	// Check if streamlit already installed
	exist, _ := st.streamlitInstalled()
	if !exist {
		klog.V(5).Infoln("Streamlit app does not exist, skip uninstall")
		return nil
	}
	// begin to uninstall
	// 1. Delete ingress
	ingress := &networkv1.Ingress{}
	err := st.client.Get(st.ctx, client.ObjectKey{Namespace: st.namespace.Name, Name: StreamlitAppName}, ingress)
	if err != nil && !errors.IsNotFound(err) {
		klog.Errorln("failed to get streamlit ingress", err)
		return err
	}
	if err == nil {
		err = st.client.Delete(st.ctx, ingress, &client.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			klog.Errorln("failed to uninstall streamlit ingress", err)
			return err
		}
	}

	// 2. Delete PVC
	pvc := &corev1.PersistentVolumeClaim{}
	err = st.client.Get(st.ctx, client.ObjectKey{Namespace: st.namespace.Name, Name: StreamlitAppName}, pvc)
	if err != nil && !errors.IsNotFound(err) {
		klog.Errorln("failed to get streamlit pvc", err)
		return err
	}
	if err == nil {
		err = st.client.Delete(st.ctx, pvc, &client.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			klog.Errorln("failed to uninstall streamlit pvc", err)
			return err
		}
	}

	// 3. Delete deployment
	deployment := &appsv1.Deployment{}
	err = st.client.Get(st.ctx, client.ObjectKey{Namespace: st.namespace.Name, Name: StreamlitAppName}, deployment)
	if err != nil && !errors.IsNotFound(err) {
		klog.Errorln("failed to get streamlit deployment", err)
		return err
	}
	if err == nil {
		err = st.client.Delete(st.ctx, deployment, &client.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			klog.Errorln("failed to uninstall streamlit deployment", err)
			return err
		}
	}

	// 4. Delete service
	service := &corev1.Service{}
	err = st.client.Get(st.ctx, client.ObjectKey{Namespace: st.namespace.Name, Name: StreamlitAppName}, service)
	if err != nil && !errors.IsNotFound(err) {
		klog.Errorln("failed to get streamlit service", err)
		return err
	}
	if err == nil {
		err = st.client.Delete(st.ctx, service, &client.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			klog.Info("failed to uninstall streamlit service")
			return err
		}
	}

	return nil
}

// Check if streamlit app already installed
func (st *StreamlitDeployer) streamlitInstalled() (bool, error) {
	streamlitService := corev1.Service{}
	err := st.client.Get(st.ctx, types.NamespacedName{Namespace: st.namespace.Name, Name: StreamlitAppName}, &streamlitService)
	if err == nil && streamlitService.ObjectMeta.Name != "" {
		return true, nil
	}
	return false, err
}
