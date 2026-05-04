package indicator

import (
	"math"
	"testing"
)

// TestCalculateRSI_AllUp verifies that a strictly monotonically increasing
// price series does not trigger a div-by-zero (avgLoss == 0) and instead
// returns the canonical RSI = 100.
func TestCalculateRSI_AllUp(t *testing.T) {
	prices := []float64{10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25}
	rsi := CalculateRSI(prices, 14)
	if len(rsi) == 0 {
		t.Fatal("expected non-empty RSI for all-up series")
	}
	for i, v := range rsi {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Fatalf("RSI[%d] = %v; expected finite value", i, v)
		}
		if v != 100 {
			t.Fatalf("RSI[%d] = %v; expected 100 for all-up series", i, v)
		}
	}
}

// TestCalculateRSI_AllDown verifies the symmetric all-down case.
func TestCalculateRSI_AllDown(t *testing.T) {
	prices := []float64{25, 24, 23, 22, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10}
	rsi := CalculateRSI(prices, 14)
	if len(rsi) == 0 {
		t.Fatal("expected non-empty RSI for all-down series")
	}
	for i, v := range rsi {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Fatalf("RSI[%d] = %v; expected finite value", i, v)
		}
		if v != 0 {
			t.Fatalf("RSI[%d] = %v; expected 0 for all-down series", i, v)
		}
	}
}

// TestCalculateRSI_Mixed sanity-checks that a mixed series produces values in
// the (0, 100) open interval.
func TestCalculateRSI_Mixed(t *testing.T) {
	prices := []float64{44, 44.34, 44.09, 44.15, 43.61, 44.33, 44.83, 45.10, 45.42, 45.84, 46.08, 45.89, 46.03, 45.61, 46.28, 46.28, 46.0, 46.03, 46.41, 46.22}
	rsi := CalculateRSI(prices, 14)
	if len(rsi) == 0 {
		t.Fatal("expected RSI values")
	}
	last := rsi[len(rsi)-1]
	if last <= 0 || last >= 100 {
		t.Fatalf("RSI=%v out of expected open range (0,100)", last)
	}
}

// TestCalculateRSI_InsufficientData asserts the empty-slice contract is held
// when caller supplies fewer than `period` prices.
func TestCalculateRSI_InsufficientData(t *testing.T) {
	if rsi := CalculateRSI([]float64{1, 2, 3}, 14); len(rsi) != 0 {
		t.Fatalf("expected empty slice, got %v", rsi)
	}
}

// TestCalculateMACD_InsufficientData asserts MACD does not panic and returns
// empty slices when prices are shorter than slow period — previously this
// path indexed `slowEMA[i]` against an empty slice and crashed.
func TestCalculateMACD_InsufficientData(t *testing.T) {
	prices := []float64{1, 2, 3, 4, 5}
	macd, signal := CalculateMACD(prices, 12, 26, 9)
	if len(macd) != 0 || len(signal) != 0 {
		t.Fatalf("expected empty slices for short input, got macd=%d signal=%d", len(macd), len(signal))
	}
}

// TestCalculateMACD_BadParams covers non-positive periods.
func TestCalculateMACD_BadParams(t *testing.T) {
	prices := make([]float64, 100)
	for i := range prices {
		prices[i] = float64(i)
	}
	if m, s := CalculateMACD(prices, 0, 26, 9); len(m) != 0 || len(s) != 0 {
		t.Fatalf("expected empty result for zero fast period")
	}
}

// TestCalculateMACD_Sufficient verifies the happy path produces aligned
// MACD/signal lines.
func TestCalculateMACD_Sufficient(t *testing.T) {
	prices := make([]float64, 100)
	for i := range prices {
		prices[i] = math.Sin(float64(i)/5.0)*10 + 100
	}
	macd, signal := CalculateMACD(prices, 12, 26, 9)
	if len(macd) != len(prices) {
		t.Fatalf("macd length = %d, want %d", len(macd), len(prices))
	}
	if len(signal) != len(macd) {
		t.Fatalf("signal length = %d, want %d", len(signal), len(macd))
	}
}

// TestCalculateATR_Basic exercises the volatility indicator on a known small
// fixture so that future regressions in true-range math are caught early.
func TestCalculateATR_Basic(t *testing.T) {
	highs := []float64{10, 11, 12, 13, 14, 15}
	lows := []float64{8, 9, 10, 11, 12, 13}
	closes := []float64{9, 10, 11, 12, 13, 14}
	atr := CalculateATR(highs, lows, closes, 3)
	if len(atr) == 0 {
		t.Fatal("expected non-empty ATR")
	}
	if atr[0] <= 0 {
		t.Fatalf("ATR must be positive on a non-flat series, got %v", atr[0])
	}
}

// TestCalculateADX_InsufficientData ensures ADX returns empty rather than
// panicking on very short OHLC series.
func TestCalculateADX_InsufficientData(t *testing.T) {
	highs := []float64{1, 2, 3}
	lows := []float64{0.5, 1.5, 2.5}
	closes := []float64{0.8, 1.8, 2.8}
	if adx := CalculateADX(highs, lows, closes, 14); len(adx) != 0 {
		t.Fatalf("expected empty ADX for short input, got len=%d", len(adx))
	}
}

// TestCalculateEMA_LengthMatches asserts EMA preserves input length so that
// MACD's element-wise subtraction is well defined.
func TestCalculateEMA_LengthMatches(t *testing.T) {
	prices := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	ema, err := CalculateEMA(prices, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ema) != len(prices) {
		t.Fatalf("EMA length = %d, want %d", len(ema), len(prices))
	}
	if ema[len(ema)-1] <= 0 {
		t.Fatalf("EMA tail must be positive on positive input")
	}
}
