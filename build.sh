#!/usr/bin/env bash

case $(uname -s) in
    Darwin*)   ext="dylib";;
    MINGW64*) 
        ext="dll";
        go build ./inst/inst.go;
        ;;
    *)         ext="so";;
esac

set -x

go build -buildmode=c-shared -o kom2.$ext "$@"
