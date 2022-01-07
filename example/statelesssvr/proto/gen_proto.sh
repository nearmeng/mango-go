#!/bin/sh

trpc create -f -p echo.proto

mkdir -p tmp
cp ./echo/stub/echo/*.go ./tmp/

rm -rf echo
mv tmp echo