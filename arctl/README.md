# arctl

arctl(arcadia command line tool) 

## Quick Install

```shell
go install github.com/kubeagi/arcadia/arctl@latest
```

If build succeeded, `arctl` will be built into `bin/arctl` under `arcadia`

```shell
❯ arctl -h
Command line tools for Arcadia

Usage:
  arctl [usage] [flags]
  arctl [command]

Available Commands:
  load        Load documents into VectorStore
  chat        Do LLM chat with similarity search(optional)
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command

Flags:
  -h, --help   help for arctl

Use "arctl [command] --help" for more information about a command.
```

## Usage
### Load documents into vector store

```shell
❯ arctl load -h
Load documents into VectorStore

Usage:
  arctl load [usage] [flags]

Flags:
      --chunk-overlap int             chunk overlap for embedding (default 30)
      --chunk-size int                chunk size for embedding (default 300)
      --document string               path of the document to load
      --document-language string      language of the document(Only text,html,csv supported now) (default "text")
      --embedding-llm-apikey string   apiKey to access embedding service
      --embedding-llm-type string     llm type to use(Only zhipuai,openai supported now) (default "zhipuai")
  -h, --help                          help for load
      --namespace string              namespace/collection of the document to load into (default "arcadia")
      --vector-store string           vector stores to use(Only chroma supported now) (default "http://localhost:8000")
```

Required Arguments:
- `--embedding-llm-apikey`
- `--document`

For example:
```shell
arctl load  --embedding-llm-apikey 26b2bc55fae40752055cadfc4792f9de.wagA4NIwg5aZJWhm --document ./README.md
```

### Chat with LLM
```shell
❯ arctl chat -h
Do LLM chat with similarity search(optional)

Usage:
  arctl chat [usage] [flags]

Flags:
      --chat-llm-apikey string        apiKey to access embedding service
      --chat-llm-type string          llm type to use(Only zhipuai,openai supported now) (default "zhipuai")
      --embedding-llm-apikey string   apiKey to access embedding service.Must required when embedding similarity search is enabled
      --embedding-llm-type string     llm type to use(Only zhipuai,openai supported now) (default "zhipuai")
      --enable-embedding-search       enable embedding similarity search
  -h, --help                          help for chat
      --method string                 Invoke method used when access LLM service(invoke/sse-invoke) (default "sse-invoke")
      --model string                  which model to use: chatglm_lite/chatglm_std/chatglm_pro (default "chatglm_lite")
      --namespace string              namespace/collection to query from (default "arcadia")
      --num-docs int                  number of documents to be returned with SimilarSearch (default 3)
      --question string               question text to be asked
      --score-threshold float         score threshold for similarity search(Higher is better)
      --temperature float32           temperature for chat (default 0.95)
      --top-p float32                 top-p for chat (default 0.7)
      --vector-store string           vector stores to use(Only chroma supported now) (default "http://localhost:8000")
```

Now `arctl chat` has two modes which is controlled by flag `--enable-embedding-search`:
- normal chat without embedding search(default)
- enable similarity search with embedding

#### Normal chat(Without embedding)

```shell
arctl chat --chat-llm-apikey 26b2bc55fae40752055cadfc4792f9de.wagA4NIwg5aZJWhm --model chatglm_pro --question "介绍一下Arcadia"
```

Required Arguments:
- `--chat-llm-apikey`
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

#### Enable Similarity Search

```shell
arctl chat --enable-embedding-search --chat-llm-apikey 26b2bc55fae40752055cadfc4792f9de.wagA4NIwg5aZJWhm --embedding-llm-apikey 26b2bc55fae40752055cadfc4792f9de.wagA4NIwg5aZJWhm --model chatglm_pro  --num-docs 10 --question "介绍一下Arcadia"
```

Required Arguments:
- `--enable-embedding-search`
- `--embedding-llm-apikey`
- `--chat-llm-apikey`
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
- ✅ openai

4. LLM Service

- ✅ zhipuai
