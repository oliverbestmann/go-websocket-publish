import hashlib
import uuid

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
