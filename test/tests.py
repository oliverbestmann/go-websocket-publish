from contextlib import closing

import pytest

from helpers import *


def test_404_if_stream_does_not_exists():
    with pytest.raises(websocket.WebSocketBadStatusException) as err:
        client_ws("/streams/" + random_stream())

    assert err.value.status_code == 404


def test_401_if_stream_exists_and_invalid_token():
    stream = random_stream()
    server_ws(path_stream(stream, suffix="/publish")).close()

    with pytest.raises(websocket.WebSocketBadStatusException) as err:
        client_ws(path_stream(stream, "invalid-token"))

    assert err.value.status_code == 401


def test_404_if_stream_is_deleted_again():
    stream = random_stream()
    server_ws(path_stream(stream, suffix="/publish")).close()
    server_api("DELETE", path_stream(stream))

    with pytest.raises(websocket.WebSocketBadStatusException) as err:
        client_ws("/streams/" + random_stream())

    assert err.value.status_code == 404


def test_okay_if_stream_exists_and_valid_token():
    stream = random_stream()
    server_ws(path_stream(stream, suffix="/publish")).close()

    token = server_api("POST", path_stream(stream, suffix="/tokens")).token
    client_ws(path_stream(stream, token))


def test_okay_if_stream_exists_and_token_rejected():
    stream = random_stream()
    server_ws(path_stream(stream, suffix="/publish")).close()
    token = server_api("POST", path_stream(stream, suffix="/tokens")).token
    client_ws(path_stream(stream, token))

    server_api("DELETE", path_stream(stream, token))

    with pytest.raises(websocket.WebSocketBadStatusException) as err:
        client_ws(path_stream(stream, token))

    assert err.value.status_code == 401


def test_receive_message_from_stream():
    stream = random_stream()
    server = server_ws(path_stream(stream, suffix="/publish"))
    token = server_api("POST", path_stream(stream, suffix="/tokens")).token
    client = client_ws(path_stream(stream, token))

    server.send("payload of the message")
    assert client.recv() == "payload of the message"

    server.send("a second message")
    assert client.recv() == "a second message"


def test_receive_message_from_stream_if_server_reconnects():
    stream = random_stream()
    server = server_ws(path_stream(stream, suffix="/publish"))
    token = server_api("POST", path_stream(stream, suffix="/tokens")).token
    client = client_ws(path_stream(stream, token))

    server.send("payload of the message")
    assert client.recv() == "payload of the message"

    # reconnect
    server.close()
    server = server_ws(path_stream(stream, suffix="/publish"))

    server.send("a second message")
    assert client.recv() == "a second message"


def test_close_client_stream_if_token_rejected():
    stream = random_stream()
    server_ws(path_stream(stream, suffix="/publish")).close()
    token = server_api("POST", path_stream(stream, suffix="/tokens")).token
    client = client_ws(path_stream(stream, token))

    server_api("DELETE", path_stream(stream, token))

    with pytest.raises(websocket.WebSocketConnectionClosedException):
        client.recv()


def test_server_does_not_buffer_too_much_if_client_lags():
    stream = random_stream()
    with closing(server_ws(path_stream(stream, suffix="/publish"))) as server:
        token = server_api("POST", path_stream(stream, suffix="/tokens")).token
        with closing(client_ws(path_stream(stream, token))) as client:
            N = 1024

            # send 10k messages
            for idx in range(1, N):
                server.send("x")

            server.send("last message")

            # check how many messages we get back
            count = 0
            while True:
                count += 1
                if client.recv() == "last message":
                    break

            assert count < N
