package hyperliquid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"trade/internal/config"
)

// Client handles authenticated requests to the Hyperliquid exchange.
type Client struct {
	cfg        config.HyperliquidConfig
	httpClient *http.Client
}

// NewClient creates a new Hyperliquid client instance.
func NewClient(cfg config.HyperliquidConfig) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// LimitOrderRequest describes the payload sent when submitting an order.
type LimitOrderRequest struct {
	Symbol          string  `json:"symbol"`
	Side            string  `json:"side"`
	Price           float64 `json:"price"`
	Size            float64 `json:"size"`
	Leverage        float64 `json:"leverage"`
	TakeProfitRatio float64 `json:"take_profit_ratio"`
	StopLossRatio   float64 `json:"stop_loss_ratio"`
	TimeInForce     string  `json:"time_in_force"`
	PositionTTL     string  `json:"position_ttl"`
}

// PlaceLimitOrder submits a limit order to the Hyperliquid exchange.
func (c *Client) PlaceLimitOrder(ctx context.Context, req LimitOrderRequest) error {
	endpoint := fmt.Sprintf("%s%s", strings.TrimSuffix(c.cfg.BaseURL, "/"), c.cfg.OrderEndpoint)

	payload := map[string]any{
		"api_key":    c.cfg.APIKey,
		"api_secret": c.cfg.APISecret,
		"order":      req,
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal order payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create order request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("execute order request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("hyperliquid rejected order with status %s", resp.Status)
	}

	return nil
}
