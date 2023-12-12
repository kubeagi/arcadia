## Enable streamlit for your namespace
We use streamlit as the tool for doing data analysis and LLM app playground, and we can create a separate streamlit environment for each namespace for better isolation.

1. To enable streamlit for your namespace, add the annotation ```kubeagi.k8s.com.cn/streamlit.installed: "true"``` to the namespace.
```yaml
apiVersion: v1
kind: Namespace
metadata:
  annotations:
    # enable streamlit for this namespace
    # change the value to 'false' will delete all resources related to this streamlit environment
    kubeagi.k8s.com.cn/streamlit.installed: "true"
  creationTimestamp: "2023-12-08T02:36:26Z"
```
Then there will be deployment, service and ingress created for streamlit under this namespace.

## Access streamlit
1. Check the ingress configuration of the streamlit using ```kubectl get ing streamlit-app -n <your-namespace> -o yaml```

2. It'll have a context path similar as ```/chats/<your-namespace>```, /chats is the global context configured at kubeagi-system's arcadia-config configMap.

So you can visit streamlit using address like ```https://portal.172.40.20.125.nip.io/chats/<your-namespace>```


## Customize your streamlit app
Based on the streamlit design, we can customize streamlit application easily using python script.

1. Check your streamlit pod
```shell
# get the pod
kubectl get pods -n <your-namespace>
# copy python script to your streamlit pages directory
# you can find some samples under examples/app-ui-using-streamlit directory
kubectl cp your-demo-streamlit-app.py <pod-name>:/app/pages/
```

2. Then access the application using ```https://portal.172.40.20.125.nip.io/chats/<your-namespace>/your-demo-streamlit-app```
