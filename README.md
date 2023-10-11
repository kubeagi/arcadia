# Arcadia

Our vision is to make it easier for cloud-native applications to integrate with AI, thereby making the cloud more intelligent and impactful. We provides two ways for users to develop AI applications :point_down:

- :fire:[Arcadia Operator](https://github.com/kubeagi/arcadia/tree/main/charts/arcadia) provides comprehensive features to develop,build,publish AI applications 
  - `Dataset`: automatically process `Files` with embedding models,then store vectors into vector stores
  - `LLMs`
    - :cloud: LLM service provider 
    - Local distributed LLMs(OpenAI API compatible)
  - `Prompts`: call llm and keep results 
  - `Application`: provide templates and tools to `build,publish`
  - ...
- :fire:[Pure Golang toolchains](#pure-go-toolchains)  to develop with your own needs

## Quick start

1. Install arcadia operator

We recommend that install arcadia under namespace `arcadia`

```shell
helm repo add arcadia https://kubeagi.github.io/arcadia
helm repo update
helm install --namespace arcadia --create-namespace arcadia arcadia/arcadia 
```

2. Add a LLM along with the auth secret

> Update apiKey(`Base64 encoded`) in [secret](https://github.com/kubeagi/arcadia/blob/main/config/samples/arcadia_v1alpha1_llm.yaml#L7).

```shell
kubectl apply -f config/samples/arcadia_v1alpha1_llm.yaml
```

### 3. Create a prompt

```shell
kubectl apply -f config/samples/arcadia_v1alpha1_prompt.yaml
```

After the prompt got created, you can see the prompt in the following command:

```shell
kubectl get prompt prompt-zhipuai-sample -oyaml
```

If no error is found, you can use this command to get the prompt response data.

```shell
kubectl get prompt prompt-zhipuai-sample --output="jsonpath={.status.data}" | base64 --decode
```

Output:

```shell
{"code":200,"data":{"choices":[{"content":"\" Kubernetes (also known as K8s) is an open-source container orchestration system for automating the deployment, scaling, and management of containerized applications. It was originally designed by Google, and is now maintained by the Cloud Native Computing Foundation (CNCF).\\n\\nKubernetes provides a platform-as-a-service (PaaS) model, which allows developers to deploy, run, and scale containerized applications with minimal configuration and effort. It does this by abstracting the underlying infrastructure and providing a common set of APIs and tools that can be used to deploy, manage, and scale applications consistently across different environments.\\n\\nKubernetes is widely adopted by organizations of all sizes and has a large, active community of developers contributing to its continued development and improvement. It is available on a variety of platforms, including Linux, Windows, and 移动设备，and can be deployed on-premises, in the cloud, or in a hybrid environment.\"","role":"assistant"}],"request_id":"7865480399259975113","task_id":"7865480399259975113","task_status":"SUCCESS","usage":{"total_tokens":203}},"msg":"操作成功","success":true}
```

## CLI

We provide a Command Line Tool `arctl` to interact with `arcadia` and `LLMs`. See [here](./arctl/README.md) for more details.

## Pure Go Toolchains

To enhace the AI capability in Golang, we developed some packages.Here are the examples of how to use them.

- [chat_with_document](https://github.com/kubeagi/arcadia/tree/main/examples/chat_with_document): a chat server which allows you to chat with your document
- [embedding](https://github.com/kubeagi/arcadia/tree/main/examples/embedding) shows how to embedes your document to vector store with embedding service
- [rbac](https://github.com/kubeagi/arcadia/blob/main/examples/rbac/main.go) shows how to inquiry the security risks in your RBAC with AI.
- [zhipuai](https://github.com/kubeagi/arcadia/blob/main/examples/zhipuai/main.go) shows how to use this [zhipuai client](https://github.com/kubeagi/arcadia/tree/main/pkg/llms/zhipuai)

### LLMs

- ✅ [ZhiPuAI(智谱 AI)](https://github.com/kubeagi/arcadia/tree/main/pkg/llms/zhipuai)
  - [example](https://github.com/kubeagi/arcadia/blob/main/examples/zhipuai/main.go)

### Embeddings

> Fully compatible with [langchain embeddings](https://github.com/tmc/langchaingo/tree/main/embeddings)

- ✅[ZhiPuAI(智谱 AI) Embedding](https://github.com/kubeagi/arcadia/tree/main/pkg/embeddings/zhipuai)

### VectorStores

> Fully compatible with [langchain vectorstores](https://github.com/tmc/langchaingo/tree/main/vectorstores)

- ✅[ChromaDB](https://docs.trychroma.com/)

## Contribute to Arcadia

If you want to contribute to Arcadia, refer to [contribute guide](CONTRIBUTING.md).

## Support

If you need support, start with the troubleshooting guide, or create GitHub [issues](https://github.com/kubeagi/arcadia/issues/new)
