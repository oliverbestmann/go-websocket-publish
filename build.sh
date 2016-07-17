#!/bin/sh
set -e

glide install

# build nativley
go build -a

# build for windows
GOOS=windows GOARCH=386 go build -o go-websocket-publish.exe
