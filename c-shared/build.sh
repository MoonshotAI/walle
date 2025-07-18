#!/bin/bash -e

go build -buildmode=c-shared -o libwalle.so main.go