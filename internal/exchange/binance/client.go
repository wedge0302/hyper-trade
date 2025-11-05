package binance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"trade/internal/config"
	"trade/internal/strategy"
	"trade/internal/ws"
)

// Client handles market data subscriptions with Binance.
type Client struct {
	cfg config.BinanceConfig
}

// NewClient instantiates a Binance client using the provided configuration.
func NewClient(cfg config.BinanceConfig) *Client {
	return &Client{cfg: cfg}
}

// SubscribeTicker subscribes to the live ticker stream for the provided symbol.
func (c *Client) SubscribeTicker(ctx context.Context, symbol string) (<-chan strategy.TickerUpdate, <-chan error) {
	updates := make(chan strategy.TickerUpdate)
	errs := make(chan error, 1)

	go func() {
		defer close(updates)
		defer close(errs)

		streamURL := fmt.Sprintf("%s/%s@ticker", strings.TrimSuffix(c.cfg.WebSocketURL, "/"), strings.ToLower(symbol))
		conn, err := ws.Dial(ctx, streamURL)
		if err != nil {
			errs <- fmt.Errorf("dial binance websocket: %w", err)
			return
		}
		defer conn.Close()

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			message, err := conn.ReadMessage()
			if err != nil {
				if errors.Is(err, io.EOF) {
					errs <- fmt.Errorf("websocket closed: %w", err)
				} else {
					errs <- fmt.Errorf("read message: %w", err)
				}
				return
			}

			var payload tickerPayload
			if err := json.Unmarshal(message, &payload); err != nil {
				errs <- fmt.Errorf("unmarshal ticker payload: %w", err)
				return
			}

			price, err := payload.Price()
			if err != nil {
				errs <- fmt.Errorf("parse price: %w", err)
				return
			}

			updates <- strategy.TickerUpdate{
				Symbol: symbol,
				Price:  price,
				Time:   payload.EventTime(),
			}
		}
	}()

	return updates, errs
}

type tickerPayload struct {
	EventTimeMs int64  `json:"E"`
	Symbol      string `json:"s"`
	LastPrice   string `json:"c"`
}

func (p tickerPayload) Price() (float64, error) {
	return strconv.ParseFloat(p.LastPrice, 64)
}

func (p tickerPayload) EventTime() time.Time {
	if p.EventTimeMs == 0 {
		return time.Now().UTC()
	}
	return time.UnixMilli(p.EventTimeMs)
}
