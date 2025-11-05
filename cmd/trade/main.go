package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"trade/internal/config"
	"trade/internal/exchange/binance"
	"trade/internal/exchange/hyperliquid"
	"trade/internal/strategy"
)

func main() {
	var (
		configPath string
		symbolsCSV string
		amplitude  float64
		amount     float64
		takeProfit float64
		stopLoss   float64
		ttl        time.Duration
		window     time.Duration
		leverage   float64
	)

	flag.StringVar(&configPath, "config", "config/config.json", "Path to the JSON config file")
	flag.StringVar(&symbolsCSV, "symbols", "", "Comma separated list of trading symbols (e.g. BTCUSDT,ETHUSDT)")
	flag.Float64Var(&amplitude, "amplitude", 0.0035, "Amplitude threshold used to trigger orders (0.0035 = 0.35%)")
	flag.Float64Var(&amount, "amount", 0.01, "Order size to submit when a signal is triggered")
	flag.Float64Var(&takeProfit, "take-profit", 0.20, "Take profit ratio (0.20 = 20%)")
	flag.Float64Var(&stopLoss, "stop-loss", 0.10, "Stop loss ratio (0.10 = 10%)")
	flag.DurationVar(&ttl, "ttl", 5*time.Minute, "Maximum time before a new order can be opened for the same symbol")
	flag.DurationVar(&window, "window", 500*time.Millisecond, "Lookback window to compute price amplitude")
	flag.Float64Var(&leverage, "leverage", 1, "Leverage to use when placing orders")
	flag.Parse()

	if symbolsCSV == "" {
		log.Fatal("missing required -symbols flag")
	}

	symbols := strings.Split(symbolsCSV, ",")

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	params := strategy.Params{
		Symbols:            symbols,
		AmplitudeThreshold: amplitude,
		OrderAmount:        amount,
		TakeProfitRatio:    takeProfit,
		StopLossRatio:      stopLoss,
		PositionTTL:        ttl,
		AmplitudeWindow:    window,
		Leverage:           leverage,
	}

	binanceClient := binance.NewClient(cfg.Binance)
	hyperliquidClient := hyperliquid.NewClient(cfg.Hyperliquid)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := strategy.Run(ctx, binanceClient, hyperliquidClient, params); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("strategy stopped: %v", err)
	}
}
