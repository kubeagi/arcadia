# Schedule a Kind cluster without GPU enabled

It is quite easy to schedule a kind cluster if you don't want to enable GPU support. See following steps:

1. Install Kind CLI

Follow [document](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) to install **kind** to your local machine

2. Create kind cluster `kubeagi`

At the root directory of [`kubeagi/arcadia`](https://github.com/kubeagi/arcadia).

Run command:

```shell
make kind
```

Now you have a kind cluster runnning! 
