# syntax=docker/dockerfile:1

FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o trade ./cmd/trade

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/trade /usr/local/bin/trade
COPY config/config.json /app/config.json

ENTRYPOINT ["trade"]
CMD ["-symbols", "BTCUSDT", "-config", "/app/config.json"]
