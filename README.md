go-websocket-publish
====================

Build
-----

Install glide using `go get github.com/Masterminds/glide`,
then run the `build.sh` shell script.

Usage
-----

Start the binary. It will open a webserver on port 8081. You can then
stream data via websocket to `/streams/some-id/publish`, read this
data using `/streams/some-id` and delete the stream by performing
a http DELETE request on `/streams/some-id`.

