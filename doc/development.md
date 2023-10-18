# Development

## Init a new operator project

```bash
bash hack/install-operator-sdk

operator-sdk init --domain kubeagi.k8s.com.cn --component-config true --owner kubeagi --project-name arcadia --repo github.com/kubeagi/arcadia
```

## Create a CRD

```bash
operator-sdk create api --resource --controller --namespaced=true --group arcadia --version v1alpha1 --kind Laboratory
```

### Regenerate after changes on CRD

```bash
make generate && make manifests
```
