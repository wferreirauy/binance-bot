package strategy

import (
	"fmt"
	"math"
	"sort"

	"github.com/wferreirauy/binance-bot/config"
	"github.com/wferreirauy/binance-bot/exchange"
	"github.com/wferreirauy/binance-bot/indicator"
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
	closes := ohlcv.Closes[:i+1]
	dema := indicator.CalculateDEMA(closes, cfg.Indicators.Dema.Length)
	rsi := indicator.CalculateRSI(closes, cfg.Indicators.Rsi.Length)
	macdLine, signalLine := indicator.CalculateMACD(closes, cfg.Indicators.Macd.FastLength, cfg.Indicators.Macd.SlowLength, cfg.Indicators.Macd.SignalLength)
	bb, err := indicator.CalculateBollingerBands(closes, cfg.Indicators.BollingerBands.Length, cfg.Indicators.BollingerBands.Multiplier)
	if err != nil || len(dema) == 0 || len(rsi) == 0 || len(macdLine) < 2 || len(signalLine) < 2 || len(bb.LowerBand) == 0 {
		return SignalHold
	}
	price := closes[len(closes)-1]
	if snapshot.Position {
		if price <= snapshot.EntryPrice*(1-snapshot.Config.Backtest.FeePct/100) {
			return SignalHold
		}
		return SignalSell
	}
	currentDema := dema[len(dema)-1]
	lowerBand := bb.LowerBand[len(bb.LowerBand)-1]
	upperBand := bb.UpperBand[len(bb.UpperBand)-1]
	distanceToUpper := math.Abs(currentDema - upperBand)
	distanceToLower := math.Abs(currentDema - lowerBand)
	macdCrossOk := macdLine[len(macdLine)-2] <= signalLine[len(signalLine)-2] &&
		macdLine[len(macdLine)-1] > signalLine[len(signalLine)-1]
	if rsi[len(rsi)-1] < float64(cfg.Indicators.Rsi.LowerLimit) &&
		macdCrossOk &&
		snapshot.Tendency == "up" &&
		distanceToLower < distanceToUpper {
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
	closes := ohlcv.Closes[:i+1]
	dema := indicator.CalculateDEMA(closes, cfg.Indicators.Dema.Length)
	rsi := indicator.CalculateRSI(closes, cfg.Indicators.Rsi.Length)
	macdLine, signalLine := indicator.CalculateMACD(closes, cfg.Indicators.Macd.FastLength, cfg.Indicators.Macd.SlowLength, cfg.Indicators.Macd.SignalLength)
	bb, err := indicator.CalculateBollingerBands(closes, cfg.Indicators.BollingerBands.Length, cfg.Indicators.BollingerBands.Multiplier)
	if err != nil || len(dema) == 0 || len(rsi) == 0 || len(macdLine) == 0 || len(signalLine) == 0 || len(bb.LowerBand) == 0 {
		return SignalHold
	}
	score := 0
	currentDema := dema[len(dema)-1]
	lowerBand := bb.LowerBand[len(bb.LowerBand)-1]
	upperBand := bb.UpperBand[len(bb.UpperBand)-1]
	if rsi[len(rsi)-1] < float64(cfg.Indicators.Rsi.LowerLimit) {
		score++
	}
	if macdLine[len(macdLine)-1] > signalLine[len(signalLine)-1] {
		score++
	}
	if snapshot.Tendency == "up" {
		score++
	}
	if math.Abs(currentDema-lowerBand) < math.Abs(currentDema-upperBand) {
		score++
	}
	minScore := cfg.ScalpMode.MinScore
	if minScore <= 0 {
		minScore = 3
	}
	if score >= minScore {
		return SignalBuy
	}
	return SignalHold
}

func init() {
	RegisterStrategy(classicBullStrategy{})
	RegisterStrategy(scalpBullStrategy{})
}
