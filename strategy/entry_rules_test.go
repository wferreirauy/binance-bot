package strategy

import (
	"testing"

	"github.com/wferreirauy/binance-bot/config"
	"github.com/wferreirauy/binance-bot/exchange"
)

// testConfig returns a minimal config with the default indicator thresholds
// used across the entry-rule tests.
func testConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Indicators.Rsi.UpperLimit = 70
	cfg.Indicators.Rsi.LowerLimit = 30
	cfg.Indicators.Rsi.Length = 14
	cfg.Indicators.Macd.FastLength = 12
	cfg.Indicators.Macd.SlowLength = 26
	cfg.Indicators.Macd.SignalLength = 9
	cfg.Indicators.BollingerBands.Length = 20
	cfg.Indicators.BollingerBands.Multiplier = 2.0
	cfg.Indicators.Dema.Length = 9
	cfg.Indicators.Adx.Threshold = 25
	return cfg
}

// bullSignals builds an entrySignals snapshot where every classic bull
// condition holds: RSI oversold, fresh MACD bullish crossover, tendency up,
// DEMA nearer the lower band, strong ADX and confirmed volume.
func bullSignals() *entrySignals {
	return &entrySignals{
		Price:    100,
		Tendency: "up",
		RSI:      []float64{35, 28},
		// prev: macd <= signal, last: macd > signal → bullish crossover
		MACDLine:        []float64{-0.5, 0.2},
		SignalLine:      []float64{-0.3, 0.1},
		DEMA:            101,
		LowerBand:       99,
		UpperBand:       110,
		ADXVal:          30,
		ADXStrong:       true,
		CurrentVolume:   1500,
		AvgVolume:       1000,
		VolumeConfirmed: true,
	}
}

// bearSignals mirrors bullSignals for the bear direction.
func bearSignals() *entrySignals {
	return &entrySignals{
		Price:    100,
		Tendency: "down",
		RSI:      []float64{65, 75},
		// prev: macd >= signal, last: macd < signal → bearish crossover
		MACDLine:        []float64{0.5, -0.2},
		SignalLine:      []float64{0.3, -0.1},
		DEMA:            109,
		LowerBand:       90,
		UpperBand:       110,
		ADXVal:          30,
		ADXStrong:       true,
		CurrentVolume:   1500,
		AvgVolume:       1000,
		VolumeConfirmed: true,
	}
}

func TestClassicBullEntryAllConditionsMet(t *testing.T) {
	met, conditions := evaluateClassicEntry(bullSignals(), testConfig(), true)
	if !met {
		t.Fatalf("expected bull entry, conditions: %+v", conditions)
	}
	if len(conditions) != 6 {
		t.Fatalf("expected 6 condition entries, got %d", len(conditions))
	}
}

func TestClassicBearEntryAllConditionsMet(t *testing.T) {
	met, conditions := evaluateClassicEntry(bearSignals(), testConfig(), false)
	if !met {
		t.Fatalf("expected bear entry, conditions: %+v", conditions)
	}
}

func TestClassicEntryBlockedByWeakADX(t *testing.T) {
	sig := bullSignals()
	sig.ADXStrong = false
	if met, _ := evaluateClassicEntry(sig, testConfig(), true); met {
		t.Fatal("weak ADX must block classic entry")
	}
}

func TestClassicEntryBlockedByUnconfirmedVolume(t *testing.T) {
	sig := bullSignals()
	sig.VolumeConfirmed = false
	if met, _ := evaluateClassicEntry(sig, testConfig(), true); met {
		t.Fatal("unconfirmed volume must block classic entry")
	}
}

func TestClassicEntryBlockedByTendencyMismatch(t *testing.T) {
	sig := bullSignals()
	sig.Tendency = "down"
	if met, _ := evaluateClassicEntry(sig, testConfig(), true); met {
		t.Fatal("bull entry requires up tendency")
	}
}

func TestClassicEntryBlockedWithoutMACDCross(t *testing.T) {
	sig := bullSignals()
	// already above signal on both bars → no fresh crossover
	sig.MACDLine = []float64{0.3, 0.4}
	sig.SignalLine = []float64{0.1, 0.1}
	if met, _ := evaluateClassicEntry(sig, testConfig(), true); met {
		t.Fatal("no fresh MACD crossover must block classic entry")
	}
}

func TestComputeEntrySignalsPermissiveDefaults(t *testing.T) {
	cfg := testConfig()
	// ADX and volume disabled (Period/MaPeriod = 0) → permissive gates.
	n := 60
	ohlcv := &exchange.OHLCV{}
	for i := 0; i < n; i++ {
		price := 100 + float64(i%7)
		ohlcv.Opens = append(ohlcv.Opens, price)
		ohlcv.Highs = append(ohlcv.Highs, price+1)
		ohlcv.Lows = append(ohlcv.Lows, price-1)
		ohlcv.Closes = append(ohlcv.Closes, price)
		ohlcv.Volumes = append(ohlcv.Volumes, 1000)
	}
	sig, err := computeEntrySignals(ohlcv, cfg, "up")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sig.ADXStrong {
		t.Fatal("disabled ADX must default to strong")
	}
	if !sig.VolumeConfirmed {
		t.Fatal("disabled volume MA must default to confirmed")
	}
	if sig.Price != ohlcv.Closes[n-1] {
		t.Fatalf("price mismatch: %v != %v", sig.Price, ohlcv.Closes[n-1])
	}
}

func TestComputeEntrySignalsEnforcesADXWhenConfigured(t *testing.T) {
	cfg := testConfig()
	cfg.Indicators.Adx.Period = 14
	cfg.Indicators.Adx.Threshold = 99 // impossible bar for flat series
	n := 60
	ohlcv := &exchange.OHLCV{}
	for i := 0; i < n; i++ {
		price := 100 + float64(i%3)
		ohlcv.Opens = append(ohlcv.Opens, price)
		ohlcv.Highs = append(ohlcv.Highs, price+0.5)
		ohlcv.Lows = append(ohlcv.Lows, price-0.5)
		ohlcv.Closes = append(ohlcv.Closes, price)
		ohlcv.Volumes = append(ohlcv.Volumes, 1000)
	}
	sig, err := computeEntrySignals(ohlcv, cfg, "up")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sig.ADXStrong {
		t.Fatal("flat market cannot beat ADX threshold 99; gate must hold")
	}
}

func TestComputeEntrySignalsRejectsShortSeries(t *testing.T) {
	ohlcv := &exchange.OHLCV{Closes: []float64{100}}
	if _, err := computeEntrySignals(ohlcv, testConfig(), "up"); err == nil {
		t.Fatal("expected error for insufficient candles")
	}
}

func TestComputeEntrySignalsAtMatchesWindow(t *testing.T) {
	cfg := testConfig()
	n := 80
	full := &exchange.OHLCV{}
	for i := 0; i < n; i++ {
		price := 100 + float64(i)*0.1
		full.Opens = append(full.Opens, price)
		full.Highs = append(full.Highs, price+1)
		full.Lows = append(full.Lows, price-1)
		full.Closes = append(full.Closes, price)
		full.Volumes = append(full.Volumes, 1000+float64(i))
	}
	at := 50
	sigAt, err := computeEntrySignalsAt(full, at, cfg, "up")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	window := &exchange.OHLCV{
		Opens: full.Opens[:at+1], Highs: full.Highs[:at+1], Lows: full.Lows[:at+1],
		Closes: full.Closes[:at+1], Volumes: full.Volumes[:at+1],
	}
	sigWin, err := computeEntrySignals(window, cfg, "up")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sigAt.Price != sigWin.Price || sigAt.DEMA != sigWin.DEMA ||
		sigAt.RSI[len(sigAt.RSI)-1] != sigWin.RSI[len(sigWin.RSI)-1] {
		t.Fatal("windowed computation must match direct computation on the same slice")
	}
}