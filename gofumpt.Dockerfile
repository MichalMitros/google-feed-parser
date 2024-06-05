# Just for demonstration purposes, to make formatting independent from local Go version.
FROM golang:1.22.2-alpine3.19
RUN go install mvdan.cc/gofumpt@v0.6.0
