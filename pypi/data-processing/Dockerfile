FROM python:3.10.13-slim

ENV TZ=Asia/Shanghai

RUN sed -i 's/deb.debian.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apt/sources.list.d/debian.sources

RUN export DEBIAN_FRONTEND=noninteractive \
    && apt-get update \
    && apt-get install -y tzdata \
    && ln -fs /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && dpkg-reconfigure --frontend noninteractive tzdata \
    && apt-get install -y python3-distutils curl python3-pip \
    && apt-get install -y wget \
    && apt-get install -y opencc

RUN wget https://github.com/explosion/spacy-models/releases/download/zh_core_web_sm-3.5.0/zh_core_web_sm-3.5.0-py3-none-any.whl -O /tmp/zh_core_web_sm-3.5.0-py3-none-any.whl \
    && pip3 install /tmp/zh_core_web_sm-3.5.0-py3-none-any.whl -i https://pypi.org/simple \
    && rm /tmp/zh_core_web_sm-3.5.0-py3-none-any.whl

ENV DEFAULT_CONFIG=arcadia-config
ENV POD_NAMESPACE=arcadia

EXPOSE 28888

ADD . /arcadia_app/
WORKDIR /arcadia_app

RUN chmod 777 /arcadia_app/entrypoint.sh

RUN pip install -r requirements.txt
RUN playwright install &&  playwright install-deps

ENTRYPOINT ["./entrypoint.sh"]


