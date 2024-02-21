# Examples in arcadia

- [chat_with_worker](https://github.com/kubeagi/arcadia/tree/main/examples/chat_with_document): a chat server which allows you to chat with the help of the model services deployed in kubeagi
- [chat_with_document](https://github.com/kubeagi/arcadia/tree/main/examples/chat_with_document): a chat server which allows you to chat with your document
- [zhipuai](https://github.com/kubeagi/arcadia/blob/main/examples/zhipuai/main.go) shows how to use this [zhipuai client](https://github.com/kubeagi/arcadia/tree/main/pkg/llms/zhipuai)
- [dashscope](https://github.com/kubeagi/arcadia/blob/main/examples/dashscope/main.go) shows how to use this [dashscope client](https://github.com/kubeagi/arcadia/tree/main/pkg/llms/dashscope) to chat with qwen-7b-chat / qwen-14b-chat / llama2-7b-chat-v2 / llama2-13b-chat-v2 and use embedding with dashscope text-embedding-v1 / text-embedding-async-v1
- [embedding](https://github.com/kubeagi/arcadia/tree/main/examples/embedding) shows how to embedes your document to vector store with embedding service
- [rbac](https://github.com/kubeagi/arcadia/blob/main/examples/rbac/main.go) shows how to inquiry the security risks in your RBAC with AI.
- [beijing_gjj_bot](https://github.com/kubeagi/arcadia/tree/main/examples/beijing_gjj_bot) show how to build a chatgpt specialized for Beijing Provident Fund regulations