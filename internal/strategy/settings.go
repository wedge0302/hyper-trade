package strategy

import (
	"errors"
	"sync"
	"time"
)

const (
	defaultAmplitudeThreshold = 0.0035
	defaultOrderAmount        = 10.0
	defaultTakeProfit         = 0.20
	defaultStopLoss           = 0.10
	defaultLeverage           = 5.0
	defaultAmplitudeWindow    = 500 * time.Millisecond
	defaultInitialBalance     = 1000.0
)

// Settings manages concurrent access to the strategy parameters.
type Settings struct {
	mu     sync.RWMutex
	params Params
}

// NewSettings constructs a Settings instance using the provided parameters.
func NewSettings(params Params) (*Settings, error) {
	if len(params.Symbols) == 0 {
		return nil, errors.New("no trading symbols configured")
	}

	s := &Settings{params: params}
	s.applyDefaultsLocked()
	return s, nil
}

// Snapshot returns a copy of the current parameters.
func (s *Settings) Snapshot() Params {
	s.mu.RLock()
	defer s.mu.RUnlock()

	params := s.params
	if len(params.Symbols) > 0 {
		symbols := make([]Symbol, len(params.Symbols))
		copy(symbols, params.Symbols)
		params.Symbols = symbols
	}

	return params
}

// Symbols returns the configured symbol mappings.
func (s *Settings) Symbols() []Symbol {
	s.mu.RLock()
	defer s.mu.RUnlock()

	symbols := make([]Symbol, len(s.params.Symbols))
	copy(symbols, s.params.Symbols)
	return symbols
}

// Update safely mutates the underlying parameters.
func (s *Settings) Update(mutator func(p *Params) error) (Params, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := mutator(&s.params); err != nil {
		return Params{}, err
	}

	s.applyDefaultsLocked()
	return s.params, nil
}

func (s *Settings) applyDefaultsLocked() {
	if s.params.AmplitudeThreshold <= 0 {
		s.params.AmplitudeThreshold = defaultAmplitudeThreshold
	}
	if s.params.OrderAmount <= 0 {
		s.params.OrderAmount = defaultOrderAmount
	}
	if s.params.TakeProfitRatio <= 0 {
		s.params.TakeProfitRatio = defaultTakeProfit
	}
	if s.params.StopLossRatio <= 0 {
		s.params.StopLossRatio = defaultStopLoss
	}
	if s.params.Leverage <= 0 {
		s.params.Leverage = defaultLeverage
	}
	if s.params.AmplitudeWindow <= 0 {
		s.params.AmplitudeWindow = defaultAmplitudeWindow
	}
	if s.params.InitialBalance <= 0 {
		s.params.InitialBalance = defaultInitialBalance
	}
}
