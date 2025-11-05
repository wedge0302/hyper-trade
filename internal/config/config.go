package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config holds credentials and endpoints for external services.
type Config struct {
	Binance     BinanceConfig     `json:"binance"`
	Hyperliquid HyperliquidConfig `json:"hyperliquid"`
}

// BinanceConfig contains credentials and endpoints required to communicate with Binance.
type BinanceConfig struct {
	APIKey       string `json:"api_key"`
	APISecret    string `json:"api_secret"`
	BaseURL      string `json:"base_url"`
	WebSocketURL string `json:"websocket_url"`
}

// HyperliquidConfig contains credentials and endpoints required to communicate with Hyperliquid.
type HyperliquidConfig struct {
	APIKey        string `json:"api_key"`
	APISecret     string `json:"api_secret"`
	BaseURL       string `json:"base_url"`
	OrderEndpoint string `json:"order_endpoint"`
}

// Load reads configuration values from a JSON file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}
