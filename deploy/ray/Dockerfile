# Use ray 2.9.0 built using python 3.9.x
# Add vllm and tiktoken package
FROM rayproject/ray-ml:2.9.0-py39
# Official: https://pypi.org/simple
ARG PYTHON_INDEX_URL="https://pypi.mirrors.ustc.edu.cn/simple/"

RUN python3.9 -m pip install vllm tiktoken -i ${PYTHON_INDEX_URL}
