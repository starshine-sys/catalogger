# a lot of this is yoinked from YAGPDB's Dockerfile because we barely know how this works </3

FROM golang:latest AS builder

WORKDIR /build
COPY . ./
RUN go mod download -x
ENV CGO_ENABLED 0
RUN go build -v -o catalogger -ldflags="-X github.com/starshine-sys/catalogger/common.Version=`git rev-parse --short HEAD`" ./cmd/catalogger/

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /build/catalogger catalogger

CMD ["/app/catalogger", "bot"]
