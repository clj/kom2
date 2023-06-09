#!/usr/bin/env bash

src=$1
version=$2

cd $src
for lib in kom2-*; do
    ext=${lib##*.}
    filename=${lib%%.*}
    filename=${filename/darwin/macos}
    zip=${filename/kom2/kicad-odbc-middleware2-$version}.zip
    cp $lib kom2.$ext
    zip $zip kom2.$ext
    rm kom2.$ext
done
