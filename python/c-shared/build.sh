#!/bin/bash -e

go build -buildmode=c-shared -o libwalle.so main.go
mkdir -p ../walle/lib
cp libwalle.so ../walle/lib/libwalle.so