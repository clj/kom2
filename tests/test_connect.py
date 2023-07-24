import json
import platform
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


@pytest.fixture
def driver_name():
    return {"Linux": "kom2.so", "Darwin": "kom2.dylib", "Windows": "kom2.dll"}[
        platform.system()
    ]


def test_connect_without_server(driver_name):
    with pytest.raises(pyodbc.OperationalError) as exception:
        pyodbc.connect(f"Driver={driver_name}")

    assert exception.value.args[0] == "08001"
    assert "No Server specified" in exception.value.args[1]


def test_connect_no_credentials(driver_name):
    with pytest.raises(pyodbc.OperationalError) as exception:
        pyodbc.connect(f"Driver={driver_name};server=asdf")

    assert exception.value.args[0] == "08001"
    assert "No APIToken or Username+Password specified" in exception.value.args[1]


def test_connect_invalid_server(driver_name):
    with pytest.raises(pyodbc.OperationalError) as exception:
        pyodbc.connect(f"Driver={driver_name};server=asdf://asdf;apitoken=asdf")

    assert exception.value.args[0] == "08001"
    assert "Error updating category list" in exception.value.args[1]


def test_connect_no_server(driver_name, port):
    hostname, portnumber = port
    with pytest.raises(pyodbc.OperationalError) as exception:
        pyodbc.connect(
            f"Driver={driver_name};server=http://{hostname}:{portnumber};apitoken=asdf;httptimeout=1ms"
        )

    assert exception.value.args[0] == "08001"
    assert "Error updating category list" in exception.value.args[1]


def test_connect_log(driver_name, tmp_path):
    logfile = tmp_path / "logfile.log"
    with pytest.raises(pyodbc.OperationalError) as exception:
        pyodbc.connect(f"Driver={driver_name};logfile={logfile}")

    assert exception.value.args[0] == "08001"
    assert "No Server specified" in exception.value.args[1]

    log_lines = [json.loads(line) for line in logfile.read_text().splitlines()]
    assert any("No Server specified" in log_line["error"] for log_line in log_lines)
