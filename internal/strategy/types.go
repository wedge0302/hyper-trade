package strategy

import "time"

// Params defines the tunable strategy settings supplied by the user.
type Params struct {
	Symbols            []Symbol
	Symbols            []string
	AmplitudeThreshold float64
	OrderAmount        float64
	TakeProfitRatio    float64
	StopLossRatio      float64
	PositionTTL        time.Duration
	AmplitudeWindow    time.Duration
	Leverage           float64
	InitialBalance     float64
}

// TickerUpdate represents a single price update emitted by a market data feed.
type TickerUpdate struct {
	Symbol string
	Price  float64
	Time   time.Time
}

// Symbol maps an asset's identifiers between Binance and Hyperliquid.
type Symbol struct {
	Binance     string
	Hyperliquid string
}

// SymbolMetrics tracks the latest market and execution observations for a symbol.
type SymbolMetrics struct {
	BinanceSymbol     string
	HyperliquidSymbol string
	LastPrice         float64
	LastAmplitude     float64
	LastUpdate        time.Time
	LastOrderTime     time.Time
}

// Summary captures the aggregated financial performance of the running strategy.
type Summary struct {
	StartTime      time.Time
	InitialBalance float64
	CurrentBalance float64
	TotalProfit    float64
	TotalVolume    float64
	TradeCount     int
}

// Position represents a simplified view of an opened or closed trade.
type Position struct {
	ID         string
	Symbol     string
	Side       string
	Size       float64
	EntryPrice float64
	OpenedAt   time.Time
	ClosedAt   time.Time
	Status     string
	Profit     float64
}

// LogEntry records notable runtime events for display in the dashboard.
type LogEntry struct {
	Time    time.Time
	Message string
}

// DashboardSnapshot bundles the monitoring data required by the HTTP UI.
type DashboardSnapshot struct {
	Summary          Summary
	Symbols          []SymbolMetrics
	CurrentPositions []Position
	PositionHistory  []Position
	Logs             []LogEntry
}
