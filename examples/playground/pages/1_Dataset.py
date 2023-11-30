import streamlit as st
from k8s.custom_resources import (
    Dataset, VersionedDataset
)


def submit_create_dataset(name: str, contentType: str):
    # create dataset
    Dataset().create(
        namespace="",
        name=name,
        contentType=contentType
    )
    # create v1 for dataset
    VersionedDataset().create(
        namespace="", dataset=name, name=name + "-v1", version="v1")


def submit_create_versioneddataset(dataset: str, name: str, version: str):
    VersionedDataset().create(
        namespace="", dataset=dataset, name=name, version=version)


with st.sidebar:
    # dataset selector
    options = list(Dataset().list(namespace=""))
    options.insert(0, "New a dataset")
    ds = st.sidebar.selectbox(
        "Choose a Dataset",
        options)

    if ds == 'New a dataset':
        name = st.text_input("Enter the new Dataset name...")
        contentType = st.selectbox("Choose Content Type",
                                   ['text', 'image', 'video'])
        st.button('Create', key='button_create_dataset',
                  on_click=submit_create_dataset, args=(name, contentType))
        ds = name

    # versioneddataset selector
    options = list(VersionedDataset().list(
        namespace="", label_selector="arcadia.kubeagi.k8s.com.cn/owner="+ds))
    options.insert(0, "New a version")
    vds = st.sidebar.selectbox(
        "Choose a Version",
        options)
    if vds == 'New a version':
        version = st.text_input("Enter the new version")
        vds = ds+"-" + version
        st.button('Create', key='button_create_versioneddataset',
                  on_click=submit_create_versioneddataset, args=(ds, vds, version))

    # cache to state
    st.session_state['dataset'] = ds
    st.session_state['versioneddataset'] = vds

st.title("💬 Dataset")

uploaded_file = st.file_uploader("Choose a file to Dataset!")
if uploaded_file is not None:
    bytes_data = uploaded_file.getvalue()
    st.write(bytes_data)


# table shows all files in this versioned dataset
st.table()
