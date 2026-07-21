package strategy

import (
	"fmt"
	"math"

	"github.com/wferreirauy/binance-bot/config"
	"github.com/wferreirauy/binance-bot/exchange"
	"github.com/wferreirauy/binance-bot/indicator"
)

// entrySignals is the single, shared indicator snapshot used for entry
// decisions. Both the live trade loops (bull/bear/dynamic) and the
// registered backtest strategies derive their decisions from this struct,
// guaranteeing backtests exercise the same code path as live trading.
type entrySignals struct {
	Price     float64
	PrevPrice float64
	Tendency  string

	RSI        []float64 // smoothed per config (smooth-length 0/1 = raw RSI)
	MACDLine   []float64
	SignalLine []float64
	BB         indicator.BollingerBands

	DEMA      float64
	LowerBand float64
	UpperBand float64

	ADXVal    float64
	ADXStrong bool

	CurrentVolume   float64
	AvgVolume       float64
	VolumeConfirmed bool

	ATRVal float64
}

// computeEntrySignals derives every indicator needed for an entry decision
// from raw OHLCV data. Disabled indicators (period <= 0) or indicators
// without enough data keep permissive semantics: ADXStrong and
// VolumeConfirmed default to true, matching the historical live-loop
// behavior.
func computeEntrySignals(ohlcv *exchange.OHLCV, cfg *config.Config, tendency string) (*entrySignals, error) {
	if ohlcv == nil || len(ohlcv.Closes) < 2 {
		return nil, fmt.Errorf("entry signals: need at least 2 closes")
	}
	closes := ohlcv.Closes

	rsi := indicator.CalculateSmoothedRSI(closes, cfg.Indicators.Rsi.Length, cfg.Indicators.Rsi.SmoothLength)
	macdLine, signalLine := indicator.CalculateMACD(closes, cfg.Indicators.Macd.FastLength, cfg.Indicators.Macd.SlowLength, cfg.Indicators.Macd.SignalLength)
	if len(rsi) == 0 || len(macdLine) < 2 || len(signalLine) < 2 {
		return nil, fmt.Errorf("entry signals: insufficient candles for RSI/MACD")
	}
	bb, err := indicator.CalculateBollingerBands(closes, cfg.Indicators.BollingerBands.Length, cfg.Indicators.BollingerBands.Multiplier)
	if err != nil {
		return nil, fmt.Errorf("BollingerBands: %w", err)
	}
	if len(bb.LowerBand) == 0 || len(bb.UpperBand) == 0 {
		return nil, fmt.Errorf("entry signals: empty Bollinger bands")
	}
	dema := indicator.CalculateDEMA(closes, cfg.Indicators.Dema.Length)
	if len(dema) == 0 {
		return nil, fmt.Errorf("entry signals: empty DEMA")
	}

	sig := &entrySignals{
		Price:      closes[len(closes)-1],
		PrevPrice:  closes[len(closes)-2],
		Tendency:   tendency,
		RSI:        rsi,
		MACDLine:   macdLine,
		SignalLine: signalLine,
		BB:         bb,
		DEMA:       dema[len(dema)-1],
		LowerBand:  bb.LowerBand[len(bb.LowerBand)-1],
		UpperBand:  bb.UpperBand[len(bb.UpperBand)-1],
	}

	sig.ADXStrong = true
	if cfg.Indicators.Adx.Period > 0 {
		adx := indicator.CalculateADX(ohlcv.Highs, ohlcv.Lows, closes, cfg.Indicators.Adx.Period)
		if len(adx) > 0 {
			sig.ADXVal = adx[len(adx)-1]
			sig.ADXStrong = sig.ADXVal > float64(cfg.Indicators.Adx.Threshold)
		}
	}

	sig.VolumeConfirmed = true
	if cfg.Indicators.Volume.MaPeriod > 0 && len(ohlcv.Volumes) > 0 {
		volumeMA := indicator.CalculateSMA(ohlcv.Volumes, cfg.Indicators.Volume.MaPeriod)
		sig.CurrentVolume = ohlcv.Volumes[len(ohlcv.Volumes)-1]
		if len(volumeMA) > 0 {
			sig.AvgVolume = volumeMA[len(volumeMA)-1]
			sig.VolumeConfirmed = sig.CurrentVolume > sig.AvgVolume
		}
	}

	if cfg.Indicators.Atr.Period > 0 {
		atr := indicator.CalculateATR(ohlcv.Highs, ohlcv.Lows, closes, cfg.Indicators.Atr.Period)
		if len(atr) > 0 {
			sig.ATRVal = atr[len(atr)-1]
		}
	}
	return sig, nil
}

// computeEntrySignalsAt computes entry signals on the candle window [0, i].
// Backtests use this to replay decisions bar by bar on the same code path
// as live trading.
func computeEntrySignalsAt(ohlcv *exchange.OHLCV, i int, cfg *config.Config, tendency string) (*entrySignals, error) {
	if ohlcv == nil || i < 1 || i >= len(ohlcv.Closes) {
		return nil, fmt.Errorf("entry signals: index %d out of range", i)
	}
	window := &exchange.OHLCV{
		Opens:   sliceTo(ohlcv.Opens, i),
		Highs:   sliceTo(ohlcv.Highs, i),
		Lows:    sliceTo(ohlcv.Lows, i),
		Closes:  sliceTo(ohlcv.Closes, i),
		Volumes: sliceTo(ohlcv.Volumes, i),
	}
	return computeEntrySignals(window, cfg, tendency)
}

func sliceTo(s []float64, i int) []float64 {
	if i+1 > len(s) {
		return s
	}
	return s[:i+1]
}

// evaluateClassicEntry applies the classic all-conditions-must-hold entry
// rules for either direction and returns the per-condition breakdown for
// logging. AI approval is layered on top by the caller.
func evaluateClassicEntry(sig *entrySignals, cfg *config.Config, isBull bool) (bool, []entryCondition) {
	lastRSI := sig.RSI[len(sig.RSI)-1]
	lastMACD := sig.MACDLine[len(sig.MACDLine)-1]
	lastSignal := sig.SignalLine[len(sig.SignalLine)-1]
	prevMACD := sig.MACDLine[len(sig.MACDLine)-2]
	prevSignal := sig.SignalLine[len(sig.SignalLine)-2]
	distanceToUpper := math.Abs(sig.DEMA - sig.UpperBand)
	distanceToLower := math.Abs(sig.DEMA - sig.LowerBand)

	var rsiOk, macdCrossOk, tendOk, bbOk bool
	var conditions []entryCondition
	if isBull {
		rsiOk = lastRSI < float64(cfg.Indicators.Rsi.LowerLimit)
		macdCrossOk = prevMACD <= prevSignal && lastMACD > lastSignal
		tendOk = sig.Tendency == "up"
		bbOk = distanceToLower < distanceToUpper
		conditions = []entryCondition{
			{Name: fmt.Sprintf("RSI %.1f < %d", lastRSI, cfg.Indicators.Rsi.LowerLimit), Met: rsiOk},
			{Name: fmt.Sprintf("MACD bullish crossover (%.6f > %.6f)", lastMACD, lastSignal), Met: macdCrossOk},
			{Name: fmt.Sprintf("Tendency %s = up", sig.Tendency), Met: tendOk},
			{Name: fmt.Sprintf("Closer to lower BB (lower=%.4f, upper=%.4f)", distanceToLower, distanceToUpper), Met: bbOk},
			{Name: fmt.Sprintf("ADX strong (%.1f > %d)", sig.ADXVal, cfg.Indicators.Adx.Threshold), Met: sig.ADXStrong},
			{Name: fmt.Sprintf("Volume confirmed (%.0f > avg %.0f)", sig.CurrentVolume, sig.AvgVolume), Met: sig.VolumeConfirmed},
		}
	} else {
		rsiOk = lastRSI > float64(cfg.Indicators.Rsi.UpperLimit)
		macdCrossOk = prevMACD >= prevSignal && lastMACD < lastSignal
		tendOk = sig.Tendency == "down"
		bbOk = distanceToUpper < distanceToLower
		conditions = []entryCondition{
			{Name: fmt.Sprintf("RSI %.1f > %d", lastRSI, cfg.Indicators.Rsi.UpperLimit), Met: rsiOk},
			{Name: fmt.Sprintf("MACD bearish crossover (%.6f < %.6f)", lastMACD, lastSignal), Met: macdCrossOk},
			{Name: fmt.Sprintf("Tendency %s = down", sig.Tendency), Met: tendOk},
			{Name: fmt.Sprintf("Closer to upper BB (upper=%.4f, lower=%.4f)", distanceToUpper, distanceToLower), Met: bbOk},
			{Name: fmt.Sprintf("ADX strong (%.1f > %d)", sig.ADXVal, cfg.Indicators.Adx.Threshold), Met: sig.ADXStrong},
			{Name: fmt.Sprintf("Volume confirmed (%.0f > avg %.0f)", sig.CurrentVolume, sig.AvgVolume), Met: sig.VolumeConfirmed},
		}
	}
	met := rsiOk && macdCrossOk && tendOk && bbOk && sig.ADXStrong && sig.VolumeConfirmed
	return met, conditions
}

// scalpInputFromSignals adapts the shared signals to the scalp evaluator.
func scalpInputFromSignals(sig *entrySignals, cfg *config.Config, isBull bool, closes []float64) scalpEvalInput {
	return scalpEvalInput{
		IsBull: isBull, Cfg: cfg,
		Closes: closes, RSI: sig.RSI,
		MACDLine: sig.MACDLine, SignalLine: sig.SignalLine,
		BB: sig.BB, Tendency: sig.Tendency,
		ADXStrong: sig.ADXStrong, ADXVal: sig.ADXVal,
		CurrentVolume: sig.CurrentVolume, AvgVolume: sig.AvgVolume,
		ATRVal: sig.ATRVal, Price: sig.Price,
	}
}

// macdCrossLabel renders the MACD cross state for the TUI panel, keeping the
// per-direction display convention used by the trade loops: the label flips
// only on an actual crossover in the trade direction.
func macdCrossLabel(sig *entrySignals, isBull bool) string {
	lastMACD := sig.MACDLine[len(sig.MACDLine)-1]
	lastSignal := sig.SignalLine[len(sig.SignalLine)-1]
	prevMACD := sig.MACDLine[len(sig.MACDLine)-2]
	prevSignal := sig.SignalLine[len(sig.SignalLine)-2]
	if isBull {
		if prevMACD <= prevSignal && lastMACD > lastSignal {
			return "BULLISH"
		}
		return "BEARISH"
	}
	if prevMACD >= prevSignal && lastMACD < lastSignal {
		return "BEARISH"
	}
	return "BULLISH"
}