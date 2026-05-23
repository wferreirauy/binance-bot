package indicator

import "math"

// CalculateSmoothedRSI applies an EMA(smoothLength) to prices before computing
// the standard RSI. When smoothLength <= 1 it is equivalent to CalculateRSI.
// Smoothing reduces noise on fast (1m) candles without significantly delaying
// turning points.
func CalculateSmoothedRSI(prices []float64, period, smoothLength int) []float64 {
	if smoothLength <= 1 {
		return CalculateRSI(prices, period)
	}
	smoothed, err := CalculateEMA(prices, smoothLength)
	if err != nil || len(smoothed) == 0 {
		return CalculateRSI(prices, period)
	}
	return CalculateRSI(smoothed, period)
}

// CalculateStochRSI computes Stochastic-RSI: %K of RSI over a lookback window.
// Returns values in [0,1]. Useful on 1m for catching turns 1-2 bars earlier
// than raw RSI, at the cost of more noise.
func CalculateStochRSI(prices []float64, rsiPeriod, stochPeriod int) []float64 {
	rsi := CalculateRSI(prices, rsiPeriod)
	if len(rsi) < stochPeriod {
		return []float64{}
	}
	out := make([]float64, 0, len(rsi)-stochPeriod+1)
	for i := stochPeriod - 1; i < len(rsi); i++ {
		minV := rsi[i-stochPeriod+1]
		maxV := minV
		for j := i - stochPeriod + 2; j <= i; j++ {
			if rsi[j] < minV {
				minV = rsi[j]
			}
			if rsi[j] > maxV {
				maxV = rsi[j]
			}
		}
		if maxV == minV {
			out = append(out, 0.5)
			continue
		}
		out = append(out, (rsi[i]-minV)/(maxV-minV))
	}
	return out
}

// BollingerWidthRatio returns the latest band width (upper-lower) divided by
// the average band width over the last `avgWindow` bars. A return value below
// ~0.6 typically indicates a squeeze (low volatility coiling).
func BollingerWidthRatio(bb BollingerBands, avgWindow int) float64 {
	n := len(bb.UpperBand)
	if n == 0 || avgWindow <= 0 || n < avgWindow {
		return 0
	}
	current := bb.UpperBand[n-1] - bb.LowerBand[n-1]
	if current <= 0 {
		return 0
	}
	var sum float64
	for i := n - avgWindow; i < n; i++ {
		sum += bb.UpperBand[i] - bb.LowerBand[i]
	}
	avg := sum / float64(avgWindow)
	if avg <= 0 {
		return 0
	}
	return current / avg
}

// findSwingExtrema locates indices of local minima (lows=true) or maxima
// (lows=false) where the bar is strictly more extreme than `lookback`
// neighbors on each side. Returns indices in ascending order.
func findSwingExtrema(prices []float64, lookback int, lows bool) []int {
	if lookback < 1 || len(prices) < 2*lookback+1 {
		return nil
	}
	var out []int
	for i := lookback; i < len(prices)-lookback; i++ {
		extreme := true
		for j := 1; j <= lookback; j++ {
			if lows {
				if prices[i-j] <= prices[i] || prices[i+j] <= prices[i] {
					extreme = false
					break
				}
			} else {
				if prices[i-j] >= prices[i] || prices[i+j] >= prices[i] {
					extreme = false
					break
				}
			}
		}
		if extreme {
			out = append(out, i)
		}
	}
	return out
}

// BullishDivergence reports true when, within the last `lookback` bars of
// `closes`/`oscillator` (same length), price made a lower low but the
// oscillator made a higher low at the two most recent swings.
func BullishDivergence(closes, oscillator []float64, lookback, swingPad int) bool {
	if len(closes) != len(oscillator) || len(closes) < lookback {
		return false
	}
	start := len(closes) - lookback
	priceSlice := closes[start:]
	oscSlice := oscillator[start:]
	priceLows := findSwingExtrema(priceSlice, swingPad, true)
	if len(priceLows) < 2 {
		return false
	}
	oscLows := findSwingExtrema(oscSlice, swingPad, true)
	if len(oscLows) < 2 {
		return false
	}
	pA, pB := priceLows[len(priceLows)-2], priceLows[len(priceLows)-1]
	oA, oB := oscLows[len(oscLows)-2], oscLows[len(oscLows)-1]
	// require swings to be roughly aligned (within swingPad bars)
	if math.Abs(float64(pA-oA)) > float64(swingPad) || math.Abs(float64(pB-oB)) > float64(swingPad) {
		return false
	}
	return priceSlice[pB] < priceSlice[pA] && oscSlice[oB] > oscSlice[oA]
}

// BearishDivergence reports true when, within the last `lookback` bars, price
// made a higher high but the oscillator made a lower high at the two most
// recent swings.
func BearishDivergence(closes, oscillator []float64, lookback, swingPad int) bool {
	if len(closes) != len(oscillator) || len(closes) < lookback {
		return false
	}
	start := len(closes) - lookback
	priceSlice := closes[start:]
	oscSlice := oscillator[start:]
	priceHighs := findSwingExtrema(priceSlice, swingPad, false)
	if len(priceHighs) < 2 {
		return false
	}
	oscHighs := findSwingExtrema(oscSlice, swingPad, false)
	if len(oscHighs) < 2 {
		return false
	}
	pA, pB := priceHighs[len(priceHighs)-2], priceHighs[len(priceHighs)-1]
	oA, oB := oscHighs[len(oscHighs)-2], oscHighs[len(oscHighs)-1]
	if math.Abs(float64(pA-oA)) > float64(swingPad) || math.Abs(float64(pB-oB)) > float64(swingPad) {
		return false
	}
	return priceSlice[pB] > priceSlice[pA] && oscSlice[oB] < oscSlice[oA]
}

// RecentExtremeLookback returns the highest and lowest closes within the last
// `lookback` bars (excluding the most recent bar). Useful for deciding whether
// the latest bar prints a fresh extreme that should block counter-trend
// entries.
func RecentExtremeLookback(closes []float64, lookback int) (high, low float64) {
	n := len(closes)
	if lookback <= 0 || n < lookback+1 {
		return 0, 0
	}
	high = closes[n-lookback-1]
	low = high
	for i := n - lookback - 1; i < n-1; i++ {
		if closes[i] > high {
			high = closes[i]
		}
		if closes[i] < low {
			low = closes[i]
		}
	}
	return high, low
}
