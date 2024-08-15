#! /bin/sh

docker build -t taotoxicboy/go-order-cronjob-service:$1 --platform linux/amd64 . && \
    docker push taotoxicboy/go-order-cronjob-service:$1