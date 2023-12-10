import streamlit as st
import requests
import os

with st.sidebar:
    server_url = st.text_input("服务 graphql-server/go-server 请求地址, 默认为 http://arcadia-apiserver.kubeagi-system.svc:8081/chat", key="url")
    conversion_id = st.text_input("如果想继续的话，可以输入上次的conversion_id，留空表示新对话", key="conversion_id")

st.title("💬 Chat with kubeagi")
st.caption("🚀 A chatbot powered by Kubeagi")
if "messages" not in st.session_state:
    st.session_state["messages"] = [{"role": "assistant", "content": "您好，您可以问我任何关于考勤制度的问题，很高心为您服务。"}]

if "first_show" not in st.session_state:
    st.session_state["first_show"] = True

if not server_url:
    server_url = "http://arcadia-apiserver.kubeagi-system.svc:8081/chat"

for msg in st.session_state.messages:
    st.chat_message(msg["role"]).write(msg["content"])

if prompt := st.chat_input():
    response = requests.post(server_url,
    json={"query":prompt,"response_mode":"blocking","conversion_id":conversion_id,"app_name":"chat-with-kaoqin-kb", "app_namespace":"kubeagi-system"})
    st.session_state.messages.append({"role": "user", "content": prompt})
    st.chat_message("user").write(prompt)
    msg = response.json()["message"]
    conversion_id = response.json()["conversion_id"]

    if st.session_state["first_show"]:
        st.info('这次聊天的 conversion_id 是： '+conversion_id, icon="ℹ️")
        st.session_state["first_show"] = False

    st.session_state.messages.append({"role": "assistant", "content": msg})
    st.chat_message("assistant").write(msg)
