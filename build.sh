#!/usr/bin/env bash

case $(uname -s) in
    Darwin*)   ext="dylib";;
    *)         ext="so";;
esac

set -x

echo $(uname -s)

go build -v -buildmode=c-shared -o kom2.$ext
