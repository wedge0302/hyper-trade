package strategy

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Metrics stores the latest observations for each configured symbol.
type Metrics struct {
	mu               sync.RWMutex
	data             map[string]SymbolMetrics
	summary          Summary
	currentPositions []Position
	positionHistory  []Position
	logs             []LogEntry
}

const (
	maxHistoryEntries = 200
	maxLogEntries     = 200
	maxOpenPositions  = 50
)

// NewMetrics initialises the metrics store for the given symbols.
func NewMetrics(symbols []Symbol) *Metrics {
	data := make(map[string]SymbolMetrics, len(symbols))
	for _, sym := range symbols {
		data[sym.Binance] = SymbolMetrics{
			BinanceSymbol:     sym.Binance,
			HyperliquidSymbol: sym.Hyperliquid,
		}
	}

	return &Metrics{
		data: data,
		summary: Summary{
			StartTime: time.Now(),
		},
	}
}

// SetInitialBalance updates the account baseline used for performance metrics.
func (m *Metrics) SetInitialBalance(balance float64) {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if balance < 0 {
		balance = 0
	}

	m.summary.InitialBalance = balance
	if m.summary.TotalProfit == 0 {
		m.summary.CurrentBalance = balance
	} else {
		m.summary.CurrentBalance = balance + m.summary.TotalProfit
	}
}

// Update records the latest price action for a symbol.
func (m *Metrics) Update(symbol Symbol, price, amplitude float64, updateTime, lastOrder time.Time) {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := symbol.Binance
	metric, ok := m.data[key]
	if !ok {
		metric = SymbolMetrics{
			BinanceSymbol:     symbol.Binance,
			HyperliquidSymbol: symbol.Hyperliquid,
		}
	}

	metric.LastPrice = price
	metric.LastAmplitude = amplitude
	metric.LastUpdate = updateTime
	metric.LastOrderTime = lastOrder
	m.data[key] = metric
}

// RecordOrder captures a placed order for monitoring purposes.
func (m *Metrics) RecordOrder(symbol Symbol, side string, price, size float64) {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	position := Position{
		ID:         fmt.Sprintf("%s-%d", symbol.Hyperliquid, now.UnixNano()),
		Symbol:     symbol.Hyperliquid,
		Side:       side,
		Size:       size,
		EntryPrice: price,
		OpenedAt:   now,
		Status:     "Open",
	}

	notional := price * size
	if notional > 0 {
		m.summary.TotalVolume += notional
	}
	m.summary.TradeCount++
	m.summary.CurrentBalance = m.summary.InitialBalance + m.summary.TotalProfit

	m.currentPositions = append(m.currentPositions, position)
	if len(m.currentPositions) > maxOpenPositions {
		m.currentPositions = m.currentPositions[len(m.currentPositions)-maxOpenPositions:]
	}
	m.positionHistory = append(m.positionHistory, position)
	if len(m.positionHistory) > maxHistoryEntries {
		m.positionHistory = m.positionHistory[len(m.positionHistory)-maxHistoryEntries:]
	}
}

// AddLog records a message for display in the monitoring UI.
func (m *Metrics) AddLog(message string) {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	entry := LogEntry{Time: time.Now(), Message: message}
	m.logs = append(m.logs, entry)
	if len(m.logs) > maxLogEntries {
		m.logs = m.logs[len(m.logs)-maxLogEntries:]
	}
}

// Snapshot returns a copy of the metrics for dashboard rendering.
func (m *Metrics) Snapshot() DashboardSnapshot {
	if m == nil {
		return DashboardSnapshot{}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]SymbolMetrics, 0, len(m.data))
	for _, metric := range m.data {
		out = append(out, metric)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].BinanceSymbol < out[j].BinanceSymbol
	})

	summary := m.summary
	summary.CurrentBalance = summary.InitialBalance + summary.TotalProfit

	currentPositions := make([]Position, len(m.currentPositions))
	copy(currentPositions, m.currentPositions)

	history := make([]Position, len(m.positionHistory))
	copy(history, m.positionHistory)

	logs := make([]LogEntry, len(m.logs))
	copy(logs, m.logs)

	return DashboardSnapshot{
		Summary:          summary,
		Symbols:          out,
		CurrentPositions: currentPositions,
		PositionHistory:  history,
		Logs:             logs,
	}
}
