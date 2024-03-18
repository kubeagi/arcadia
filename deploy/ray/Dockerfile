# Use ray 2.9.3 and vllm 0.3.3
# Add vllm and tiktoken package
FROM rayproject/ray-ml:2.9.3-py39
# Official: https://pypi.org/simple
ARG PYTHON_INDEX_URL="https://pypi.mirrors.ustc.edu.cn/simple/"

RUN python3.9 -m pip install vllm tiktoken -i ${PYTHON_INDEX_URL}
