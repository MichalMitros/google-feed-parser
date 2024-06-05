FROM golang:1.22.2-alpine3.18 AS builder
WORKDIR /go/src/app

ENV CGO_ENABLED=1
ARG VERSION="n/a"

RUN apk --no-cache add git=2.40.1-r0 build-base=0.5-r3 && \
    go install github.com/cespare/reflex@latest && \
    go install github.com/pressly/goose/v3/cmd/goose@v3.20.0 && \
    go install github.com/go-jet/jet/v2/cmd/jet@v2.11.1

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -o /google-feed-parser \
    -ldflags "-X 'main.Version=$VERSION'" \
    /go/src/app/cmd/parser

FROM alpine:3.18.6 AS certs
RUN apk add --no-cache ca-certificates=20240226-r0

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /google-feed-parser /
USER 9000
ENTRYPOINT [ "/google-feed-parser" ]
