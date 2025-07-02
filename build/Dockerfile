FROM golang:1.24 AS build

ARG ci_commit_tag='development'
ARG DEBIAN_FRONTEND=noninteractive

ENV CI_COMMIT_TAG=${ci_commit_tag}
ENV TZ="Asia/Jakarta"

WORKDIR /home/ubuntu/apps

RUN apt update && apt upgrade -y \
    && apt-get install -y software-properties-common tzdata curl
    
RUN ln -fs /usr/share/zoneinfo/Asia/Jakarta /etc/localtime \
    && dpkg-reconfigure -f noninteractive tzdata

RUN useradd -s /bin/bash -m ubuntu \
    && usermod -aG www-data ubuntu \
    && mkdir -p /home/ubuntu/apps \
    && chown -R ubuntu:ubuntu /home/ubuntu

USER ubuntu
