# arctl

arctl(arcadia command line tool) provides comprehensive tools to help you build and deploy AIGC applications.

## Quick Install

### Pre-requisites

- Go1.20+

```shell
go install github.com/kubeagi/arcadia/arctl@latest
```

Now have a try!❤️ 

```shell
❯ arctl -h
Command line tools for Arcadia

Usage:
  arctl [command]

Available Commands:
  chat        Do LLM chat with similarity search(optional)
  completion  Generate the autocompletion script for the specified shell
  dataset     Manage dataset locally
  help        Help about any command

Flags:
  -h, --help          help for arctl
      --home string   home directory to use (default "/Users/bjwswang/.arcadia")

Use "arctl [command] --help" for more information about a command.
```

### Build from local code 

1. Clone `arcadia`

```shell
git clone https://github.com/kubeagi/arcadia.git
```

2. Build 

```shell
make arctl
```

3. Have a try! ❤️ 

```shell
❯ ./bin/arctl -h
Command line tools for Arcadia

Usage:
  arctl [command]

Available Commands:
  chat        Do LLM chat with similarity search(optional)
  completion  Generate the autocompletion script for the specified shell
  dataset     Manage dataset locally
  help        Help about any command

Flags:
  -h, --help          help for arctl
      --home string   home directory to use (default "/Users/bjwswang/.arcadia")

Use "arctl [command] --help" for more information about a command.
```

## Usage
### Local dataset management

You can use `arctl` to manage your dataset locally with the following commands:

- `arctl dataset list`: list your local datasets
- `arctl dataset create`: create a new local dataset
- `arctl dataset delete`: delete a local dataset and reset the vector store
- `arctl dataset show`: show a local dataset info
- `arctl dataset execute`: execute a local dataset with additional local files

#### Create a local dataset


```shell
❯ arctl dataset create -h
Create dataset

Usage:
  arctl dataset create [usage] [flags]

Flags:
      --chunk-overlap int          chunk overlap for embedding (default 30)
      --chunk-size int             chunk size for embedding (default 300)
      --document-language string   language of the document(Only text,html,csv supported now) (default "text")
      --documents string           path of the documents/document directories to load(separated by comma and directories supported)
  -h, --help                       help for create
      --llm-apikey string          apiKey to access embedding service
      --llm-type string            llm type to use(Only zhipuai,openai supported now) (default "zhipuai")
      --name string                dataset(namespace/collection) of the document to load into
      --text-splitter string       text splitter to use(Only character,token,markdown supported now) (default "character")
      --vector-store string        vector stores to use(Only chroma supported now) (default "http://127.0.0.1:8000")

Global Flags:
      --home string   home directory to use (default "/Users/bjwswang/.arcadia")
```

Required Arguments:
- `--name`: The name of the dataset 
- `--llm-apikey` : The apikey of llm/embedding service
- `--documents`: The documents to load. It is a wrapped file path string with comma. A directory is also supported

For example:
> This will create a local dataset `arcadia` and load documents `./README.md` & `./examples/` into vectorstore(chromadb http://localhost:8000) with help of embedding service `zhipuai` and its apikey `26b2bc55fae40752055cadfc4792f9de.wagA4NIwg5aZJWhm`
```shell
arctl dataset create --name arcadia  --llm-apikey 26b2bc55fae40752055cadfc4792f9de.wagA4NIwg5aZJWhm --documents ./README.md,./examples
I1012 17:53:44.541770    7287 dataset.go:132] Execute dataset: arcadia
I1012 17:53:44.541812    7287 dataset.go:400] Loading document: ./README.md
I1012 17:53:55.552440    7287 dataset.go:413] Time cost 11.01 seconds for loading document: ./README.md
I1012 17:53:55.552705    7287 dataset.go:394] Loading document: examples/chat_with_document/README.md
I1012 17:54:06.595791    7287 dataset.go:413] Time cost 11.04 seconds for loading document: examples/chat_with_document/README.md
I1012 17:54:06.595822    7287 dataset.go:394] Loading document: examples/chat_with_document/handler.go
I1012 17:54:24.256721    7287 dataset.go:413] Time cost 17.66 seconds for loading document: examples/chat_with_document/handler.go
I1012 17:54:24.256749    7287 dataset.go:394] Loading document: examples/chat_with_document/main.go
I1012 17:54:26.003785    7287 dataset.go:413] Time cost 1.75 seconds for loading document: examples/chat_with_document/main.go
I1012 17:54:26.003822    7287 dataset.go:394] Loading document: examples/chat_with_document/start.go
...
```

#### List local datasets

```shell
❯ arctl dataset list -h
List dataset

Usage:
  arctl dataset list [usage] [flags]

Flags:
  -h, --help   help for list

Global Flags:
      --home string   home directory to use (default "/Users/bjwswang/.arcadia")
```

> Note: arctl will list local datasets cached under `/Users/bjwswang/.arcadia/dataset` by default

For example:

```shell
❯ arctl dataset list
| DATASET | FILES |EMBEDDING MODEL | VECTOR STORE | DOCUMENT LANGUAGE | CHUNK SIZE | CHUNK OVERLAP |
| arcadia | 4 | zhipuai | http://localhost:8000 | text | 300 | 30 |
```

#### Show a local dataset info

```shell
❯ arctl dataset show -h
Load more documents to dataset

Usage:
  arctl dataset show [usage] [flags]

Flags:
  -h, --help          help for show
      --name string   dataset(namespace/collection) of the document to load into

Global Flags:
      --home string   home directory to use (default "/Users/bjwswang/.arcadia")
```

Required Arguments:
- `--name`: The name of the dataset


For example:

```shell
❯ arctl dataset show --name arcadia
I1012 17:57:17.026985    7609 dataset.go:206]
{
  "name": "arcadia",
  "create_time": "2023-10-12 17:53:44.541654 +0800 CST m=+0.003758739",
  "llm_type": "zhipuai",
  "llm_api_key": "4fcceceb1666cd11808c218d6d619950.TCXUvaQCWFyIkxB3",
  "vector_store": "http://localhost:8000",
  "document_language": "text",
  "text_splitter": "character",
  "chunk_size": 300,
  "chunk_overlap": 30,
  "files": {
    "2905cb6e865ce2192369038eaa7f9f8c3d3ba6f2a6ae01b5c23afc21c01e4bd8": {
      "path": "./README.md",
      "size": 6176,
      "chunks": 29,
      "chunks_loaded": 29
    },
    "2b6d423b926936244c6ddb56e7b9a60e3c6d2b866feabc75bb3deda27cd2f94e": {
      "path": "examples/chat_with_document/main.go",
      "size": 982,
      "chunks": 5,
      "chunks_loaded": 5
    },
    "8e7094a9c6a062c6a4e19cb43cc3abef23b0569b51a5f64655d2761c04e5835a": {
      "path": "examples/chat_with_document/handler.go",
      "size": 6441,
      "chunks": 30,
      "chunks_loaded": 30
    },
    "bc0dcb3a672ef35b8e2c915f67ca0c7accd4883148a0506f1749971a22cafa96": {
      "path": "examples/chat_with_document/README.md",
      "size": 6267,
      "chunks": 30,
      "chunks_loaded": 30
    }
  }
}
```

#### Delete a local dataset

```shell
❯ arctl dataset delete -h
Delete dataset

Usage:
  arctl dataset delete [usage] [flags]

Flags:
  -h, --help                 help for delete
      --name string          dataset(namespace/collection) of the document to load into (default "arcadia")
      --reset-vector-store   forcely reset dataset from remote vector store

Global Flags:
      --home string   home directory to use (default "/Users/bjwswang/.arcadia")
```

Required Arguments:
- `--name`: The name of the dataset


For example:

```shell
❯ arctl dataset delete --name arcadia
I1012 18:06:04.894410    8786 dataset.go:272] Delete dataset: arcadia
I1012 18:06:04.894985    8786 dataset.go:303] Successfully delete dataset: arcadia
```

**NOTE: If you want to remove the vectors as well,you should set `--reset-vector-store` flag**

### Chat with LLM
```shell
❯ arctl chat -h
Do LLM chat with similarity search(optional)

Usage:
  arctl chat [usage] [flags]

Flags:
      --dataset string            dataset(namespace/collection) to query from
  -h, --help                      help for chat
      --llm-apikey string         apiKey to access llm service.Must required when embedding similarity search is enabled
      --llm-type string           llm type to use for embedding & chat(Only zhipuai,openai supported now) (default "zhipuai")
      --method string             Invoke method used when access LLM service(invoke/sse-invoke) (default "sse-invoke")
      --model string              which model to use: chatglm_lite/chatglm_std/chatglm_pro (default "chatglm_lite")
      --num-docs int              number of documents to be returned with SimilarSearch (default 5)
      --question string           question text to be asked
      --score-threshold float32   score threshold for similarity search(Higher is better)
      --temperature float32       temperature for chat (default 0.95)
      --top-p float32             top-p for chat (default 0.7)

Global Flags:
      --home string   home directory to use (default "/Users/bjwswang/.arcadia")
```

Now `arctl chat` has two modes which is controlled by flag `--enable-embedding-search`:
- normal chat without embedding search(default)
- enable similarity search with embedding

#### Chat without embedding

> This will chat with LLM `zhipuai` with its apikey by using model `chatglm_pro` without embedding

```shell
arctl chat --llm-apikey 26b2bc55fae40752055cadfc4792f9de.wagA4NIwg5aZJWhm --model chatglm_pro --question "介绍一下Arcadia"
```

Required Arguments:
- `--llm-apikey`
- `--question`

**Output:**
```shell
Prompts: [{user 介绍一下Arcadia}] 
 Arcadia（阿卡迪亚）是一个知名的游戏开发团队，成立于 2007 年，总部位于美国加州洛杉矶。Arcadia 主要致力于创作高质量的社交游戏和虚拟世界，旨在为玩家提供一个沉浸式的游戏体验。他们的作品在全球范围内拥有大量的玩家，其中包括许多虚拟世界和模拟经营类的游戏。

Arcadia 的创始人兼 CEO 是 Raph Koster，他是一位在游戏行业拥有丰富经验的游戏设计师。在成立 Arcadia 之前，他曾担任过多家知名游戏公司的要职，如索尼在线娱乐、艺电等。

Arcadia 开发的游戏中最知名的作品之一是《Second Life》（第二人生），这是一款非常受欢迎的虚拟世界游戏，玩家可以在游戏中创建自己的虚拟角色、探索环境、与其他玩家互动等。这款游戏自推出以来获得了许多奖项，被誉为虚拟世界游戏的典范。

除了《Second Life》之外，Arcadia 还开发了许多其他游戏，如《Champions Online》、《Star Wars: Galaxies》等。此外，他们还为其他游戏公司提供游戏开发咨询和技术支持服务。

总的来说，Arcadia 是一家在游戏行业具有较高声誉和影响力的公司，他们不断推出新的游戏作品，为玩家带来更多精彩的游戏体验。
```

#### Chat with embedding

> This will chat with LLM `zhipuai` with its apikey by using model `chatglm_pro` with embedding enabled

```shell
arctl chat --llm-apikey 26b2bc55fae40752055cadfc4792f9de.wagA4NIwg5aZJWhm --model chatglm_pro  --num-docs 10 --question "介绍一下Arcadia" --dataset arcadia
```

Required Arguments:
- `--dataset`
- `--llm-apikey`
- `--question`

**Output:**
```shell
Prompts: [{user 
      我将要询问一些问题，希望你仅使用我提供的上下文信息回答。
      请不要在回答中添加其他信息。
      若我提供的上下文不足以回答问题,
      请回复"我不确定"，再做出适当的猜测。
      请将回答内容分割为适于阅读的段落。
      } {assistant 
        好的，我将尝试仅使用你提供的上下文信息回答，并在信息不足时提供一些合理推测。
      } {user 我的问题是: 介绍一下Arcadia. 以下是我提供的上下文:--- sidebar_position: 1 ---  # 介绍## Contribute to Arcadia  If you want to contribute to Arcadia, refer to [contribute guide](CONTRIBUTING.md).  ## Support  If you need support, start with the troubleshooting guide, or create GitHub [issues](https://github.com/kubeagi/arcadia/issues/new)```of developers contributing to its continued development and improvement. It is available on a variety of platforms, including Linux, Windows, and 移动设备，and can be deployed on-premises, in the cloud, or in a hybrid# Arcadia  Our vision is to make it easier for cloud-native applications to integrate with AI, thereby making the cloud more intelligent and impactful.  ## Quick start  1. Install arcadia operatormaintained by the Cloud Native Computing Foundation (CNCF).\\n\\nKubernetes provides a platform-as-a-service (PaaS) model, which allows developers to deploy, run, and scale containerized applications with minimal configuration and effort. It does this by abstracting the underlying infrastructure andunderlying infrastructure and providing a common set of APIs and tools that can be used to deploy, manage, and scale applications consistently across different environments.\\n\\nKubernetes is widely adopted by organizations of all sizes and has a large, active community of developers contributingin the cloud, or in a hybrid environment.\"","role":"assistant"}],"request_id":"7865480399259975113","task_id":"7865480399259975113","task_status":"SUCCESS","usage":{"total_tokens":203}},"msg":"操作成功","success":true}[内核](https://github.com/kubebb/core)基于[kubernetes## Packages  To enhace the AI capability in Golang,we developed some packages.  ### LLMs  - ✅ [ZhiPuAI(智谱AI)](https://github.com/kubeagi/arcadia/tree/main/pkg/llms/zhipuai)   - [example](https://github.com/kubeagi/arcadia/blob/main/examples/zhipuai/main.go)  ### Embeddings}] 
 Arcadia 是一个项目，旨在使云原生应用程序更容易与 AI 集成，从而使云更加智能和有影响力。该项目由 Cloud Native Computing Foundation（CNCF）维护，并可在多种平台上使用，包括 Linux、Windows 和移动设备。它可以通过本地部署、云部署或混合部署。

Kubernetes 是一个提供平台即服务（PaaS）模型的系统，它使开发人员能够以最小的配置和努力来部署、运行和扩展容器化应用程序。它通过抽象底层基础设施，并提供一组通用的 API 和工具来实现这一目标，这些 API 和工具可以用于在不同的环境中一致地部署、管理和扩展应用程序。

Arcadia 项目包含了一些用于增强 Golang 中 AI 功能的软件包。其中，智谱 AI（ZhiPuAI）是一个 LLM（大型语言模型）软件包，提供了示例代码供开发者参考。
```

## Limitations

1. Vector Store

- ✅ chromadb

2. Document Types

- ✅ text
- ✅ html
- ✅ csv

3. Embedding Service

- ✅ zhipuai

4. LLM Service

- ✅ zhipuai
