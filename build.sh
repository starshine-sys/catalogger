#!/bin/sh
CGO_ENABLED=0 go build -v -ldflags="-X github.com/starshine-sys/catalogger/common.Version=`git rev-parse --short HEAD`"