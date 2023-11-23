FROM python:3.10.13-slim

ENV TZ=Asia/Shanghai

RUN sed -i 's/deb.debian.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apt/sources.list.d/debian.sources

RUN export DEBIAN_FRONTEND=noninteractive \
    && apt-get update \
    && apt-get install -y tzdata \
    && ln -fs /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && dpkg-reconfigure --frontend noninteractive tzdata \
    && apt-get install -y python3-distutils curl python3-pip \
    && apt-get install -y wget

RUN wget https://github.com/explosion/spacy-models/releases/download/zh_core_web_sm-3.5.0/zh_core_web_sm-3.5.0-py3-none-any.whl -O /tmp/zh_core_web_sm-3.5.0-py3-none-any.whl \
    && pip3 install /tmp/zh_core_web_sm-3.5.0-py3-none-any.whl -i https://pypi.org/simple \
    && rm /tmp/zh_core_web_sm-3.5.0-py3-none-any.whl


ENV MINIO_ACCESSKEY=minio_accesskey
ENV MINIO_SECRETKEY=minio_secretkey
ENV MINIO_API_URL=localhost:9000
ENV MINIO_SECURE=False
ENV MINIO_DATASET_PREFIX=dataset

ENV ZHIPUAI_API_KEY=xxxxx

ENV KNOWLEDGE_CHUNK_SIZE=500
ENV KNOWLEDGE_CHUNK_OVERLAP=50

ENV PG_HOST=localhost
ENV PG_PORT=5432
ENV PG_USER=postgres
ENV PG_PASSWORD=xxxxx
ENV PG_DATABASE=data_process

EXPOSE 28888

ADD . /arcadia_app/
WORKDIR /arcadia_app

RUN chmod 777 /arcadia_app/entrypoint.sh

RUN pip install -r requirements.txt

ENTRYPOINT ["./entrypoint.sh"]


