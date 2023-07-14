#!/usr/bin/env bash


case $(uname -s) in
    Darwin*)   ext="dylib";;
    MINGW64*)
        ext="dll";
        set -x
        go build ./inst/inst.go;
        set +x
        ;;
    *)         ext="so";;
esac

set -x

go build \
    -buildmode=c-shared \
    -o kom2.$ext \
    -ldflags="-X main.LogFile=\"$KOM2_LOGFILE\" \
              -X main.LogFormat=\"$KOM2_LOGFORMAT\" \
              -X main.LogLevel=\"$KOM2_LOGLEVEL\"" \
    "$@"
