import os
import platform
import socket

import pytest


@pytest.fixture
def port():
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    s.bind(("localhost", 0))
    yield s.getsockname()
    s.close()


@pytest.fixture(scope="session")
def driver_name():
    if name := os.getenv("KOM2_DRIVER_NAME"):
        return name
    return {"Linux": "kom2.so", "Darwin": "kom2.dylib", "Windows": "kom2.dll"}[
        platform.system()
    ]
