import json
import sys

import pyodbc
import pytest


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


@pytest.mark.skipif(sys.platform.startswith("win"), "Presumably need to fix the logfile path in windows?")
def test_connect_log(driver_name, tmp_path):
    logfile = tmp_path / "logfile.log"
    with pytest.raises(pyodbc.OperationalError) as exception:
        pyodbc.connect(f"Driver={driver_name};logfile={logfile}")

    assert exception.value.args[0] == "08001"
    assert "No Server specified" in exception.value.args[1]

    log_lines = [json.loads(line) for line in logfile.read_text().splitlines()]
    assert any("No Server specified" in log_line["error"] for log_line in log_lines)


def test_invalid_credentials(driver_name, httpserver):
    server = httpserver.url_for("")
    httpserver.expect_request("/api/user/token").respond_with_data(status=401)
    with pytest.raises(pyodbc.OperationalError) as exception:
        pyodbc.connect(
            f"Driver={driver_name};server={server};username=asdf;password=asdf"
        )

    assert exception.value.args[0] == "08001"
    assert "401" in exception.value.args[1]
