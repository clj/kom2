#!/usr/bin/env bash

case $(uname -s) in
    Darwin*)   ext="dylib";;
    MINGW64*)  ext="dll";;
    *)         ext="so";;
esac

set -x

go build -buildmode=c-shared -o kom2.$ext $*
