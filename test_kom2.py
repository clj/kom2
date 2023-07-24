import json
import socket

import pyodbc
import pytest


@pytest.fixture
def port():
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    s.bind(("localhost", 0))
    yield s.getsockname()
    s.close()


def test_connect_without_server():
    with pytest.raises(pyodbc.OperationalError) as exception:
        pyodbc.connect("Driver=kom2.dylib")

    assert exception.value.args[0] == "08001"
    assert "No Server specified" in exception.value.args[1]


def test_connect_no_credentials():
    with pytest.raises(pyodbc.OperationalError) as exception:
        pyodbc.connect("Driver=kom2.dylib;server=asdf")

    assert exception.value.args[0] == "08001"
    assert "No APIToken or Username+Password specified" in exception.value.args[1]


def test_connect_invalid_server():
    with pytest.raises(pyodbc.OperationalError) as exception:
        pyodbc.connect("Driver=kom2.dylib;server=asdf://asdf;apitoken=asdf")

    assert exception.value.args[0] == "08001"
    assert "Error updating category list" in exception.value.args[1]


def test_connect_no_server(port):
    hostname, portnumber = port
    with pytest.raises(pyodbc.OperationalError) as exception:
        pyodbc.connect(
            f"Driver=kom2.dylib;server=http://{hostname}:{portnumber};apitoken=asdf;httptimeout=1ms"
        )

    assert exception.value.args[0] == "08001"
    assert "Error updating category list" in exception.value.args[1]


def test_connect_log(tmp_path):
    logfile = tmp_path / "logfile.log"
    with pytest.raises(pyodbc.OperationalError) as exception:
        pyodbc.connect(f"Driver=kom2.dylib;logfile={logfile}")

    assert exception.value.args[0] == "08001"
    assert "No Server specified" in exception.value.args[1]

    log_line = json.loads(logfile.read_text())
    assert "No Server specified" in log_line["error"]
