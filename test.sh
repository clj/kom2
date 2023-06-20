#!/usr/bin/env bash

set -e
set -x

./build.sh
python3 test_server.py &
trap "trap - SIGTERM && kill -- $!" SIGINT SIGTERM EXIT
sleep 1
isql kom2test < test.sql
