#!/bin/sh
set -e

glide install

# build for linux
GOOS=linux CGO_ENABLED=0 go build -a

# build for windows
# GOOS=windows GOARCH=386 go build -o go-websocket-publish.exe

docker build -t oliverbestmann/go-websocket-publish .
docker push oliverbestmann/go-websocket-publish
