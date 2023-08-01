# FROM python:3.12.0b4-alpine3.18
# FROM ubuntu:22.04
FROM nvidia/cuda:11.7.1-runtime-ubuntu22.04

RUN sed -i 's/archive.ubuntu.com/mirrors.tuna.tsinghua.edu.cn/g' /etc/apt/sources.list
RUN sed -i 's/security.ubuntu.com/mirrors.tuna.tsinghua.edu.cn/g' /etc/apt/sources.list

RUN apt update -y && apt upgrade -y && apt install -y python3 curl
RUN apt install python3-pip -y
# RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
# RUN apk add curl py3-pip py3-setuptools
# RUN curl https://bootstrap.pypa.io/get-pip.py -o get-pip.py
# RUN python3 get-pip.py
RUN pip3 install fschat -i https://pypi.mirrors.ustc.edu.cn/simple/

