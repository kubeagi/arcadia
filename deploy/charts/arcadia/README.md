# Arcadia

This Helm chart installs arcadia into Kubernetes.

## **Requirements**

- Kubernetes
- Kubebb

## Configuration Definitions @ Values.yaml

**Before deploying arcadia chart, you should update values defined in `value.yaml`. **

### global

global settings of arcadia chart.

| Parameter                | Description                                                  | Default     |
| ------------------------ | ------------------------------------------------------------ | ----------- |
| `oss.bucket`             | Name of the bucket where data is stored                      | `"arcadia"` |
| `defaultVectorStoreType` | Defines the default vector database type, currently `chroma` and `pgvector` are available | `pgvector`  |

### controller

configs of the core controller for arcadia

| Parameter           | Description                     | Default                  |
| ------------------- | ------------------------------- | ------------------------ |
| `loglevel`          | klog level of controller pod    | `3`                      |
| `image`             | image of controller pod         | `kubeagi/arcadia:latest` |
| `imagePullPolicy`   | pull policy of controller image | `IfNotPresent`           |
| `resources.`        | resource used by controller pod |                          |
| `- limits.cpu`      |                                 | `"1"`                    |
| `- limits.memory`   |                                 | `1024Mi`                 |
| `- requests.cpu`    |                                 | `10m`                    |
| `- requests.memory` |                                 | `64Mi`                   |

### apiserver

graphql and bff server

| Parameter           | Description                  | Default                                                  |
| ------------------- | ---------------------------- | -------------------------------------------------------- |
| `bingKey`           |                              |                                                          |
| `loglevel`          | klog level of apiserver pod  | 3                                                        |
| `image`             | image of apiserver pod       | kubeagi/arcadia:latest                                   |
| `enableplayground`  | enable playground            | `false`                                                  |
| `port`              | port of apiserver            | `8081`                                                   |
| `ingress.enabled`   | enable ingress for apiserver | `true`                                                   |
| `ingress.path`      | path of apiserver            | `kubeagi-apis`                                           |
| `ingress.host`      | host of apiserver            | `portal.<replaced-ingress-nginx-ip>.nip.io`              |
| `oidc.enabled`      | enable oidc certification    | `true`                                                   |
| `oidc.clientID`     | oidc client ID               | `bff-client`                                             |
| `oidc.clientSecret` | oidc client Secret           |                                                          |
| `oidc.issuerURL`    | URL of issuer portal         | `https://portal.<replaced-ingress-nginx-ip>.nip.io/oidc` |
| `oidc.masterURL`    | URL of master                | `https://k8s.<replaced-ingress-nginx-ip>.nip.io`         |

### portal

portal for arcadia web console

| Parameter       | Description                       | Default                                     |
| --------------- | --------------------------------- | ------------------------------------------- |
| `enabled`       | enable arcadia web portal console | `true`                                      |
| `kubebbEnabled` | enable kubebb platform            | `true`                                      |
| `image`         | image of web console pod          | `kubeagi/ops-console:latest`                |
| `port`          | port of web console               | `80`                                        |
| `ingress.path`  | ingress path of portal            | `kubeagi-portal-public`                     |
| `ingress.host`  | host of ingress path              | `portal.<replaced-ingress-nginx-ip>.nip.io` |

### fastchat

fastchat is used as LLM serve platform for arcadia

| Parameter          | Description                       | Default                                           |
| ------------------ | --------------------------------- | ------------------------------------------------- |
| `enabled`          | enable fastchat pod               | `true`                                            |
| `image.repository` | image of fastchat pod             | `kubeagi/arcadia-fastchat`                        |
| `image.tag`        | tag of fastchat image             | `v0.2.0`                                          |
| `ingress.enabled`  | enable ingress of fastchat server | `true`                                            |
| `ingress.host`     | host of fastchat server           | `fastchat-api.<replaced-ingress-nginx-ip>.nip.io` |

### minio

minio is used as default Object-Storage-Service for arcadia.

| Parameter                  | Description                                                  | Default                                                      |
| -------------------------- | ------------------------------------------------------------ | ------------------------------------------------------------ |
| `image.repository`         | image of minio pod                                           | `kubeagi/minio`                                              |
| `image.tag`                | tag of the minio image                                       | `RELEASE.2023-02-10T18-48-39Z`                               |
| `mode`                     | minio running mode                                           | `standalone`                                                 |
| `rootUser`                 | root user name of minio                                      | `"admin"`                                                    |
| `rootPassword`             | root password of minio                                       | `"Passw0rd!"`                                                |
| `persistence.enabled`      | enable persistant storage for minio service                  | `true`                                                       |
| `persistence.storageClass` | class of persistant storage                                  | `"standard"`                                                 |
| `persistence.size`         | size of persistant storage                                   | `30Gi`                                                       |
| `ingress.enabled`          | enable ingress of minio service                              | `true`                                                       |
| `ingress.api.enabled`      | enable ingress of minio api                                  | `true`                                                       |
| `ingress.api.insecure`     | set if api can be accessed insecurely                        | `false`                                                      |
| `ingress.api.port`         | port of minio api                                            | `9000`                                                       |
| `ingress.api.host`         | host of minio api                                            | `minio-api.<replaced-ingress-nginx-ip>.nip.io`               |
| `ingress.console.enabled`  | enable ingress of minio console                              | `false`                                                      |
| `ingress.console.port`     | port of minio console                                        | `9001`                                                       |
| `ingress.console.host`     | host of minio console                                        | `minio-console.<replaced-ingress-nginx-ip>.nip.io`           |
| `ingress.cert.ipAddresses` | IP address                                                   | `<replaced-ingress-nginx-ip>`                                |
| `ingress.cert.dnsNames`    | Names of certified DNSes.                                    | `minio-api.<replaced-ingress-nginx-ip>.nip.io`<br />`minio-console.<replaced-ingress-nginx-ip>.nip.io` |
| `buckets.name`             | Name of the bucket                                           | `*default-oss-bucket`                                        |
| `buckets.policy`           | Policy to be set on the bucket.[none\|download\|upload\|public\|custom] if set to custom, customPolicy must be set. | `"none"`                                                     |
| `buckets.versioning`       | set versioning for bucket [true\|false]                      | `false`                                                      |
| `buckets.objectlocking`    | set objectlocking for bucket [true\|false]  NOTE: versioning is enabled by default if you use locking. | `false`                                                      |

### dataprocess

configs of data processing service

| Parameter                   | Description               | Default                          |
| --------------------------- | ------------------------- | -------------------------------- |
| `enabled`                   | enable dataprocess pod    | `true`                           |
| `image`                     | image of dataprocess pod  | `kubeagi/data-processing:latest` |
| `port`                      | port of dataprocess       | `28888`                          |
| `config.llm.qa_retry_count` | retry limit of QA process | `'2'`                            |

### postgresql

configs of postgresql service. Posgresql service will be used in: dataprocessing, llm application, and is used as vector store with pgvector enabled(Recommended).

| Parameter                         | Description                           | Default                                |
| --------------------------------- | ------------------------------------- | -------------------------------------- |
| `enabled`                         | enable postgresql                     | `true`                                 |
| `global.storageClass`             | storage class of postgresql           | `"standard"`                           |
| `global.postgresql.auth`          | default auth settrings of postgresql  |                                        |
| `- .username`                     | default username                      | `"admin"`                              |
| `- .password`                     | default password                      | `"Passw0rd!"`                          |
| `- .database`                     | default database                      | `"arcadia"`                            |
| `image.registry`                  | postgresql image registry             | `docker.io`                            |
| `image.repository`                | postgresql image repo name            | `kubeagi/postgresql`                   |
| `image.tag`                       | postgresql image repo tag             | `16.1.0-debian-11-r18-pgvector-v0.5.1` |
| `image.pullPolicy`                | postgresql image pull policy          | `IfNotPresent`                         |
| `primary.initdb.scriptsConfigMap` | config map when initializing database | `pg-init-data`                         |

### chromadb

configs to deploy chromadb instance

| Parameter                         | Description                    | Default            |
| --------------------------------- | ------------------------------ | ------------------ |
| `enabled`                         | enable chromadb instance       | `false`            |
| `image.repository`                | chromadb image repo name       | `kubeagi/chromadb` |
| `chromadb.apiVersion`             | chromadb api version           | `"0.4.18"`         |
| `chromadb.auth.enabled`           | enable chromadb auth           | `false`            |
| `chromadb.serverHttpPort`         | chromadb server port           | `8000`             |
| `chromadb.dataVolumeStorageClass` | class of chromadb data storage | `"standard"`       |
| `chromadb.dataVolumeSize`         | size of chromadb data storage  | `"1Gi"`            |

### ray

ray is a unified framework for scaling AI and Python applications. In kubeagi, we use ray for distributed inference. For more information on cluster configurations, please refer to http://kubeagi.k8s.com.cn/docs/Configuration/DistributedInference/run-inference-using-ray.

| Parameter         | Description                               | Default / Sample                                      |
| ----------------- | ----------------------------------------- | ----------------------------------------------------- |
| `clusters`        | lists of GPU clusters used for inference. |                                                       |
| ` - .name`        | name of GPU cluster                       | `3090-2-GPUs`                                         |
| `- .headAddress`  | head address of ray cluster               | `raycluster-kuberay-head-svc.kuberay-system.svc:6379` |
| `- pythonVersion` | python version of ray cluster             | `3.9.18`                                              |
| `- dashboardHost` | dashboard host of ray cluster             | `raycluster-kuberay-head-svc.kuberay-system.svc:8265` |
