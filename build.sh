#!/bin/sh
CGO_ENABLED=0 go build -v -ldflags="-X github.com/starshine-sys/catalogger/events.GitVer=`git rev-parse --short HEAD`"