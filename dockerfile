FROM golang:1.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o app.out ./cmd

FROM ubuntu:latest

RUN apt-get update && apt-get install -y iproute2 net-tools && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/app.out .
COPY cfg.yaml .
COPY entrypoint.sh .


ENTRYPOINT ["./app.out"]