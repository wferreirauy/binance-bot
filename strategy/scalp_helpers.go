package strategy

import (
	"fmt"

	"github.com/wferreirauy/binance-bot/config"
	"github.com/wferreirauy/binance-bot/indicator"
)

// scalpEvalInput aggregates everything ComputeScalpScore needs to evaluate a
// scalp entry. All slices are expected to be aligned to the same bar series
// where possible. The function is tolerant of short slices and indicators
// that were disabled in config (zero values).
type scalpEvalInput struct {
	IsBull        bool
	Cfg           *config.Config
	Closes        []float64
	RSI           []float64
	MACDLine      []float64
	SignalLine    []float64
	BB            indicator.BollingerBands
	Tendency      string
	ADXStrong     bool
	ADXVal        float64
	CurrentVolume float64
	AvgVolume     float64
	ATRVal        float64
	Price         float64
}

type scalpEvalResult struct {
	Score          int
	MaxScore       int
	MinScore       int
	Conditions     []entryCondition
	RegimeBlocked  bool
	RegimeReason   string
	ExtremeBlocked bool
	ExtremeReason  string
}

// weight returns the weight to assign to a signal name when weighted scoring
// is enabled. Unknown names default to 1.
func scalpWeight(cfg *config.Config, name string) int {
	if cfg == nil || !cfg.ScalpMode.WeightedScoring {
		return 1
	}
	switch name {
	case "macd", "rsi", "bb":
		return 2
	case "divergence":
		return 3
	default:
		return 1
	}
}

// addCondition appends an entryCondition and updates score/total according to
// the configured weighting.
func addCondition(res *scalpEvalResult, cfg *config.Config, name, label string, met bool) {
	w := scalpWeight(cfg, name)
	res.MaxScore += w
	if met {
		res.Score += w
	}
	res.Conditions = append(res.Conditions, entryCondition{Name: label, Met: met})
}

// evaluateScalp computes the full scalp score, condition breakdown and
// regime/recent-extreme blocking signals.
func evaluateScalp(in scalpEvalInput) scalpEvalResult {
	res := scalpEvalResult{}
	cfg := in.Cfg
	if cfg == nil || len(in.RSI) == 0 || len(in.MACDLine) < 2 || len(in.SignalLine) < 2 {
		return res
	}
	lastRSI := in.RSI[len(in.RSI)-1]
	prevRSI := lastRSI
	if len(in.RSI) >= 2 {
		prevRSI = in.RSI[len(in.RSI)-2]
	}
	prev2RSI := prevRSI
	if len(in.RSI) >= 3 {
		prev2RSI = in.RSI[len(in.RSI)-3]
	}

	// ---- Regime filter (ATR%) ----
	if in.ATRVal > 0 && in.Price > 0 {
		atrPct := in.ATRVal / in.Price * 100
		if cfg.ScalpMode.MinATRPct > 0 && atrPct < cfg.ScalpMode.MinATRPct {
			res.RegimeBlocked = true
			res.RegimeReason = fmt.Sprintf("ATR %.3f%% < min %.3f%% (regime too quiet)", atrPct, cfg.ScalpMode.MinATRPct)
		}
		if cfg.ScalpMode.MaxATRPct > 0 && atrPct > cfg.ScalpMode.MaxATRPct {
			res.RegimeBlocked = true
			res.RegimeReason = fmt.Sprintf("ATR %.3f%% > max %.3f%% (regime too volatile)", atrPct, cfg.ScalpMode.MaxATRPct)
		}
	}

	// ---- Recent-extreme guard ----
	if cfg.ScalpMode.RecentExtremeBars > 0 && len(in.Closes) > cfg.ScalpMode.RecentExtremeBars+1 {
		high, low := indicator.RecentExtremeLookback(in.Closes, cfg.ScalpMode.RecentExtremeBars)
		if in.IsBull && in.Price < low {
			res.ExtremeBlocked = true
			res.ExtremeReason = fmt.Sprintf("price %.6f below recent %d-bar low %.6f", in.Price, cfg.ScalpMode.RecentExtremeBars, low)
		}
		if !in.IsBull && in.Price > high {
			res.ExtremeBlocked = true
			res.ExtremeReason = fmt.Sprintf("price %.6f above recent %d-bar high %.6f", in.Price, cfg.ScalpMode.RecentExtremeBars, high)
		}
	}

	// ---- 1. RSI (pullback-in-trend default) ----
	middle := float64(cfg.Indicators.Rsi.MiddleLimit)
	if middle == 0 {
		middle = 50
	}
	var rsiMet bool
	var rsiLabel string
	if in.IsBull {
		rsiMet = lastRSI < middle && lastRSI > prevRSI && prevRSI > prev2RSI
		rsiLabel = fmt.Sprintf("RSI pullback rising %.1f<%.0f & rising 2 bars (%.1f,%.1f,%.1f)", lastRSI, middle, prev2RSI, prevRSI, lastRSI)
	} else {
		rsiMet = lastRSI > middle && lastRSI < prevRSI && prevRSI < prev2RSI
		rsiLabel = fmt.Sprintf("RSI pullback falling %.1f>%.0f & falling 2 bars (%.1f,%.1f,%.1f)", lastRSI, middle, prev2RSI, prevRSI, lastRSI)
	}
	addCondition(&res, cfg, "rsi", rsiLabel, rsiMet)

	// ---- 2. MACD histogram (anticipatory + optional N consecutive bars) ----
	consec := cfg.Indicators.Macd.ConsecutiveBars
	if consec <= 0 {
		consec = 1
	}
	macdMet := histDirectionHolds(in.MACDLine, in.SignalLine, consec, in.IsBull)
	hist := in.MACDLine[len(in.MACDLine)-1] - in.SignalLine[len(in.SignalLine)-1]
	prevHist := in.MACDLine[len(in.MACDLine)-2] - in.SignalLine[len(in.SignalLine)-2]
	dirSym := ">"
	if !in.IsBull {
		dirSym = "<"
	}
	macdLabel := fmt.Sprintf("MACD histogram %s prev for %d bar(s) (hist=%.6f, prev=%.6f)", dirSym, consec, hist, prevHist)

	// Optional: require meaningful prior MACD/signal separation in the lookback
	// (bull: hist must have reached <= -min-separation; bear: >= +min-separation).
	// Catches "the gap was real, now it's closing in" — filters out flat noise.
	if cfg.Indicators.Macd.MinSeparation > 0 {
		lookback := cfg.Indicators.Macd.MinSepLookback
		if lookback <= 0 {
			lookback = 20
		}
		hadSep, peakSep := macdHadMinSeparation(in.MACDLine, in.SignalLine, lookback, cfg.Indicators.Macd.MinSeparation, in.IsBull)
		macdMet = macdMet && hadSep
		macdLabel = fmt.Sprintf("%s & |peak-sep| %.6f ≥ %.6f over %d bars", macdLabel, peakSep, cfg.Indicators.Macd.MinSeparation, lookback)
	}
	addCondition(&res, cfg, "macd", macdLabel, macdMet)

	// ---- 3. Tendency / fast trend gate ----
	if cfg.ScalpMode.FastTrendGate {
		macdNow := in.MACDLine[len(in.MACDLine)-1]
		fastOk := (in.IsBull && macdNow > 0) || (!in.IsBull && macdNow < 0)
		fastLabel := fmt.Sprintf("MACD zero-line trend (MACD=%.6f %s 0)", macdNow, dirSym)
		addCondition(&res, cfg, "tendency", fastLabel, fastOk)
	} else {
		expected := "up"
		if !in.IsBull {
			expected = "down"
		}
		tendOk := in.Tendency == expected
		addCondition(&res, cfg, "tendency", fmt.Sprintf("Tendency %s = %s", in.Tendency, expected), tendOk)
	}

	// ---- 4. Bollinger price-touch ----
	if len(in.BB.LowerBand) > 0 && len(in.BB.UpperBand) > 0 {
		lower := in.BB.LowerBand[len(in.BB.LowerBand)-1]
		upper := in.BB.UpperBand[len(in.BB.UpperBand)-1]
		var bbMet bool
		var bbLabel string
		if in.IsBull {
			bbMet = in.Price <= lower
			bbLabel = fmt.Sprintf("Price touched lower BB (price=%.6f <= lower=%.6f)", in.Price, lower)
		} else {
			bbMet = in.Price >= upper
			bbLabel = fmt.Sprintf("Price touched upper BB (price=%.6f >= upper=%.6f)", in.Price, upper)
		}
		addCondition(&res, cfg, "bb", bbLabel, bbMet)
	}

	// ---- 5. ADX ----
	addCondition(&res, cfg, "adx",
		fmt.Sprintf("ADX strong (%.1f > %d)", in.ADXVal, cfg.Indicators.Adx.Threshold),
		in.ADXStrong)

	// ---- 6. Volume (with strong multiplier) ----
	volMult := cfg.ScalpMode.VolumeStrongMultiplier
	if volMult <= 0 {
		volMult = 1.0
	}
	volMet := in.AvgVolume == 0 || in.CurrentVolume > in.AvgVolume*volMult
	addCondition(&res, cfg, "volume",
		fmt.Sprintf("Volume strong (%.0f > %.1fx avg %.0f)", in.CurrentVolume, volMult, in.AvgVolume),
		volMet)

	// ---- 7. BB squeeze bonus (opt-in) ----
	if cfg.ScalpMode.BBSqueezeEnabled {
		ratio := cfg.ScalpMode.BBSqueezeRatio
		if ratio <= 0 {
			ratio = 0.6
		}
		window := cfg.ScalpMode.BBSqueezeWindow
		if window <= 0 {
			window = 20
		}
		actual := indicator.BollingerWidthRatio(in.BB, window)
		squeezed := actual > 0 && actual < ratio
		addCondition(&res, cfg, "squeeze",
			fmt.Sprintf("BB squeeze (width/avg %.2f < %.2f over %d bars)", actual, ratio, window),
			squeezed)
	}

	// ---- 8. Divergence bonus (opt-in, weight=3 in weighted mode) ----
	if cfg.ScalpMode.DivergenceEnabled && len(in.RSI) > 0 && len(in.Closes) > 0 {
		lookback := cfg.ScalpMode.DivergenceLookback
		if lookback <= 0 {
			lookback = 30
		}
		pad := cfg.ScalpMode.DivergenceSwingPad
		if pad <= 0 {
			pad = 2
		}
		// Align closes to RSI tail length (RSI is shorter by ~period).
		rsiLen := len(in.RSI)
		clo := in.Closes
		if len(clo) > rsiLen {
			clo = clo[len(clo)-rsiLen:]
		}
		var divMet bool
		if in.IsBull {
			divMet = indicator.BullishDivergence(clo, in.RSI, lookback, pad)
		} else {
			divMet = indicator.BearishDivergence(clo, in.RSI, lookback, pad)
		}
		addCondition(&res, cfg, "divergence",
			fmt.Sprintf("%s divergence over %d bars", map[bool]string{true: "Bullish", false: "Bearish"}[in.IsBull], lookback),
			divMet)
	}

	// ---- min-score ----
	res.MinScore = cfg.ScalpMode.MinScore
	if res.MinScore <= 0 {
		res.MinScore = 3
	}

	return res
}

// histDirectionHolds returns true when the MACD histogram has been moving in
// the trade direction for the last `n` bars (n>=1). For bull this means each
// bar's histogram is strictly greater than the previous bar; for bear, less.
func histDirectionHolds(macd, signal []float64, n int, isBull bool) bool {
	if n < 1 || len(macd) < n+1 || len(signal) < n+1 {
		return false
	}
	for k := 0; k < n; k++ {
		idx := len(macd) - 1 - k
		hist := macd[idx] - signal[idx]
		prev := macd[idx-1] - signal[idx-1]
		if isBull && !(hist > prev) {
			return false
		}
		if !isBull && !(hist < prev) {
			return false
		}
	}
	return true
}

// macdHadMinSeparation returns (true, peakSignedSep) when, within the last
// `lookback` bars of the histogram (macd-signal), there was at least one bar
// whose histogram reached the configured prior-divergence threshold in the
// direction opposite to the current trade — i.e. for a BULL trade the
// histogram must have been ≤ -minSep at some point (MACD meaningfully below
// signal) before now closing back in; for BEAR the histogram must have been
// ≥ +minSep. peakSignedSep is the most extreme value found in that window
// (negative for bull-side check, positive for bear-side) — useful for logging.
func macdHadMinSeparation(macd, signal []float64, lookback int, minSep float64, isBull bool) (bool, float64) {
	if lookback < 1 || minSep <= 0 || len(macd) < 2 || len(signal) < 2 {
		return false, 0
	}
	start := len(macd) - lookback
	if start < 0 {
		start = 0
	}
	if start > len(signal)-1 {
		return false, 0
	}
	if start > len(macd)-1 {
		return false, 0
	}
	var peak float64
	first := true
	for i := start; i < len(macd) && i < len(signal); i++ {
		h := macd[i] - signal[i]
		if first {
			peak = h
			first = false
			continue
		}
		if isBull {
			if h < peak { // most negative
				peak = h
			}
		} else {
			if h > peak { // most positive
				peak = h
			}
		}
	}
	if isBull {
		return peak <= -minSep, peak
	}
	return peak >= minSep, peak
}

// effectiveTPAndSL computes take-profit / stop-loss percentages, applying
// ATR-based overrides when configured. The caller passes the user's --tp/--sl
// arguments; this function returns the values to actually use.
//
// Precedence:
//
//	TPATRMultiplier > 0 → TP = ATR% × multiplier (overrides --tp)
//	SLATRMultiplier > 0 → SL = ATR% × multiplier (overrides --sl)
//	ATRStopLoss flag    → SL = max(SL, ATR% × ATRMultiplier)  (existing floor behaviour)
func effectiveTPAndSL(cfg *config.Config, tp, sl, atr, price float64) (effTP, effSL float64) {
	effTP, effSL = tp, sl
	if cfg == nil || atr <= 0 || price <= 0 {
		return
	}
	atrPct := atr / price * 100
	if cfg.ScalpMode.TPATRMultiplier > 0 {
		effTP = atrPct * cfg.ScalpMode.TPATRMultiplier
	}
	if cfg.ScalpMode.SLATRMultiplier > 0 {
		effSL = atrPct * cfg.ScalpMode.SLATRMultiplier
	} else if cfg.ScalpMode.ATRStopLoss {
		m := cfg.ScalpMode.ATRMultiplier
		if m <= 0 {
			m = 1.5
		}
		floor := atrPct * m
		if floor > effSL {
			effSL = floor
		}
	}
	return
}

// shouldMACDPeakExit returns true when the bot is in profit AND the MACD
// histogram has rolled over against the trade direction (peak/trough printed
// on the previous bar). This anticipates the reversal before TP/SL hit.
func shouldMACDPeakExit(cfg *config.Config, macd, signal []float64, isBull bool, pnlPct float64) bool {
	if cfg == nil || !cfg.ScalpMode.MACDPeakExit || pnlPct <= 0 || len(macd) < 3 || len(signal) < 3 {
		return false
	}
	h0 := macd[len(macd)-1] - signal[len(signal)-1]
	h1 := macd[len(macd)-2] - signal[len(signal)-2]
	h2 := macd[len(macd)-3] - signal[len(signal)-3]
	if isBull {
		// Was rising (h2<h1), now falling (h1>h0) → peak
		return h2 < h1 && h0 < h1
	}
	// Was falling (h2>h1), now rising (h1<h0) → trough
	return h2 > h1 && h0 > h1
}
