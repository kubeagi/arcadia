# Chat with Worker

An example to chat with [worker baichuan2-7b](https://github.com/kubeagi/arcadia/blob/main/config/samples/arcadia_v1alpha1_worker_baichuan2-7b.yaml).

We utilize [fastchat](https://github.com/lm-sys/FastChat) to provide OpenAI-Compatible RESTFul APIs.

Before using this example,please make sure you have deployed this [baichuan2-7b worker](https://github.com/kubeagi/arcadia/blob/main/config/samples/arcadia_v1alpha1_worker_baichuan2-7b.yaml)

## Usage

1. Update the baseurl in [main.go](./main.go) based on how you deploy arcadia

At this example, we deploy arcadia 

- namespace: arcadia
- with ingress: arcadia-fastchat.172.22.96.167.nip.io

Then we get a final base url `http://arcadia-fastchat.172.22.96.167.nip.io/v1`

2. Run this chat & embedding example

```golang
go run main.go
```