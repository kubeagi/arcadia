This document will introduce how to install Arcadia on a Kubernetes cluster.

## Install and Run Arcadia

#### Create or Use existing Kubernetes Cluster

Firstly you need to have a Kubernetes cluster, and a `kubectl` command-line tool must be configured to communicate with the cluster.

The Kubernetes version must equal to or greater than **v1.20.0**. To check the version, use `kubectl version --short`.

If you do not have a cluster yet, create one by using one of the following provision tools:

* [kind](https://kind.sigs.k8s.io/docs/)
* [kubeadm](https://kubernetes.io/docs/admin/kubeadm/)
* [minikube](https://minikube.sigs.k8s.io/)

#### Install arcadia using helm
1. Clone the code repoistory
```shell
git clone https://github.com/kubeagi/arcadia.git
``` 

2. Get required images

Container images are available in the Docker Hub. You can pull and push to your image registry or use them directly if your cluster can access docker hub.
```
# image of core controller
docker pull kubeagi/arcadia-controller:v0.1.0
# image of data-processing
docker pull kubeagi/data-processing:v0.1.0
# image of fastchat and its workers
docker pull kubeagi/fastchat:v0.1.0
docker pull kubeagi/fastchat-worker:v0.1.0
# image of minio
docker pull kubeagi/minio:RELEASE.2023-02-10T18-48-39Z
# image of minio client to download objects
docker pull kubeagi/minio-mc:RELEASE.2023-02-10T18-48-39Z
# image of bff-server, contains apiserver and portal
docker pull kubeagi/kubeagi-portal:v0.1.0
# image of vector database using chroma, to be replaced by pgvector
docker pull kubeagi/chroma:0.4.14
# image of streamlit, only used during runtime(not by helm)
docker pull kubeagi/streamlit:v1.29.0

```
1. Preparation

* Install kubebb following the document below

[install kubebb](http://kubebb.k8s.com.cn/docs/quick-start/quick-install)

* Install postgresql and create user, password and a database with name 'arcadia'

1）Use an existing postgresql or use docker to create a new one.

2）Use component management of kubebb to install postgresql, refer [repository management](http://kubebb.k8s.com.cn/docs/user-guid/repository_anagement)  

2. Create the namespace to install arcadia
```shell
cd deploy/charts/arcadia
# create the namespace
kubectl create ns kubeagi-system
```

3. Edit the values before run helm install, we just list the required values to update below:

* Update \<replaced-ingress-nginx-ip\> to the ip of ingress nginx
* Update values of apiserver.oidc to use the configuration of kubebb's u4a-component
* Update minio.rootUser, minio.rootPassword, minio.persistence.storageClass
* Update the values under dataprocess.postgres

For other values, update as you need.

```yaml
# NOTE: values that don't need to show was removed from yaml here

# graphql and bff server
apiserver:
  oidc:
    enabled: true
    clientID: bff-client
    issuerURL: https://portal.<replaced-ingress-nginx-ip>.nip.io/oidc
    clientSecret: 61324af0-1234-4f61-b110-ef57013267d6
    masterURL: https://k8s.<replaced-ingress-nginx-ip>.nip.io

minio:
  rootUser: "admin"
  rootPassword: "Passw0rd!"
  persistence:
    enabled: true
    storageClass: "openebs-hostpath"
    size: 30Gi

dataprocess:
  env:
    minio:
      apiURL: http://arcadia-minio:9000
    knowledge:
      chunkSize: 500
      chunkOverlap: 50
    postgres:
      # update to the actual values
      host: postgresql
      port: 5432
      user: admin
      password: Passw0rd!
      database: arcadia
```

4. Run helm install to install the resources and bring arcadia up

```shell
# use --dry-run to test the yaml resources to be deployed
helm install arcadia -n kubeagi-system --dry-run .
# install using helm
helm install arcadia -n kubeagi-system .
# check if all pod will be ready
kubectl get pods -n kubeagi-system
```

5. Complete the post installation steps

* Configure data processing service
```
TODO
```

* Start run your first LLM application *[run llm application](./run-llm-application.md)*

6. Access the kubeagi Portal

Visit https://portal.<replaced-ingress-nginx-ip>.nip.io and use proper user to login, same as how kubebb works.

### Some notes
1. Expose minio console to visit your data, you can enable *minio.ingress.console* and access the console from https://minio-console.<replaced-ingress-nginx-ip>.nip.io

2. Minio API will be exposed, as *minio.ingress.api* will be enabled by default. And files will be uploaded using MinIO API directly. API address is https://minio-api.<replaced-ingress-nginx-ip>.nip.io

## Uninstall

1. Remove arcadia

   ```
   $ helm uninstall arcadia -n kubeagi-system 
   ```
