# Jupyterlab

[Jupyterlab](https://github.com/jupyterlab/jupyterlab) is a web-based interactive development environment for Jupyter notebooks, code, and data.

## Requirements

- Kubernetes

## Install Jupyterlab

### With Helm

#### 1. Get Repo Info

```shell
helm repo add kubeagi https://kubeagi.github.io/arcadia
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
  version: 0.1.1
  component:
    name: kubeagi.jupyterlab
    # If you have changed the namespace in step 1, you should update this field.
    namespace: kubebb-system
```

### 3. Optionally enable `ingress`

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
  version: 0.1.1
  override:
    set:
    - ingress.enabled=true
    # ingressNodeIP is `172.18.0.2`
    - ingress.host=jupyterlab.172.18.0.2.nip.io
  component:
    name: kubeagi.jupyterlab
    # If you have changed the namespace in step 1, you should update this field.
    namespace: kubebb-system
```

### 3. Optionally enable `persistence`

If you want to enable `persistence` for jupyterlab, you should update field `override` in `ComponentPlan` before install.For example:

```yaml
apiVersion: core.kubebb.k8s.com.cn/v1alpha1
kind: ComponentPlan
metadata:
  name: jupyterlab
  namespace: default
spec:
  approved: true
  name: jupyterlab
  version: 0.1.1
  override:
    set:
    - persistence.enable=true
  component:
    name: kubeagi.jupyterlab
    # If you have changed the namespace in step 1, you should update this field.
    namespace: kubebb-system
```

### 4. Optionally set `hashed_password`

If you want to set `hashed_password` for jupyterlab, you should update field `override` in `ComponentPlan` before install.For example:

```yaml
apiVersion: core.kubebb.k8s.com.cn/v1alpha1
kind: ComponentPlan
metadata:
  name: jupyterlab
  namespace: default
spec:
  approved: true
  name: jupyterlab
  version: 0.1.1
  override:
    set:
    - conf.hashed_password="{hash_of_your_password}}"
  component:
    name: kubeagi.jupyterlab
    # If you have changed the namespace in step 1, you should update this field.
    namespace: kubebb-system
```

To get a hashed password, you can use the following this [guide](https://jupyter-server.readthedocs.io/en/latest/operators/public-server.html#preparing-a-hashed-password)

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
| `conf.hashed_password` | The hashed password of jupyterlab | hashed of `Passw0rd!` |
| `persistence.enabled` | Enable persistence of jupyterlab data | `false` |
| `persistence.mountPath` | Mount the persistent volume at this path in container | `/home//home/jovyan` |
| `persistence.existingClaim` | Use existing claimed volume | `""` |
| `persistence.storageClass` | Storage class which will be used to claim pv | `""` |
| `persistence.accessMode` | Access mode of volume claim | `"ReadWriteOnce"` |
| `persistence.size` | Size of this volume | `"5Gi"` |
