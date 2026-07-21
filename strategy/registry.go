package strategy

import (
	"fmt"
	"sort"

	"github.com/wferreirauy/binance-bot/config"
	"github.com/wferreirauy/binance-bot/exchange"
)

type Signal string

const (
	SignalBuy  Signal = "BUY"
	SignalSell Signal = "SELL"
	SignalHold Signal = "HOLD"
)

type MarketSnapshot struct {
	Symbol     string
	Index      int
	OHLCV      *exchange.OHLCV
	Tendency   string
	Config     *config.Config
	Position   bool
	EntryPrice float64
}

type StrategyDefinition interface {
	Name() string
	Decide(snapshot MarketSnapshot) Signal
}

var registeredStrategies = map[string]StrategyDefinition{}

func RegisterStrategy(def StrategyDefinition) {
	registeredStrategies[def.Name()] = def
}

func GetStrategy(name string) (StrategyDefinition, error) {
	def, ok := registeredStrategies[name]
	if !ok {
		return nil, fmt.Errorf("unknown strategy %q; available: %v", name, StrategyNames())
	}
	return def, nil
}

func StrategyNames() []string {
	names := make([]string, 0, len(registeredStrategies))
	for name := range registeredStrategies {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

type classicBullStrategy struct{}

func (classicBullStrategy) Name() string { return "classic-bull" }

func (classicBullStrategy) Decide(snapshot MarketSnapshot) Signal {
	cfg := snapshot.Config
	ohlcv := snapshot.OHLCV
	i := snapshot.Index
	if i < 2 || i >= len(ohlcv.Closes) {
		return SignalHold
	}
	price := ohlcv.Closes[i]
	if snapshot.Position {
		if price <= snapshot.EntryPrice*(1-snapshot.Config.Backtest.FeePct/100) {
			return SignalHold
		}
		return SignalSell
	}
	// Same decision path as the live loops: shared signals + shared rules.
	sig, err := computeEntrySignalsAt(ohlcv, i, cfg, snapshot.Tendency)
	if err != nil {
		return SignalHold
	}
	if met, _ := evaluateClassicEntry(sig, cfg, true); met {
		return SignalBuy
	}
	return SignalHold
}

type scalpBullStrategy struct{}

func (scalpBullStrategy) Name() string { return "scalp-bull" }

func (scalpBullStrategy) Decide(snapshot MarketSnapshot) Signal {
	cfg := snapshot.Config
	ohlcv := snapshot.OHLCV
	i := snapshot.Index
	if i < 2 || i >= len(ohlcv.Closes) {
		return SignalHold
	}
	price := ohlcv.Closes[i]
	if snapshot.Position {
		if price >= snapshot.EntryPrice*(1+cfg.Backtest.FeePct/100) {
			return SignalSell
		}
		return SignalHold
	}
	// Same decision path as the live loops: shared signals feed the scalp
	// evaluator, so backtests honor ADX/volume exactly like live trading.
	sig, err := computeEntrySignalsAt(ohlcv, i, cfg, snapshot.Tendency)
	if err != nil {
		return SignalHold
	}
	eval := evaluateScalp(scalpInputFromSignals(sig, cfg, true, ohlcv.Closes[:i+1]))
	if eval.RegimeBlocked || eval.ExtremeBlocked {
		return SignalHold
	}
	if eval.Score >= eval.MinScore {
		return SignalBuy
	}
	return SignalHold
}

func init() {
	RegisterStrategy(classicBullStrategy{})
	RegisterStrategy(scalpBullStrategy{})
}
