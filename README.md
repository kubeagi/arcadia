# Arcadia

Our vision is to make it easier for cloud-native applications to integrate with AI, thereby making the cloud more intelligent and impactful.

## Quick start

### 1. Install arcadia

```shell
helm repo add arcadia https://kubeagi.github.io/arcadia
helm repo update
helm install arcadia arcadia/arcadia
```

### 2. Add a LLM

> Take [ZhiPuAI](https://www.zhipuai.cn/) as an example.

1. Prepare auth info

Update apiKey(`Base64 encoded`) in [zhipuai's secret](https://github.com/kubeagi/arcadia/blob/main/config/samples/arcadia_v1alpha1_llm.yaml#L7).

> On how to get apiKey, please refer to [ZhiPuAI](https://open.bigmodel.cn/dev/api#auth)

2. Create a LLM along with the auth secret

```shell
kubectl apply -f config/samples/arcadia_v1alpha1_llm.yaml
```

### 3. Create a prompt

```shell
kubectl apply -f config/samples/arcadia_v1alpha1_prompt.yaml
```

After prompt got created, you can see the prompt in the following command:

```shell
kubectl get prompt prompt-zhipuai-sample -oyaml
```

If no error found,you can use this command to get the prompt response data.

```shell
kubectl get prompt prompt-zhipuai-sample --output="jsonpath={.status.data}" | base64 --decode
```

Output:

```shell
{"code":200,"data":{"choices":[{"content":"\" Kubernetes (also known as K8s) is an open-source container orchestration system for automating the deployment, scaling, and management of containerized applications. It was originally designed by Google, and is now maintained by the Cloud Native Computing Foundation (CNCF).\\n\\nKubernetes provides a platform-as-a-service (PaaS) model, which allows developers to deploy, run, and scale containerized applications with minimal configuration and effort. It does this by abstracting the underlying infrastructure and providing a common set of APIs and tools that can be used to deploy, manage, and scale applications consistently across different environments.\\n\\nKubernetes is widely adopted by organizations of all sizes and has a large, active community of developers contributing to its continued development and improvement. It is available on a variety of platforms, including Linux, Windows, and 移动设备，and can be deployed on-premises, in the cloud, or in a hybrid environment.\"","role":"assistant"}],"request_id":"7865480399259975113","task_id":"7865480399259975113","task_status":"SUCCESS","usage":{"total_tokens":203}},"msg":"操作成功","success":true}
```

## Clients

We developed some clients in Golang to interfact with AI in Golang.

### ZhiPuAI(ChatGLM)

- [Client](https://github.com/kubeagi/arcadia/tree/main/pkg/llms/zhipuai)
- [Examples on How to use this Library](https://github.com/kubeagi/arcadia/blob/main/examples/zhipuai/main.go)

For ChatGLM,see [here](https://github.com/THUDM/ChatGLM2-6B)
For `智谱AI`,see [here](https://open.bigmodel.cn/)

## Contribute to Arcadia

If you want to contribute to Arcadia, refer to [contribute guide](CONTRIBUTING.md).

## Support

If you need support, start with the troubleshooting guide, or create GitHub [issues](https://github.com/kubeagi/arcadia/issues/new)
