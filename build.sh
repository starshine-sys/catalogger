#!/bin/sh
CGO_ENABLED=0 ${GOCMD:-go} build -v -o catalogger -ldflags="-buildid= -X github.com/starshine-sys/catalogger/common.Version=`git rev-parse --short HEAD`" ./cmd/catalogger/
