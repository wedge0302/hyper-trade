package strategy

import "time"

// Params defines the tunable strategy settings supplied by the user.
type Params struct {
	Symbols            []string
	AmplitudeThreshold float64
	OrderAmount        float64
	TakeProfitRatio    float64
	StopLossRatio      float64
	PositionTTL        time.Duration
	AmplitudeWindow    time.Duration
	Leverage           float64
}

// TickerUpdate represents a single price update emitted by a market data feed.
type TickerUpdate struct {
	Symbol string
	Price  float64
	Time   time.Time
}
