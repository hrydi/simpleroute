FROM golang:1.24 AS base-go

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

FROM node:lts-alpine3.21 AS base-node
ENV TZ=Asia/Jakarta
RUN apk update
RUN apk upgrade
RUN apk add ca-certificates && update-ca-certificates
RUN apk add --no-cache --update \
    tzdata
    
RUN rm -rf /var/cache/apk/* /tmp/*

FROM base-node AS dev-ui
ARG vite_port
ENV VITE_PORT=${vite_port:-4567}
ENV WORKDIR="/home/node/ui"
ENV PATH=$PATH:/home/node/.yarn/bin
USER root
RUN chown -R node:node /home/node
ADD ./example/ui/vue/*.json /tmp/
RUN cd /tmp && \
    yarn install && \
    mkdir -p ${WORKDIR} && \
    mv /tmp/node_modules ${WORKDIR}
WORKDIR ${WORKDIR}
ADD ./example/ui/vue/ ${WORKDIR}/
RUN chown -R node:node ${HOME}
RUN chown -R node:node ${WORKDIR}
USER node

FROM base-node AS pre-build
ARG vite_port
ENV VITE_PORT=${vite_port:-4567}
ENV WORKDIR="/home/node/ui"
ENV PATH=$PATH:/home/node/.yarn/bin
USER root
RUN chown -R node:node /home/node
RUN mkdir -p ${WORKDIR}
WORKDIR ${WORKDIR}
ADD ./example/ui/vue/ ${WORKDIR}/
RUN chown -R node:node ${HOME}
RUN chown -R node:node ${WORKDIR}
USER node

FROM dev-ui AS build-ui
RUN yarn build

FROM base-go AS build-go
WORKDIR /home/ubuntu/apps

COPY ./go.mod ./go.mod ./
RUN go mod download
COPY ./ .
COPY --from=build-ui /home/node/ui/dist /home/ubuntu/apps/example/ui/vue/dist
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -ldflags="-s -w" -o simplerouteapp ./example

FROM scratch AS prod
COPY --from=build-go /etc/passwd /etc/passwd
COPY --from=build-go /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-go --chown=1000:1000 /home/ubuntu/apps/simplerouteapp /simplerouteapp
USER ubuntu
ENTRYPOINT ["/simplerouteapp"]
