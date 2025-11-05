BINARY := trade

.PHONY: build run tidy test

build:
go build -o bin/$(BINARY) ./cmd/trade

run:
go run ./cmd/trade -symbols BTCUSDT

tidy:
go mod tidy

test:
go test ./...
