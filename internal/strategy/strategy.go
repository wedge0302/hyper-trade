package strategy

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"trade/internal/exchange/hyperliquid"
)

// TickerSource defines the behaviour required from a market data provider.
type TickerSource interface {
	SubscribeTicker(ctx context.Context, symbol string) (<-chan TickerUpdate, <-chan error)
}

// OrderExecutor defines the methods required to place orders on an exchange.
type OrderExecutor interface {
	PlaceLimitOrder(ctx context.Context, req hyperliquid.LimitOrderRequest) error
}

// Run executes the trading strategy.
func Run(ctx context.Context, source TickerSource, executor OrderExecutor, settings *Settings, metrics *Metrics) error {
	symbols := settings.Symbols()
	if len(symbols) == 0 {
		return fmt.Errorf("no symbols provided")
	}

func Run(ctx context.Context, source TickerSource, executor OrderExecutor, params Params) error {
	if len(params.Symbols) == 0 {
		return fmt.Errorf("no symbols provided")
	}

	window := params.AmplitudeWindow
	if window <= 0 {
		window = 500 * time.Millisecond
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	errCh := make(chan error, len(symbols))

	for _, sym := range symbols {
		symbol := sym

		wg.Add(1)
		go func(sym Symbol) {
			defer wg.Done()
			if err := runForSymbol(ctx, source, executor, settings, metrics, sym); err != nil {
	errCh := make(chan error, len(params.Symbols))

	for _, sym := range params.Symbols {
		symbol := strings.ToUpper(strings.TrimSpace(sym))
		if symbol == "" {
			continue
		}

		wg.Add(1)
		go func(sym string) {
			defer wg.Done()
			if err := runForSymbol(ctx, source, executor, params, sym, window); err != nil {
				select {
				case errCh <- err:
				default:
				}
				cancel()
			}
		}(symbol)
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

func runForSymbol(ctx context.Context, source TickerSource, executor OrderExecutor, settings *Settings, metrics *Metrics, symbol Symbol) error {
	updates, errs := source.SubscribeTicker(ctx, symbol.Binance)
func runForSymbol(ctx context.Context, source TickerSource, executor OrderExecutor, params Params, symbol string, window time.Duration) error {
	updates, errs := source.SubscribeTicker(ctx, symbol)

	var (
		priceWindow []TickerUpdate
		lastOrder   time.Time
	)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err, ok := <-errs:
			if !ok {
				return nil
			}
			if err != nil {
				return fmt.Errorf("%s ticker stream failed: %w", symbol.Binance, err)
				return fmt.Errorf("%s ticker stream failed: %w", symbol, err)
			}
		case update, ok := <-updates:
			if !ok {
				return nil
			}

			params := settings.Snapshot()
			window := params.AmplitudeWindow
			if window <= 0 {
				window = 500 * time.Millisecond
			}

			priceWindow = append(priceWindow, update)
			cutoff := update.Time.Add(-window)

			idx := 0
			for ; idx < len(priceWindow); idx++ {
				if priceWindow[idx].Time.After(cutoff) {
					break
				}
			}
			priceWindow = priceWindow[idx:]

			amplitude := calculateAmplitude(priceWindow)

			metrics.Update(symbol, update.Price, amplitude, update.Time, lastOrder)

			if amplitude < params.AmplitudeThreshold {
				continue
			}

			if params.PositionTTL > 0 && !lastOrder.IsZero() && time.Since(lastOrder) < params.PositionTTL {
				continue
			}

			side := determineSide(priceWindow)

			ttlValue := ""
			if params.PositionTTL > 0 {
				ttlValue = params.PositionTTL.String()
			}

			if update.Price <= 0 {
				continue
			}

			orderSize := params.OrderAmount / update.Price
			if orderSize <= 0 {
				continue
			}

			order := hyperliquid.LimitOrderRequest{
				Symbol:          symbol.Hyperliquid,
				Side:            side,
				Price:           update.Price,
				Size:            orderSize,
			order := hyperliquid.LimitOrderRequest{
				Symbol:          symbol,
				Side:            side,
				Price:           update.Price,
				Size:            params.OrderAmount,
				Leverage:        params.Leverage,
				TakeProfitRatio: params.TakeProfitRatio,
				StopLossRatio:   params.StopLossRatio,
				TimeInForce:     "GTC",
				PositionTTL:     ttlValue,
			}

			if err := executor.PlaceLimitOrder(ctx, order); err != nil {
				return fmt.Errorf("place order for %s: %w", symbol.Hyperliquid, err)
			}

			lastOrder = time.Now()
			metrics.Update(symbol, update.Price, amplitude, update.Time, lastOrder)
			metrics.RecordOrder(symbol, side, update.Price, orderSize)
			metrics.AddLog(fmt.Sprintf("%s order submitted for %s at %.4f (size %.4f)", side, symbol.Hyperliquid, update.Price, orderSize))
				return fmt.Errorf("place order for %s: %w", symbol, err)
			}

			lastOrder = time.Now()
		}
	}
}

func calculateAmplitude(points []TickerUpdate) float64 {
	if len(points) < 2 {
		return 0
	}

	minPrice := points[0].Price
	maxPrice := points[0].Price

	for _, point := range points[1:] {
		if point.Price < minPrice {
			minPrice = point.Price
		}
		if point.Price > maxPrice {
			maxPrice = point.Price
		}
	}

	if minPrice == 0 {
		return 0
	}

	return (maxPrice - minPrice) / minPrice
}

func determineSide(points []TickerUpdate) string {
	if len(points) < 2 {
		return "buy"
	}

	first := points[0].Price
	last := points[len(points)-1].Price

	if last >= first {
		return "buy"
	}
	return "sell"
}
