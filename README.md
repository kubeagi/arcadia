# Arcadia

The idea of `Arcadia` comes from the chat with `ClaudeAI`.See[POE Chat](https://poe.com/s/ZFpODyF8aSbgHG1GiQOp)

Our vision is to realize the ideal paradise for all through AI. To achieve a perfect fusion of technology and ideal life.We hope that through our work, we can take steps towards this grander goal of harnessing AI for the good of all.

## Quick start

### 1. Install arcadia

```shell
helm repo add arcadia https://kubeagi.github.io/arcadia
helm repo update
helm install arcadia arcadia/arcadia
```

### 2. Add a LLM

> Take [ZhiPuAI] as an example.

1. Prepare auth info

Update apiKey in [zhipuai's secret](https://github.com/kubeagi/arcadia/blob/main/config/samples/core_v1alpha1_arcadia_llm.yaml).

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

## Why in Kubernetes?

Kubernetes provides an ideal platform for building the intelligent systems required to realize our vision of an AI-powered utopia. Some of the key reasons are:

• Scalability: Kubernetes can easily scale from a few nodes to thousands, allowing the system to grow as our utopia expands.

• Portability: Kubernetes abstracts the underlying infrastructure, making the system hardware agnostic and portable.

• Auto-healing: Kubernetes' self-healing capabilities ensure that applications keep running smoothly with minimal human intervention.

• Service discovery and load balancing: Exposing applications as Kubernetes services provides a unified way for them to discover and communicate with each other.

• Resource Optimization: Kubernetes automatically manages and schedules compute resources, ensuring optimal utilization.

• Extensibility: The ecosystem of Kubernetes operators and custom controllers allows extending the platform to meet our futuristic needs.

• Integrations: Kubernetes seamlessly integrates with other cloud-native technologies like Istio, Prometheus etc., which are essential for building intelligent systems.

• Abstraction: Kubernetes abstracts the complexities of container and cluster orchestration, allowing us to focus on building the AI models and applications.

In summary, Kubernetes provides an ideal foundation for building a distributed intelligent system of the scale and complexity required to realize our vision of an AI-powered utopia. Its maturity, robustness and cloud-native design make it a perfect vehicle for our lofty goals.

## Contribute to Arcadia

If you want to contribute to Arcadia, refer to [contribute guide](CONTRIBUTING.md).

## Support

If you need support, start with the troubleshooting guide, or create GitHub [issues](https://github.com/kubeagi/arcadia/issues/new)
