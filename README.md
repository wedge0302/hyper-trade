# Trade

High-frequency trading prototype written in Go. The service listens to Binance ticker streams over WebSockets and places limit orders on Hyperliquid when the intrawindow price amplitude exceeds a configurable threshold.

## Features

- Concurrent ticker subscriptions for multiple symbols using goroutines.
- Configurable trading parameters (symbols, notional order size, amplitude threshold, take-profit/stop-loss ratios, leverage, and order cooldown).
- Automatic limit order submissions to Hyperliquid with embedded take-profit and stop-loss ratios.
- Built-in monitoring dashboard with live metrics and in-browser parameter editing.
- Dockerized build for easy deployment.

## Getting started

### Prerequisites

- Go 1.21+
- Binance and Hyperliquid API credentials

### Configuration

Credentials and API endpoints live in `config/config.json`. **Do not commit sensitive keys.**

```json
{
  "binance": {
    "api_key": "your-binance-api-key",
    "api_secret": "your-binance-api-secret",
    "base_url": "https://api.binance.com",
    "websocket_url": "wss://stream.binance.com:9443/ws"
  },
  "hyperliquid": {
    "api_key": "your-hyperliquid-api-key",
    "api_secret": "your-hyperliquid-api-secret",
    "base_url": "https://api.hyperliquid.com",
    "order_endpoint": "/orders"
  }
}
```

You can create environment-specific overrides such as `config/config.secrets.json` (ignored by git).

### Running locally

Symbols are configured in `config/symbols.json` using the following shape so each exchange receives the correct identifier:

```json
[
  { "binance": "BTCUSDT", "hyperliquid": "BTC-PERP" },
  { "binance": "ETHUSDT", "hyperliquid": "ETH-PERP" }
]
```

Run the strategy from the command line:

```bash
go run ./cmd/trade \
  -config config/config.json \
  -symbols-file config/symbols.json \
  -amount 10 \
  -amplitude 0.0035 \
  -take-profit 0.2 \
  -stop-loss 0.1 \
  -leverage 5 \
  -ttl 1m \
  -monitor-addr :8080
```

Flags:

- `-symbols-file`: path to the JSON list of `{binance, hyperliquid}` symbol mappings.
- `-amount`: notional order size in USDT per trade (defaults to `10`).
- `-amplitude`: amplitude threshold (0.0035 = 0.35%).
- `-take-profit`: take profit ratio (0.20 = 20%).
- `-stop-loss`: stop loss ratio (0.10 = 10%).
- `-leverage`: leverage multiplier (defaults to `5`).
- `-ttl`: minimum cooldown before the same symbol can trigger another order.
- `-window`: lookback window for amplitude calculation (default `500ms`).
- `-monitor-addr`: HTTP listen address for the monitoring UI (default `:8080`).
- `-config`: path to the JSON configuration file.

### Monitoring dashboard

The embedded dashboard is served on the `-monitor-addr` address. It displays the latest price, amplitude, and order timestamps for each symbol and lets you adjust core strategy parameters (amplitude threshold, order notional, take-profit/stop-loss ratios, leverage, TTL, and amplitude window) without restarting the process.

### Docker

Build and run using Docker:

```bash
docker build -t trade .
docker run --rm \
  -v $(pwd)/config/config.json:/app/config.json \
  -v $(pwd)/config/symbols.json:/app/symbols.json \
  trade \
  -symbols-file /app/symbols.json \
  -amount 10
```

## Development

```bash
make build   # Build binary to ./bin/trade
make test    # Run unit tests (if present)
make tidy    # Update go.sum dependencies
```

## Disclaimer

This project is a technical prototype. It omits risk management controls required for production trading systems.
