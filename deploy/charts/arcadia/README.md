# Arcadia

This Helm chart installs arcadia into Kubernetes.

## **Requirements**

- Kubernetes
- Kubebb

## Configuration Definitions @ Values.yaml

**Before deploying arcadia chart, you should update values defined in `value.yaml`.**

### global

global settings of arcadia chart.

| Parameter                | Description                                                  | Default     |
| ------------------------ | ------------------------------------------------------------ | ----------- |
| `global.storage.class`          | Defines the default storage class for arcadia components,like `minio` `postgresql` `chroma` | `standard`  |
| `global.defaultVectorStoreType` | Defines the default vector database type, currently `chroma` and `pgvector` are available | `pgvector`  |
| `global.hostConfig`             | Defines the default host config for arcadia deployments      | hostnames with almost all the ingress hosts  |

### config

configs of the operation console for arcadia

| Parameter                | Description                                                  | Default     |
| ------------------------ | ------------------------------------------------------------ | ----------- |
| `config.embedder.enabled`  | 标识是否启用系统默认的嵌入服务	                                   | `false`      |
| `config.embedder.model`	  | 指定作为系统默认嵌入服务使用的模型名称	                             | `bge-large-zh-v1.5` |
| `config.rerank.enabled`	  | 标识是否启用默认的重排序服务	|  true | 
| `config.rerank.model`	   | 设置用于重排序服务的默认模型名称 | 	`bge-reranker-large` | 


### controller

configs of the core controller for arcadia

| Parameter           | Description                     | Default                  |
| ------------------- | ------------------------------- | ------------------------ |
| `controller.loglevel`          | klog level of controller pod. 1: error 3: info  5: debug     | `3`                      |
| `controller.image`             | image of controller pod         | `kubeagi/arcadia:latest` |
| `controller.imagePullPolicy`   | pull policy of controller image | `IfNotPresent`           |
| `controller.resources`        | resource used by controller pod |                          |

### apiserver

graphql and bff server

| Parameter           | Description                  | Default                                                  |
| ------------------- | ---------------------------- | -------------------------------------------------------- |
| `apiserver.loglevel`          | klog level of apiserver pod   1: error 3: info  5: debug | 3                                                        |
| `apiserver.image`             | image of apiserver pod       | kubeagi/arcadia:latest                                   |
| `apiserver.enableplayground`  | enable playground            | `false`                                                  |
| `apiserver.port`              | port of apiserver            | `8081`                                                   |
| `apiserver.ingress.enabled`   | enable ingress for apiserver | `true`                                                   |
| `apiserver.ingress.path`      | path of apiserver            | `kubeagi-apis`                                           |
| `apiserver.ingress.host`      | host of apiserver            | `portal.<replaced-ingress-nginx-ip>.nip.io`              |
| `apiserver.oidc.enabled`      | enable oidc certification    | `true`                                                   |
| `apiserver.oidc.clientID`     | oidc client ID               | `bff-client`                                             |
| `apiserver.oidc.clientSecret` | oidc client Secret           |  `61324af0-1234-4f61-b110-ef57013267d6`                   |
| `apiserver.oidc.issuerURL`    | URL of issuer portal         | `https://portal.<replaced-ingress-nginx-ip>.nip.io/oidc` |
| `apiserver.oidc.masterURL`    | URL of master                | `https://k8s.<replaced-ingress-nginx-ip>.nip.io`         |

### opsconsole

portal for arcadia operation console

| Parameter       | Description                       | Default                                     |
| --------------- | --------------------------------- | ------------------------------------------- |
| `opsconsole.enabled`       | enable arcadia web portal console | `true`                                      |
| `opsconsole.kubebbEnabled` | enable kubebb platform            | `true`                                      |
| `opsconsole.image`         | image of web console pod          | `kubeagi/ops-console:latest`                |
| `opsconsole.ingress.path`  | ingress path of portal            | `kubeagi-portal-public`                     |
| `opsconsole.ingress.host`  | host of ingress path              | `portal.<replaced-ingress-nginx-ip>.nip.io` |

### gpts

configuration for gpt store

| Parameter       | Description                       | Default                                     |
| --------------- | --------------------------------- | ------------------------------------------- |
| `gpts.enabled`       | enable arcadia gpt store       | `true`                     |
| `gpts.public_namespace`       | all gpt resources are public in this namespace      | `true`                     |
| `gpts.agentportal.image`         | image of web console pod          | `kubeagi/agent-portal:latest`                |
| `gpts.agentportal.ingress.path`  | ingress path of agent portal            |      ``                                     |
| `gpts.agentportal.ingress.host`  | host of ingress path for agent portal             | `gpts.<replaced-ingress-nginx-ip>.nip.io` |

### fastchat

fastchat is used as LLM serve platform for arcadia

| Parameter          | Description                       | Default                                           |
| ------------------ | --------------------------------- | ------------------------------------------------- |
| `fastchat.enabled`          | enable fastchat pod               | `true`                                            |
| `fastchat.image.repository` | image of fastchat pod             | `kubeagi/arcadia-fastchat`                        |
| `fastchat.image.tag`        | tag of fastchat image             | `v0.2.0`                                          |
| `fastchat.ingress.enabled`  | enable ingress of fastchat server | `true`                                            |
| `fastchat.ingress.host`     | host of fastchat server           | `fastchat-api.<replaced-ingress-nginx-ip>.nip.io` |

### minio

minio is used as default Object-Storage-Service for arcadia.

| Parameter                  | Description                                                  | Default                                                      |
| -------------------------- | ------------------------------------------------------------ | ------------------------------------------------------------ |
| `minio.image.repository`         | image of minio pod                                           | `kubeagi/minio`                                              |
| `minio.image.tag`                | tag of the minio image                                       | `RELEASE.2023-02-10T18-48-39Z`                               |
| `minio.mode`                     | minio running mode                                           | `standalone`                                                 |
| `minio.rootUser`                 | root user name of minio                                      | `"admin"`                                                    |
| `minio.rootPassword`             | root password of minio                                       | `"Passw0rd!"`                                                |
| `minio.persistence.enabled`      | enable persistant storage for minio service                  | `true`                                                       |
| `minio.persistence.storageClass` | class of persistant storage                                  | `"standard"`                                                 |
| `minio.persistence.size`         | size of persistant storage                                   | `30Gi`                                                       |
| `minio.ingress.enabled`          | enable ingress of minio service                              | `true`                                                       |
| `minio.ingress.api.enabled`      | enable ingress of minio api                                  | `true`                                                       |
| `minio.ingress.api.insecure`     | set if api can be accessed insecurely                        | `false`                                                      |
| `minio.ingress.api.port`         | port of minio api                                            | `9000`                                                       |
| `minio.ingress.api.host`         | host of minio api                                            | `minio-api.<replaced-ingress-nginx-ip>.nip.io`               |
| `minio.ingress.console.enabled`  | enable ingress of minio console                              | `false`                                                      |
| `minio.ingress.console.port`     | port of minio console                                        | `9001`                                                       |
| `minio.ingress.console.host`     | host of minio console                                        | `minio-console.<replaced-ingress-nginx-ip>.nip.io`           |
| `minio.ingress.cert.ipAddresses` | IP address                                                   | `<replaced-ingress-nginx-ip>`                                |
| `minio.ingress.cert.dnsNames`    | Names of certified DNSes.                                    | `minio-api.<replaced-ingress-nginx-ip>.nip.io`<br />`minio-console.<replaced-ingress-nginx-ip>.nip.io` |

### dataprocess

configs of data processing service

| Parameter                   | Description               | Default                          |
| --------------------------- | ------------------------- | -------------------------------- |
| `dataprocess.enabled`                   | enable dataprocess pod    | `true`                           |
| `dataprocess.image`                     | image of dataprocess pod  | `kubeagi/data-processing:latest` |
| `dataprocess.port`                      | port of dataprocess       | `28888`                          |
| `dataprocess.config.llm.qa_retry_count` | retry limit of QA process | `'2'`                            |
| `dataprocess.config.worker` | default pallel workers when generation QA  | `'1'`                            |
| `dataprocess.config.chunkSize` | default chunksize when split document  | `'500'`                            |

### postgresql

configs of postgresql service. Posgresql service will be used in: dataprocessing, llm application, and is used as vector store with pgvector enabled(Recommended).

| Parameter                         | Description                           | Default                                |
| --------------------------------- | ------------------------------------- | -------------------------------------- |
| `postgresql.enabled`                         | enable postgresql                     | `true`                                 |
| `postgresql.global.storageClass`             | storage class of postgresql           | `"standard"`                           |
| `postgresql.global.postgresql.auth`          | default auth settrings of postgresql  |                                        |
| `- .username`                     | default username                      | `"admin"`                              |
| `- .password`                     | default password                      | `"Passw0rd!"`                          |
| `- .database`                     | default database                      | `"arcadia"`                            |
| `postgresql.image.registry`                  | postgresql image registry             | `docker.io`                            |
| `postgresql.image.repository`                | postgresql image repo name            | `kubeagi/postgresql`                   |
| `postgresql.image.tag`                       | postgresql image repo tag             | `16.1.0-debian-11-r18-pgvector-v0.5.1` |
| `postgresql.image.pullPolicy`                | postgresql image pull policy          | `IfNotPresent`                         |
| `postgresql.primary.initdb.scriptsConfigMap` | config map when initializing database | `pg-init-data`                         |

### chromadb

configs to deploy chromadb instance

| Parameter                         | Description                    | Default            |
| --------------------------------- | ------------------------------ | ------------------ |
| `chromadb.enabled`                         | enable chromadb instance       | `false`            |
| `chromadb.image.repository`                | chromadb image repo name       | `kubeagi/chromadb` |
| `chromadb.chromadb.apiVersion`             | chromadb api version           | `"0.4.18"`         |
| `chromadb.chromadb.auth.enabled`           | enable chromadb auth           | `false`            |
| `chromadb.chromadb.serverHttpPort`         | chromadb server port           | `8000`             |
| `chromadb.chromadb.dataVolumeStorageClass` | class of chromadb data storage | `"standard"`       |
| `chromadb.chromadb.dataVolumeSize`         | size of chromadb data storage  | `"1Gi"`            |

### ray

ray is a unified framework for scaling AI and Python applications. In kubeagi, we use ray for distributed inference. For more information on cluster configurations, please refer to http://kubeagi.k8s.com.cn/docs/Configuration/DistributedInference/run-inference-using-ray.

| Parameter         | Description                               | Default / Sample                                      |
| ----------------- | ----------------------------------------- | ----------------------------------------------------- |
| `ray.clusters`        | lists of GPU clusters used for inference. |                                                       |
| ` - .name`        | name of GPU cluster                       | `3090-2-GPUs`                                         |
| `- .headAddress`  | head address of ray cluster               | `raycluster-kuberay-head-svc.kuberay-system.svc:6379` |
| `- pythonVersion` | python version of ray cluster             | `3.9.18`                                              |
| `- dashboardHost` | dashboard host of ray cluster             | `raycluster-kuberay-head-svc.kuberay-system.svc:8265` |
