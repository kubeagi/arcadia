# Data Processing 

## Current Version Main Features

Data Processing is used for data processing through MinIO, databases, Web APIs, etc. The data types handled include:
- txt
- json  
- doc
- html
- excel
- csv
- pdf
- markdown
- ppt

### Current Text Type Processing  

The data processing process includes: cleaning abnormal data, filtering, de-duplication, and anonymization.

## Design

![Design](../assets/data_process.drawio.png)

## Local Development
### Software Requirements

Before setting up the local data-process environment, please make sure the following software is installed:

- Python 3.10.x

### Environment Setup

Install the Python dependencies in the requirements.txt file

### Running

Run the server.py file in the data_manipulation directory

# isort
isort is a tool for sorting imports alphabetically within your Python code. It helps maintain a consistent and clean import order. 

## install
```shell
pip install isort
```

## isort a file
```shell
isort server.py
```

## isort a directory
```shell
isort data_manipulation
```


# config.yml
## dev phase
The example config.yml is as the following:
```yaml
minio:
  access_key: '${MINIO_ACCESSKEY: hpU4SCmj5jixxx}'
  secret_key: '${MINIO_SECRETKEY: xxx}'
  api_url: '${MINIO_API_URL: 172.22.96.136.nip.io}'
  secure: '${MINIO_SECURE: True}'
  dataset_prefix: '${MINIO_DATASET_PREFIX: dataset}'

llm:
  qa_retry_count: '${LLM_QA_RETRY_COUNT: 100}'

knowledge:
  chunk_size: '${KNOWLEDGE_CHUNK_SIZE: 500}'
  chunk_overlap: '${KNOWLEDGE_CHUNK_OVERLAP: 50}'

backendPg:
  host: '${PG_HOST: localhost}'
  port: '${PG_PORT: 5432}'
  user: '${PG_USER: postgres}'
  password: '${PG_PASSWORD: 123456}'
  database: '${PG_DATABASE: arcadia}'

kubernetes:
  default_config: '${DEFAULT_CONFIG: arcadia-config}'
  pod_namespace: '${POD_NAMESPACE: arcadia}'
```

\${MINIO_ACCESSKEY: hpU4SCmj5jixxx} 

MINIO_ACCESSKEY is the environment variable name. 

hpU4SCmj5jixxx is the default value if the environment variable is not set.


## release phase
The example config.yml is as the following:
```yaml
minio:
  access_key: 'hpU4SCmj5jixxx'
  secret_key: 'minio_sk'
  api_url: '172.22.96.136.nip.io'
  secure: 'True'
  dataset_prefix: 'dataset'

llm:
  qa_retry_count: '100'

knowledge:
  chunk_size: '500'
  chunk_overlap: '50'

backendPg:
  host: 'localhost'
  port: '5432'
  user: 'admin'
  password: '123456'
  database: 'arcadia'

kubernetes:
  default_config: 'arcadia-config'
  pod_namespace: 'arcadia'
```
In the K8s, you can use the config map to point to the /arcadia_app/data_manipulation/config.yml file.
