FROM golang:1.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o app.out ./cmd

FROM ubuntu:latest
ENV CONFIG_PATH=/app/config/cfg.yaml
ENV KEY_PATH=/app/data/keys

RUN apt-get update
RUN apt-get install -y iproute2 net-tools
RUN rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/app.out .
VOLUME config config
VOLUME data data

ENTRYPOINT ["/bin/bash", "-c", "./app.out --cfg $CONFIG_PATH --key $KEY_PATH"]