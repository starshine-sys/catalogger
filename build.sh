#!/bin/sh
CGO_ENABLED=0 go build -v -x -ldflags="-extldflags=-static -X github.com/starshine-sys/catalogger/events.GitVer=${git rev-parse --short HEAD}" -tags netgo