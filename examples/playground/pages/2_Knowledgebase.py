import streamlit as st
from k8s import client


kubeEnv = client.KubeEnv(namespace="")

with st.sidebar:
    st.session_state['embedder'] = st.sidebar.selectbox(
        "Choose a Embedder",
        kubeEnv.list_embedders(namespace="").keys())


with st.sidebar:
    st.session_state['embedder'] = st.sidebar.selectbox(
        "Choose a Vector",
        kubeEnv.list_vectorstores(namespace="").keys())


st.title("💬 KnowledgeBase")
