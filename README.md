# Arcadia: A diverse, simple, and secure all-in-one LLMOps platform

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

**Arcadia** is a all-in-one enterprise-grade LLMOps platform that provides a unified interface for developers and operators to build, debug,deploy and manage AI agents with a orchestration engine(**RAG(Retrieval Augmented Generation)** and **LLM finetuning** has been supported).

## Features

* Build,debug,deploy AI agents on ops-console(GUI for LLMOps)
* Chat with AGI agent on agent-portal(GUI for gpt chat)
* Enterprise-grade infratructure with [KubeBB](https://github.com/kubebb): Multi-tenant isolation (data, model services), built-in OIDC, RBAC, and auditing, supporting different companies and departments to develop through a unified platform
* Support most of the popular LLMs(large language models),embedding models,reranking models,etc..
* Inference acceleration with [vllm](https://github.com/vllm-project/vllm),distributed inference with [ray](https://github.com/ray-project/ray),quantization, and more
* Support fine-tuining with [llama-factory](https://github.com/hiyouga/LLaMA-Factory)
* Built on langchaingo(golang), has better performance and maintainability

## Architecture

Our design and development in Arcadia design follows operator pattern which extends Kubernetes APIs.

![Arch](./docs/images/kubeagi.drawio.png)

For details, check [Architecture Overview](http://kubeagi.k8s.com.cn/docs/Concepts/architecture-overview)

## Quick Start

### Documentation

Visit our [online documents](http://kubeagi.k8s.com.cn/docs/intro)

Read [user guide](http://kubeagi.k8s.com.cn/docs/UserGuide/intro)

## Supported Models

### List of Models can be deployed by kubeagi

### LLMs

* [chatglm2-6b](https://huggingface.co/THUDM/chatglm2-6b)
* [chatglm3-6b](https://huggingface.co/THUDM/chatglm3-6b>)
* [qwen(7B,14B,72B)](https://huggingface.co/Qwen)
* [qwen-1.5(0.5B,1.8B,4B,14B,32B](https://huggingface.co/collections/Qwen/qwen15-65c0a2f577b1ecb76d786524)
* [baichuan2](https://huggingface.co/baichuan-inc)
* [llama2](https://huggingface.co/meta-llama)
* [mistral](https://huggingface.co/mistralai)

### Embeddings

* [bge-large-zh](https://huggingface.co/BAAI/bge-large-zh-v1.5)
* [m3e](https://huggingface.co/moka-ai/m3e-base)

### Reranking

* [bge-reranker-large](https://huggingface.co/BAAI/bge-reranker-large) ***reranking***
* [bce-reranking](<https://github.com/netease-youdao/BCEmbedding>) ***reranking***

### List of Online(third party) LLM Services can be integrated by kubeagi

* [OpenAI](https://openai.com/)
* [Google Gemini](https://gemini.google.com/)
* [智谱AI](https://github.com/kubeagi/arcadia/tree/main/pkg/llms/zhipuai)
  * [example](https://github.com/kubeagi/arcadia/blob/main/examples/zhipuai/main.go)
  * [embedding](https://github.com/kubeagi/arcadia/tree/main/pkg/embeddings/zhipuai)
* [DashScope(灵积模型服务)](https://github.com/kubeagi/arcadia/tree/main/pkg/llms/dashscope)
  * [example](https://github.com/kubeagi/arcadia/blob/main/examples/dashscope/main.go)
  * [text-embedding-v1(通用文本向量 同步接口)](https://help.aliyun.com/zh/dashscope/developer-reference/text-embedding-api-details)

## Supported VectorStores

> Fully compatible with [langchain vectorstores](https://github.com/tmc/langchaingo/tree/main/vectorstores)

* ✅ [PG Vector](https://github.com/tmc/langchaingo/tree/main/vectorstores/pgvector), KubeAGI adds the PG vector support to [langchaingo](https://github.com/tmc/langchaingo) project.
* ✅ [ChromaDB](https://docs.trychroma.com/)

## Pure Go Toolchains

Thanks to [langchaingo](https://github.com/tmc/langchaingo),we can have comprehensive AI capability in Golang!But in order to meet our own unique needs, we have further developed a number of other toolchains:

* [Optimized DocumentLoaders](https://github.com/kubeagi/arcadia/tree/main/pkg/documentloaders): optimized csv,etc...
* [Extended LLMs](https://github.com/kubeagi/arcadia/tree/main/pkg/llms): zhipuai,dashscope,etc...
* [Tools](https://github.com/kubeagi/arcadia/tree/main/pkg/tools): bingsearch,weather,etc...
* [AppRuntime](https://github.com/kubeagi/arcadia/tree/main/pkg/appruntime): powerful node(LLM,Chain,KonwledgeBase,vectorstore,Agent,etc...) orchestration runtime for arcadia

We have provided some examples on how to use them. See more details at [here](https://github.com/kubeagi/arcadia/tree/main/examples)

## Contribute to Arcadia

If you want to contribute to Arcadia, refer to [contribute guide](http://kubeagi.k8s.com.cn/docs/Contribute/prepare-and-start).

## Support

If you need support, start with the troubleshooting guide, or create GitHub [issues](https://github.com/kubeagi/arcadia/issues/new)
