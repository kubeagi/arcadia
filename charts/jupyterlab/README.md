# Jupyterlab

[Jupyterlab](https://github.com/jupyterlab/jupyterlab) is a web-based interactive development environment for Jupyter notebooks, code, and data.

## Requirements

- Kubernetes

## Install Jupyterlab

### With Helm

#### 1. Get Repo Info

```shell
helm repo add kubeagi https://kubeagi.github.io/charts
helm repo update
```

#### 2. Install Jupyterlab

```shell
helm install [RELEASE_NAME] kubeagi/jupyterlab
```

**If you want to enable `ingress` for Jupyterlab, you should update field `ingress` in `values.yaml` before install.**

### With Kubebb Core

#### 1. Create a repository `kubeagi` into kubebb

```shell
kubectl apply -f https://raw.githubusercontent.com/kubeagi/arcadia/main/.kubeagi_repo.yaml
```

> Note: If you want to create this repository in other namespace, you should update field `metadata.namespace` in `.kubeagi_repo.yaml` before apply.

#### 2. Install Jupyterlab with `ComponentPlan`

```shell
apiVersion: core.kubebb.k8s.com.cn/v1alpha1
kind: ComponentPlan
metadata:
  name: jupyterlab
  namespace: default
spec:
  approved: true
  name: jupyterlab
  version: 0.1.0
  component:
    name: kubeagi.jupyterlab
    # If you have changed the namespace in step 1, you should update this field.
    namespace: kubebb-system
```

If you want to enable `ingress` with the help of `u4a-component (IngressNodeIP is 172.18.0.2)`, you should update field `override` in `ComponentPlan` before install.For example:

```yaml
apiVersion: core.kubebb.k8s.com.cn/v1alpha1
kind: ComponentPlan
metadata:
  name: jupyterlab
  namespace: default
spec:
  approved: true
  name: jupyterlab
  version: 0.1.0
  override:
    set:
    - ingress.enabled=true
    # ingressNodeIP is `172.18.0.2`
    - ingress.hosts[0].host=jupyterlab.172.18.0.2.nip.io
  component:
    name: kubeagi.jupyterlab
    # If you have changed the namespace in step 1, you should update this field.
    namespace: kubebb-system
```

## Configuration

The following table lists the important configurable parameters of the Jupyterlab chart and their default values.

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `nameOverride` | Override the name of the chart | `""` |
| `image.repository` | Image repository | `jupyter/tensorflow-notebook` |
| `image.tag` | Image tag | `lab-4.0.3` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |
| `imagePullSecrets` | Image pull secrets to access image registry | `[]` |
| `ingress.enabled` | Enable ingress | `false` |
| `ingress.className` | Ingress class name | `"portal-ingress"` |
| `ingress.host` | Set Ingress host | `jupyterlab.172.18.0.2.nip.io` (Must update this to real ingress node ip if `ingress.enabled` is `true`)|
