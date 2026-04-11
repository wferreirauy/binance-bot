package indicator

import (
	"errors"
	"fmt"
	"math"
)

// RoundFloat returns a float rounded by the given precision
func RoundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

// CalculateRSI computes the Relative Strength Index
func CalculateRSI(prices []float64, period int) []float64 {
	if len(prices) < period {
		return []float64{}
	}

	rsiValues := make([]float64, 0, len(prices)-period+1)

	var gains, losses float64
	for i := 1; i <= period; i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)
	initialRS := avgGain / avgLoss
	rsiValues = append(rsiValues, 100-(100/(1+initialRS)))

	for i := period; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			avgGain = ((avgGain * float64(period-1)) + change) / float64(period)
			avgLoss = (avgLoss * float64(period-1)) / float64(period)
		} else {
			avgGain = (avgGain * float64(period-1)) / float64(period)
			avgLoss = ((avgLoss * float64(period-1)) - change) / float64(period)
		}

		rs := avgGain / avgLoss
		rsiValues = append(rsiValues, 100-(100/(1+rs)))
	}

	return rsiValues
}

// CalculateSMA computes the Simple Moving Average
func CalculateSMA(prices []float64, period int) []float64 {
	if len(prices) < period {
		return []float64{}
	}

	sma := make([]float64, len(prices)-(period-1))

	for i := 0; i <= len(prices)-period; i++ {
		sum := 0.0
		for j := 0; j < period; j++ {
			sum += prices[i+j]
		}
		sma[i] = sum / float64(period)
	}

	return sma
}

// CalculateEMA computes the Exponential Moving Average
func CalculateEMA(prices []float64, period int) ([]float64, error) {
	if len(prices) < period {
		return []float64{}, fmt.Errorf("number of prices is less than the defined period")
	}
	multiplier := 2.0 / float64(period+1)
	ema := make([]float64, len(prices))
	ema[0] = prices[0]

	for i := 1; i < len(prices); i++ {
		ema[i] = ((prices[i] - ema[i-1]) * multiplier) + ema[i-1]
	}

	return ema, nil
}

// CalculateDEMA computes the Double Exponential Moving Average
func CalculateDEMA(prices []float64, period int) []float64 {
	if len(prices) < period {
		return []float64{}
	}

	ema1, _ := CalculateEMA(prices, period)
	ema2, _ := CalculateEMA(ema1, period)

	dema := make([]float64, len(prices))
	for i := range prices {
		dema[i] = 2*ema1[i] - ema2[i]
	}

	return dema
}

// CalculateMACD computes MACD line and signal line
func CalculateMACD(prices []float64, fastPeriod, slowPeriod, signalPeriod int) ([]float64, []float64) {
	fastEMA, _ := CalculateEMA(prices, fastPeriod)
	slowEMA, _ := CalculateEMA(prices, slowPeriod)

	macdLine := make([]float64, len(prices))
	for i := 0; i < len(prices); i++ {
		macdLine[i] = fastEMA[i] - slowEMA[i]
	}

	signalLine, _ := CalculateEMA(macdLine, signalPeriod)

	return macdLine, signalLine
}

// BollingerBands stores the upper, middle (SMA), and lower bands
type BollingerBands struct {
	UpperBand  []float64
	MiddleBand []float64
	LowerBand  []float64
}

// standardDeviation calculates the Standard Deviation for a given set of prices and their mean
func standardDeviation(prices []float64, mean float64) float64 {
	var sum float64
	for _, price := range prices {
		sum += math.Pow(price-mean, 2)
	}
	variance := sum / float64(len(prices))
	return math.Sqrt(variance)
}

// CalculateBollingerBands calculates the Bollinger Bands for a given list of prices
func CalculateBollingerBands(prices []float64, period int, multiplier float64) (BollingerBands, error) {
	if len(prices) < period {
		return BollingerBands{}, errors.New("not enough prices to calculate Bollinger Bands")
	}

	var upperBand, middleBand, lowerBand []float64

	for i := 0; i <= len(prices)-period; i++ {
		window := prices[i : i+period]
		sma := CalculateSMA(window, period)
		stdDev := standardDeviation(window, sma[len(sma)-1])

		upper := sma[len(sma)-1] + multiplier*stdDev
		lower := sma[len(sma)-1] - multiplier*stdDev

		middleBand = append(middleBand, sma[len(sma)-1])
		upperBand = append(upperBand, upper)
		lowerBand = append(lowerBand, lower)
	}

	return BollingerBands{
		UpperBand:  upperBand,
		MiddleBand: middleBand,
		LowerBand:  lowerBand,
	}, nil
}

// CalculateATR computes Average True Range for volatility measurement
func CalculateATR(highs, lows, closes []float64, period int) []float64 {
	if len(highs) < period+1 {
		return []float64{}
	}

	trueRanges := make([]float64, len(highs)-1)
	for i := 1; i < len(highs); i++ {
		highLow := highs[i] - lows[i]
		highClose := math.Abs(highs[i] - closes[i-1])
		lowClose := math.Abs(lows[i] - closes[i-1])
		trueRanges[i-1] = math.Max(highLow, math.Max(highClose, lowClose))
	}

	atr := make([]float64, 0, len(trueRanges)-period+1)
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += trueRanges[i]
	}
	atr = append(atr, sum/float64(period))

	for i := period; i < len(trueRanges); i++ {
		atr = append(atr, (atr[len(atr)-1]*float64(period-1)+trueRanges[i])/float64(period))
	}

	return atr
}

// wilderSmooth applies Wilder's smoothing method
func wilderSmooth(data []float64, period int) []float64 {
	if len(data) < period {
		return []float64{}
	}

	smoothed := make([]float64, 0, len(data)-period+1)
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += data[i]
	}
	smoothed = append(smoothed, sum)

	for i := period; i < len(data); i++ {
		smoothed = append(smoothed, smoothed[len(smoothed)-1]-smoothed[len(smoothed)-1]/float64(period)+data[i])
	}

	return smoothed
}

// CalculateADX computes Average Directional Index for trend strength
func CalculateADX(highs, lows, closes []float64, period int) []float64 {
	if len(highs) < period*2+1 {
		return []float64{}
	}

	plusDM := make([]float64, len(highs)-1)
	minusDM := make([]float64, len(highs)-1)
	tr := make([]float64, len(highs)-1)

	for i := 1; i < len(highs); i++ {
		upMove := highs[i] - highs[i-1]
		downMove := lows[i-1] - lows[i]
		if upMove > downMove && upMove > 0 {
			plusDM[i-1] = upMove
		}
		if downMove > upMove && downMove > 0 {
			minusDM[i-1] = downMove
		}
		highLow := highs[i] - lows[i]
		highClose := math.Abs(highs[i] - closes[i-1])
		lowClose := math.Abs(lows[i] - closes[i-1])
		tr[i-1] = math.Max(highLow, math.Max(highClose, lowClose))
	}

	smoothTR := wilderSmooth(tr, period)
	smoothPlusDM := wilderSmooth(plusDM, period)
	smoothMinusDM := wilderSmooth(minusDM, period)

	minLen := len(smoothTR)
	if len(smoothPlusDM) < minLen {
		minLen = len(smoothPlusDM)
	}
	if len(smoothMinusDM) < minLen {
		minLen = len(smoothMinusDM)
	}

	dx := make([]float64, minLen)
	for i := 0; i < minLen; i++ {
		if smoothTR[i] == 0 {
			continue
		}
		plusDI := 100 * smoothPlusDM[i] / smoothTR[i]
		minusDI := 100 * smoothMinusDM[i] / smoothTR[i]
		diSum := plusDI + minusDI
		if diSum > 0 {
			dx[i] = 100 * math.Abs(plusDI-minusDI) / diSum
		}
	}

	if len(dx) < period {
		return []float64{}
	}

	adx := make([]float64, 0, len(dx)-period+1)
	dxSum := 0.0
	for i := 0; i < period; i++ {
		dxSum += dx[i]
	}
	adx = append(adx, dxSum/float64(period))

	for i := period; i < len(dx); i++ {
		adx = append(adx, (adx[len(adx)-1]*float64(period-1)+dx[i])/float64(period))
	}

	return adx
}

// CalculateVWAP computes Volume Weighted Average Price
func CalculateVWAP(highs, lows, closes, volumes []float64) []float64 {
	if len(closes) == 0 {
		return []float64{}
	}

	vwap := make([]float64, len(closes))
	cumulativeTPV := 0.0
	cumulativeVolume := 0.0

	for i := range closes {
		typicalPrice := (highs[i] + lows[i] + closes[i]) / 3.0
		cumulativeTPV += typicalPrice * volumes[i]
		cumulativeVolume += volumes[i]
		if cumulativeVolume > 0 {
			vwap[i] = cumulativeTPV / cumulativeVolume
		}
	}

	return vwap
}
