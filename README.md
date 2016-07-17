go-websocket-publish
====================

Docker
------

Run the docker container using 
```
docker run --rm -p 8081:8081 oliverbestmann/go-websocket-publish:latest
```

Validate by checking the list of active streams:
```
curl localhost:8081/streams
```

Usage
-----

Start the binary. It will open a webserver on port 8081. You can then
stream data via websocket to `/streams/some-id/publish`, read this
data using `/streams/some-id` and delete the stream by performing
a http DELETE request on `/streams/some-id`.


Build
-----

Install glide using `go get github.com/Masterminds/glide`,
then run the `build.sh` shell script.
