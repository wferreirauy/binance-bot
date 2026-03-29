# syntax=docker/dockerfile:1

FROM golang:1.26.0

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o /binance-bot

ENTRYPOINT ["/binance-bot"]

