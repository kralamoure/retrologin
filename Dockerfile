FROM golang:1.17.6-bullseye AS builder

WORKDIR /app
COPY . .

RUN go install -v ./...

FROM ubuntu:20.04

LABEL org.opencontainers.image.source="https://github.com/kralamoure/retrologin"

RUN apt-get update && apt-get upgrade -y

WORKDIR /app
COPY --from=builder /go/bin/ .

ENTRYPOINT ["./retrologin"]
