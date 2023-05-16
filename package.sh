#!/usr/bin/env bash

src=$1
version=$2

cd $src
for lib in kom2-*; do
    ext=${lib##*.}
    filename=${lib%%.*}
    filename=${filename/darwin/macos}
    zip=${filename/kom/kicad-odbc-middleware2-$version}.zip
    cp $lib kom.$ext
    zip $zip kom.$ext
    rm kom.$ext
done
