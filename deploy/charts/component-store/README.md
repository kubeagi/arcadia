# Component Store

This Helm chart installs the `component-store` into your Kubernetes.

### Install

```shell
    helm install component-store kubebb/component-store --set ingressDomain=<ip>.nip.io
```

### Upgrade

```shell
    helm upgrade component-store kubebb/component-store --set ingressDomain=<ip>.nip.io
```

### Uninstall

```shell
    helm uninstall component-store
```

### Configuration


| Parameter           | Description                  | Default                           |
|---------------------|------------------------------|-----------------------------------|
| `ingressDomain`     | Ingress Domain               |                                   |
| `ingressClassName`  | Ingress Class                | `portal-ingress`                  |
| `image`             | Image of the component-store | `kubebb/component-store:latest`   |
| `imagePullPolicy`   | Image Pull Policy            | `IfNotPresent`                    |
| `resources`         | Resources of the container   |                                   |
