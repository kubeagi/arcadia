from openai import OpenAI
import streamlit as st


page_names_to_funcs = {
    "—": "",
    "Plotting Demo": "plotting_demo",
    "Mapping Demo": "mapping_demo",
    "DataFrame Demo": "data_frame_demo"
}

with st.sidebar:
    demo_name = st.sidebar.selectbox(
        "Choose a demo", page_names_to_funcs.keys())
    "[View the source code](https://github.com/kubeagi/arcadia/examples/playground/playground.py)"

st.title("💬 Chat for Arcadia")
st.caption("🚀 A playground powered by Steamlit")

if "messages" not in st.session_state:
    st.session_state["messages"] = [
        {"role": "assistant", "content": "How can I help you?"}]

for msg in st.session_state.messages:
    st.chat_message(msg["role"]).write(msg["content"])

if prompt := st.chat_input():
    client = OpenAI(api_key="fake-key",
                    base_url="http://arcadia-fastchat.172.22.96.167.nip.io/v1/")
    st.session_state.messages.append({"role": "user", "content": prompt})
    st.chat_message("user").write(prompt)
    response = client.chat.completions.create(
        model="baichuan2-7b-worker-baichuan-playground", messages=st.session_state.messages)
    msg = response.choices[0].message.content
    st.session_state.messages.append({"role": "assistant", "content": msg})
    st.chat_message("assistant").write(msg)
