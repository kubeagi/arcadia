# llama-factory

Originally from [llama-factory](https://github.com/huangqg/helm-charts/tree/main/charts/llama-factory)

## Usage

Before install llama-factory:

- Must replace `<replaced-ingress-nginx-ip>`with the real ingress ip address(`172.18.0.2` for example) if ingress is enabled

### Install via helm

```shell
helm install -nkubeagi-system lmf .
```

If `<replaced-ingress-nginx-ip>` is `172.18.0.2`, then the dashboard of llama factory is `https://lmf.172.18.0.2.nip.io`.