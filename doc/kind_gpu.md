# Schedule a Kind cluster with GPU Enabled

`Kind` do not support to utilize GPU by default,but fortunately there is a [**Workaround**](https://github.com/kubernetes-sigs/kind/pull/3257#issuecomment-1607287275) found by amazing guys. 

This document shows how to :

1. Create the kind cluster which can utilize GPU
2. Install [`Nvidia GPU-Operator`](https://github.com/NVIDIA/gpu-operator) to manage gpus in kubernetes cluster

## Pre-requisites

- [Docker](https://www.docker.com/)
- [Nvidia Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html)

## Create the kind cluster

1. Configure docker to use nvidia runtime

Edit `/etc/docker/daemon.json` to configure default docker runtime

```json
{
    "default-runtime": "nvidia",
    "runtimes": {
        "nvidia": {
            "args": [],
            "path": "nvidia-container-runtime"
        }
    }
}
```

Restart docker when docker runtime updated

```shell
sudo systemctl restart docker
```

2. Configure nvidia container runtime

set `accept-nvidia-visible-devices-as-volume-mounts = true` in `/etc/nvidia-container-runtime/config.toml`

3. Install Kind CLI

Follow [document](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) to install **kind** to your local machine

4. Create kind cluster `kubeagi`

At the root directory of [`kubeagi/arcadia`](https://github.com/kubeagi/arcadia).

Run command:

```shell
make kind
```

When kind created,link `ldconfig` to `ldconfig.real` in kind container.()

```shell
docker exec -ti kubeagi-control-plane ln -s /sbin/ldconfig /sbin/ldconfig.real
```

5. Install Nvidia GPU-Operator

To make it more easy to install `GPU Nvidia GPU-Operator` in kubernetes,we can use another tool [kubebb core](https://github.com/kubebb/core) to:

```shell
helm repo add nvidia https://helm.ngc.nvidia.com/nvidia
helm repo update
helm install --generate-name \
     -n gpu-operator --create-namespace \
     nvidia/gpu-operator --set driver.enabled=false
```

Check the installation status:

```shell
kubectl get pods -ngpu-operator --watch
```