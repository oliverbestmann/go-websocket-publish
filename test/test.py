import hashlib
import uuid

import pytest
import requests
import websocket
from attrdict import AttrDict


def random_stream():
    return hashlib.md5(str(uuid.uuid4())).hexdigest()


def path_stream(stream, token=None, suffix=None):
    url = "/streams/" + stream
    if token:
        url += "/tokens/" + token
    if suffix:
        url += suffix
    return url


def server_api(method, path):
    url = "http://localhost:8080" + path
    response = requests.request(method, url)
    response.raise_for_status()
    if response.status_code == 204 or len(response.content) == 0:
        return {}
    else:
        return AttrDict(response.json())


def client_ws(path):
    return websocket.create_connection("ws://localhost:8081" + path, timeout=1)


def server_ws(path):
    return websocket.create_connection("ws://localhost:8080" + path, timeout=1)


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
