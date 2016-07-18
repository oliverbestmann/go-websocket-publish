FROM scratch

EXPOSE 8080
EXPOSE 8081

COPY /go-websocket-publish /
ENTRYPOINT ["/go-websocket-publish"]
