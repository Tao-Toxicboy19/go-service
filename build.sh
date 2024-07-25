#! /bin/sh

docker build -t taotoxicboy/go-order-service:$1 --platform linux/amd64 . && \
    docker push taotoxicboy/go-order-service:$1