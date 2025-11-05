package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
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
	"trade/internal/server"
	"trade/internal/strategy"
)

func main() {
	var (
		configPath  string
		symbolsFile string
		amplitude   float64
		amount      float64
		takeProfit  float64
		stopLoss    float64
		ttl         time.Duration
		window      time.Duration
		leverage    float64
		initialBal  float64
		monitorAddr string
	)

	flag.StringVar(&configPath, "config", "config/config.json", "Path to the JSON config file")
	flag.StringVar(&symbolsFile, "symbols-file", "config/symbols.json", "Path to the JSON file containing symbol mappings")
	flag.Float64Var(&amplitude, "amplitude", 0.0035, "Amplitude threshold used to trigger orders (0.0035 = 0.35%)")
	flag.Float64Var(&amount, "amount", 10.0, "Order notional in USDT to submit when a signal is triggered")
	flag.Float64Var(&takeProfit, "take-profit", 0.20, "Take profit ratio (0.20 = 20%)")
	flag.Float64Var(&stopLoss, "stop-loss", 0.10, "Stop loss ratio (0.10 = 10%)")
	flag.DurationVar(&ttl, "ttl", 5*time.Minute, "Minimum wait time before a new order can be opened for the same symbol")
	flag.DurationVar(&window, "window", 500*time.Millisecond, "Lookback window to compute price amplitude")
	flag.Float64Var(&leverage, "leverage", 5.0, "Leverage to use when placing orders")
	flag.Float64Var(&initialBal, "initial-balance", 1000.0, "Initial account balance used for reporting")
	flag.StringVar(&monitorAddr, "monitor-addr", ":8080", "HTTP listen address for the monitoring UI")
	flag.Parse()

	symbols, err := loadSymbols(symbolsFile)
	if err != nil {
		log.Fatalf("load symbols: %v", err)
	}

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
		InitialBalance:     initialBal,
	}

	settings, err := strategy.NewSettings(params)
	if err != nil {
		log.Fatalf("initialise settings: %v", err)
	}

	metrics := strategy.NewMetrics(settings.Symbols())
	metrics.SetInitialBalance(settings.Snapshot().InitialBalance)

	}

	binanceClient := binance.NewClient(cfg.Binance)
	hyperliquidClient := hyperliquid.NewClient(cfg.Hyperliquid)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	monitor := server.New(monitorAddr, settings, metrics)
	go func() {
		if err := monitor.Run(ctx); err != nil {
			log.Printf("monitoring server stopped: %v", err)
		}
	}()

	if err := strategy.Run(ctx, binanceClient, hyperliquidClient, settings, metrics); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("strategy stopped: %v", err)
	}
}

func loadSymbols(path string) ([]strategy.Symbol, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open symbols file: %w", err)
	}
	defer file.Close()

	var payload []struct {
		Binance     string `json:"binance"`
		Hyperliquid string `json:"hyperliquid"`
	}

	if err := json.NewDecoder(file).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode symbols: %w", err)
	}

	if len(payload) == 0 {
		return nil, fmt.Errorf("symbols file %s is empty", path)
	}

	symbols := make([]strategy.Symbol, 0, len(payload))
	for _, entry := range payload {
		binance := strings.ToUpper(strings.TrimSpace(entry.Binance))
		hyper := strings.TrimSpace(entry.Hyperliquid)
		if binance == "" || hyper == "" {
			continue
		}

		symbols = append(symbols, strategy.Symbol{Binance: binance, Hyperliquid: hyper})
	}

	if len(symbols) == 0 {
		return nil, fmt.Errorf("no valid symbols found in %s", path)
	}

	return symbols, nil
}
	if err := strategy.Run(ctx, binanceClient, hyperliquidClient, params); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("strategy stopped: %v", err)
	}
}
