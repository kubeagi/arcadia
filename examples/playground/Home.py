import streamlit as st


st.title("💬 Arcadia Playground")

st.markdown(
    """
        Playground is an streamlit app built specifically for
        Arcadia LLMOps. It is quite easy to use.

        ### Run playground

        ```shell
        cd playground

        pip install -r requirements.txt

        # Kubeconfig path
        export KUBECONFIG=~/.kube/config
        # Namespace this playground watches on
        export NAMESPACE=playground

        export MINIO_ENDPOINT="arcadia-minio-api.172.22.96.167.nip.io"
        export MINIO_ACCESS_KEY=admin
        export MINIO_SECRET_KEY=Passw0rd!

        
        streamlit run Home.py
        ```

        ### Want to learn more?

        - Check out [arcadia](https://github.com/kubeagi/arcadia)
        - Contirbute to Arcadia with [contribute guide](https://github.com/kubeagi/arcadia/blob/main/CONTRIBUTING.md)
        - Request a feature in our [project](https://github.com/kubeagi/arcadia/issues)
        - Ask a question in our [community
          forums](https://github.com/kubeagi/arcadia/discussions)
    """
)
