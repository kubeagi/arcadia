# Arcadia: A diverse, simple, and secure one-stop LLMOps platform

<div align="left">
  <p>
    <a href="https://opensource.org/licenses/apache-2-0">
      <img alt="License: Apache-2.0" src="https://img.shields.io/github/license/kubeagi/arcadia" />
    </a>
    <a href="https://goreportcard.com/report/github.com/kubeagi/arcadia">
      <img alt="Go Report Card" src="https://goreportcard.com/badge/kubeagi/arcadia?style=flat-square" />
    </a>
    <a href="https://github.com/psf/black">
      <img alt="CodeStyle" src="https://img.shields.io/badge/code%20style-black-000000.svg" />
    </a>
  </p>
</div>

## What is Arcadia?

**Arcadia** comes from [Greek mythology](https://www.greekmythology.com/Myths/Places/Arcadia/arcadia.html)(a tranquil and idyllic region, representing harmony, serenity, and natural beauty). We aim to help everyone find a more perfect integration between humans and AI.

To achieve this goal, we provide this one-stop LLMOps solution. Furthermore, we can easily host **Arcadia** at any Kubernetes cluster as production ready by integrating [kubebb](https://github.com/kubebb)(Kubernetes building blocks).

## Features
* Multi-tenant isolation (data, model services), built-in OIDC, RBAC, and auditing, supporting different companies and departments to develop through a unified platform
* Kubernetes native AGI agent orchestration
* Built on langchaingo(golang), has better performance and maintainability
* Support distributed inference using Ray
* Support quality and performance evaluation of AGI agent under different configurations
* A development and operational platform for AI agents, along with an AI agent portal for end-users
* Developed based on micro frontends and low-code approach, allowing for quick scalability and integration

## Architecture

Our design and development in Arcadia design follows operator pattern which extends Kubernetes APIs.

![Arch](./docs/images/kubeagi.drawio.png)

For details, check [Architecture Overview](http://kubeagi.k8s.com.cn/docs/Concepts/architecture-overview)

## Quick Start

### Documentation

Visit our [online documents](http://kubeagi.k8s.com.cn/docs/intro)

Read [user guide](http://kubeagi.k8s.com.cn/docs/UserGuide/intro)

### LLMs

List of supported(tested) LLMs
* baichuan2-7b
* chatglm2-6b
* qwen-7b-chat / qwen-14b-chat / qwen-72b-chat
* llama2-7b
* mistral-7b
* bge-large-zh  ***embedding***
* m3e ***embedding***
* [ZhiPuAI(智谱 AI)](https://github.com/kubeagi/arcadia/tree/main/pkg/llms/zhipuai)
  - [example](https://github.com/kubeagi/arcadia/blob/main/examples/zhipuai/main.go)
  - [embedding](https://github.com/kubeagi/arcadia/tree/main/pkg/embeddings/zhipuai)
* [DashScope(灵积模型服务)](https://github.com/kubeagi/arcadia/tree/main/pkg/llms/dashscope)
  - [example](https://github.com/kubeagi/arcadia/blob/main/examples/dashscope/main.go)
  - [text-embedding-v1(通用文本向量 同步接口)](https://help.aliyun.com/zh/dashscope/developer-reference/text-embedding-api-details)

### Embeddings

> Fully compatible with [langchain embeddings](https://github.com/tmc/langchaingo/tree/main/embeddings)

### VectorStores

> Fully compatible with [langchain vectorstores](https://github.com/tmc/langchaingo/tree/main/vectorstores)

- ✅ [PG Vector](https://github.com/tmc/langchaingo/tree/main/vectorstores/pgvector), KubeAGI adds the PG vector support to [langchaingo](https://github.com/tmc/langchaingo) project.
- ✅ [ChromaDB](https://docs.trychroma.com/)

## Pure Go Toolchains

Thanks to [langchaingo](https://github.com/tmc/langchaingo),we can have comprehensive AI capability in Golang!But in order to meet our own unique needs, we have further developed a number of other toolchains:

- [Optimized DocumentLoaders](https://github.com/kubeagi/arcadia/tree/main/pkg/documentloaders): optimized csv,etc...
- [Extended LLMs](https://github.com/kubeagi/arcadia/tree/main/pkg/llms): zhipuai,dashscope,etc...
- [Tools](https://github.com/kubeagi/arcadia/tree/main/pkg/tools): bingsearch,weather,etc...
- [AppRuntime](https://github.com/kubeagi/arcadia/tree/main/pkg/appruntime): powerful node(LLM,Chain,KonwledgeBase,vectorstore,Agent,etc...) orchestration runtime for arcadia

We have provided some examples on how to use them. See more details at [here](https://github.com/kubeagi/arcadia/tree/main/examples)

## CLI

We provide a Command Line Tool `arctl` to interact with `arcadia`. See [here](http://kubeagi.k8s.com.cn/docs/Tools/arctl-tool) for more details.

- ✅ datasource management
- ✅ RAG evaluation

## Contribute to Arcadia

If you want to contribute to Arcadia, refer to [contribute guide](http://kubeagi.k8s.com.cn/docs/Contribute/prepare-and-start).

## Support

If you need support, start with the troubleshooting guide, or create GitHub [issues](https://github.com/kubeagi/arcadia/issues/new)
