ARG GO_VER=1.21
ARG PY_VER=3.11

FROM golang:${GO_VER} AS builder
ARG GOPROXY=https://goproxy.cn,direct

# build arctl
WORKDIR /go/src
COPY cmd cmd
COPY pkg/ pkg/
COPY api/ api/
COPY apiserver/ apiserver/
COPY go.mod go.mod
COPY go.sum go.sum
RUN go env -w GOPROXY=${GOPROXY}
RUN go build -o arctl ./cmd/arctl

# python environment
FROM python:${PY_VER}-slim AS runner
ARG PACKAGE_REGISTRY="mirrors.tuna.tsinghua.edu.cn"
RUN sed -i 's/deb.debian.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apt/sources.list.d/debian.sources

ENV TZ=Asia/Shanghai
RUN export DEBIAN_FRONTEND=noninteractive \
    && apt-get update \
    && apt-get install -y tzdata \
    && ln -fs /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && dpkg-reconfigure --frontend noninteractive tzdata

# Official: https://pypi.org/simple
ARG PYTHON_INDEX_URL=https://pypi.mirrors.ustc.edu.cn/simple/

WORKDIR /usr/local/bin
COPY --from=builder /go/src/arctl ./arctl
RUN python -m pip install ragas langchain==0.0.354 -i ${PYTHON_INDEX_URL}

# build ragas-once by source code
COPY ./pypi/ragas_once ./ragas_once
WORKDIR /usr/local/bin/ragas_once
RUN python setup.py bdist_wheel -d /usr/local/bin/ragas_once/dist
RUN pip install dist/ragas_once-0.0.1-py3-none-any.whl