FROM golang:1.22.2-alpine3.18
WORKDIR /go/src/app

RUN go install github.com/pressly/goose/v3/cmd/goose@v3.20.0
