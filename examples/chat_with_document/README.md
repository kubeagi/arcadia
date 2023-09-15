# chat with document API server sample

an API server sample for load and chat with your own document.

## Build from local code

1. Clone `arcadia`

    ```shell
    git clone https://github.cm/kubeagi/arcadia.git
    ```

2. Build

    ```shell
    go build -o bin/chat_with_document examples/chat_with_document/*.go
    ```

3. Start the API server

    ```shell
    $ ./bin/chat_with_document start -h
   Start the server

    Usage:
    chat start [usage] [flags]

    Flags:
    --addr string           used to listen and serve GET request (default ":8800")
    --apikey string        used to connect to ZhiPuAI platform
    -h, --help              help for start
    --namespace string     the vector database namespace (default "arcadia")
    --vector-store string   the chromaDB vector database url
   
   $ ./bin/chat_with_document start --apikey [YOUR_API_KEY] --vector-store [YOUR_CHROMADB_URL]
   Starting chat server example...
    Connecting platform...
    Connecting vector database...
    Heartbeat: &{200 OK 200 HTTP/1.1 1 1 map[Content-Length:[44] Content-Type:[application/json] Date:[Thu, 14 Sep 2023 08:08:04 GMT] Server:[uvicorn]] {{"nanosecond heartbeat":1694678884430848151}} 44 [] false false map[] 0xc000386200 <nil>}
    Creating HTTP server...
    
    ┌───────────────────────────────────────────────────┐
    │                    chat-server                    │
    │                   Fiber v2.49.1                   │
    │               http://127.0.0.1:8800               │
    │       (bound on host 0.0.0.0 and port 8800)       │
    │                                                   │
    │ Handlers ............. 5  Processes ........... 1 │
    │ Prefork ....... Disabled  PID ............. 30984 │
    └───────────────────────────────────────────────────┘
    ```

## Usage

### Load document into vector store

Example:
```shell
curl --request POST \
  --url http://localhost:8800/load \
  --header 'Content-Type: application/json' \
  --data '{
    "document": "KubeAGI 是 KubeBB 下的项目，致力于将大语言模型与 KubeBB 结合，助力开发者及 K8s 生态发展。",
    "chunk-size": 2048,
    "chunk-overlap": 128
}'
```

#### URL

- `POST /load`

#### Parameter

| Name          | Must have | Type   | Description                                    |
|---------------|-----------|--------|------------------------------------------------|
| document      | Yes       | string | content of the document                        |
| chunk-size    | No        | int    | size of the split documents, default is 2048   |
| chunk-overlap | No        | int    | overlap of the split documents, default is 128 |

#### Request Body

```json
{
    "document": "KubeAGI 是 KubeBB 下的项目，致力于将大语言模型与 KubeBB 结合，助力开发者及 K8s 生态发展。",
    "chunk-size": 2048,
    "chunk-overlap": 128
}
```

#### Response

```json
{
    "status": "OK"
}
```

### Chat with document

Example:

```shell
curl --request POST \
  --url http://localhost:8800/chat \
  --header 'Content-Type: application/json' \
  --data '{
    "content": "什么是KubeAGI？"
}'
```

#### URL

- `POST /chat`

#### Parameter

| Name    | Must have | Type   | Description  |
|---------|-----------|--------|--------------|
| content | Yes       | string | chat content |

#### Request Body

```json
{
    "content": "什么是KubeAGI？"
}
```

#### Response

```json
{
  "code":200,
  "data":{
    "request_id":"7936738270392008763",
    "task_id":"7936738270392008763",
    "task_status":"SUCCESS",
    "usage":{
      "total_tokens":189
    },
    "choices":[
      {
        "content":"KubeAGI 是一个项目，它是 KubeBB 的一部分，旨在将大语言模型与 KubeBB 相结合，以支持开发者和 K8s 生态系统的发展。KubeBB 是一个用于构建 Kubernetes 应用程序的平台，它提供了三个套件：内核 Kit、开放组件市场和底座 Kit。内核 Kit 提供声明式的组件生命周期管理和组件市场，并通过 Tekton 流水线强化低代码平台组件与底座服务的集成。开放组件市场是内核能力的 productization，作为适配底座服务的组件发布到官方组件仓库中使用，扩展 KubeBB 生态。底座 Kit 通过集成各种组件提供统一的认证中心和门户入口，包括 Low-Code Engine 和具有 Git 特性的关系数据库 Dolt。借助底座门户的菜单和路由资源和内核套件的组件管理能力，实现组件开发、测试到上线的全链路能力。",
        "role":"assistant"
      }
    ]},
  "msg":"操作成功",
  "success":true
}    
```