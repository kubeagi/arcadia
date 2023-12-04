# FastChat LLM-Worker

**Deprecated as we manage the llm-worker in arcadia operator now**

## Requirements

- Kubernetes

## Installation

### With Helm

#### 1. Clone Repo
```shell
helm repo add arcadia https://kubeagi.github.io/arcadia
helm repo update
```

#### 2. Install FastChat

```shell
helm install [RELEASE_NAME] arcadia/llm-worker
```

## Parameters

### 1. MinIO

```yaml
    - name: MINIO_ENDPOINT
      value: "your_minio_endpoint"
    - name: MINIO_ACCESS_KEY
      value: "your_minio_access_key"
    - name: MINIO_SECRET_KEY
      value: "your_minio_secret_key"
    - name: MINIO_MODEL_BUCKET_PATH
      value: "path/to/your/minio/model"
```


### 2. FastChat

```yaml
    - name: FASTCHAT_WORKER_NAME
      value: "your_worker_instance_name"            # default "baichuan2-7b-instance-1"
    - name: FASTCHAT_WORKER_MODEL_NAME
      value: "your_model_name"                      # default "baichuan2-7b"
    - name: FASTCCHAT_WORKER_ADDRESS
      value: "defined_worker_k8s_service_address:21002"
    - name: FASTCCHAT_CONTROLLER_ADDRESS
      value: "your_fastchat_controller_address:21001"
```