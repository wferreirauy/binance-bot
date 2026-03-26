package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	binance_connector "github.com/binance/binance-connector-go"
	color "github.com/fatih/color"
)

// Get price of a ticker symbol
func GetPrice(client *binance_connector.Client, symbol string) (float64, error) {
	p, err := client.NewTickerPriceService().
		Symbol(symbol).Do(context.Background())
	if err != nil {
		return 0.0, fmt.Errorf("price: could not get price: %w", err)
	}
	price, err := strconv.ParseFloat(p.Price, 64)
	if err != nil {
		return 0.0, fmt.Errorf("price: could not convert price to float: %w", err)
	}

	if price > 0 {
		return price, nil
	}
	return 0.0, fmt.Errorf("price: could not get price: %w", err)
}

// Print current ticker price
func printPrice(writer io.Writer, ticker string, price, prevPrice float64, round uint) {
	red := color.New(color.FgHiRed, color.Bold).SprintFunc()
	green := color.New(color.FgHiGreen, color.Bold).SprintFunc()
	yellow := color.New(color.FgHiYellow, color.Bold).SprintFunc()
	white := color.New(color.FgHiWhite, color.Bold).SprintFunc()

	scoin, dcoin, found := strings.Cut(ticker, "/")
	if !found {
		log.Fatal("ticker malformed, \"/\" is missing ")
	}

	now := time.Now().Format("02/01/2006 15:04:05")
	switch {
	case price < prevPrice:
		fmt.Fprintf(writer, "%s %s PRICE is %s %s\n",
			now, yellow(scoin), red(strconv.FormatFloat(price, 'f', int(round), 64)), dcoin)
	case price > prevPrice:
		fmt.Fprintf(writer, "%s %s PRICE is %s %s\n",
			now, yellow(scoin), green(strconv.FormatFloat(price, 'f', int(round), 64)), dcoin)
	default:
		fmt.Fprintf(writer, "%s %s PRICE is %s %s\n",
			now, yellow(scoin), white(strconv.FormatFloat(price, 'f', int(round), 64)), dcoin)
	}
}

// Get Historical Prices for a period
func getHistoricalPrices(client *binance_connector.Client, symbol, interval string, period int) ([]float64, error) {
	klines, err := client.NewKlinesService().Symbol(symbol).
		Interval(interval).
		Limit(period).
		Do(context.Background())
	if err != nil {
		return nil, err
	}

	var prices []float64
	for _, k := range klines {
		price, err := strconv.ParseFloat(k.Close, 64)
		if err != nil {
			return nil, err
		}
		prices = append(prices, price)
	}
	return prices, nil
}

// Tendency
func getTendency(client *binance_connector.Client, ticker, timePeriod string, period int) (string, error) {
	hp, err := getHistoricalPrices(client, ticker, timePeriod, period)
	if err != nil {
		return "", err
	}
	var tendency string
	dema := calculateDEMA(hp, 9)
	if ema, _ := calculateEMA(hp, period); len(ema) > 0 {
		if dema[len(dema)-1] > ema[len(ema)-1] {
			tendency = "up"
		} else if dema[len(dema)-1] < ema[len(ema)-1] {
			tendency = "down"
		}
	} else {
		tendency = "up"
	}
	return tendency, nil
}

// RSI
func calculateRSI(prices []float64, period int) []float64 {
	// Validation to avoid errors if there are not enough prices
	if len(prices) < period {
		return []float64{}
	}

	// Initialize the list of RSI values
	rsiValues := make([]float64, 0, len(prices)-period+1)

	// Initial calculation of gains and losses
	var gains, losses float64
	for i := 1; i <= period; i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}

	// Initial RSI calculation
	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)
	initialRS := avgGain / avgLoss
	rsiValues = append(rsiValues, 100-(100/(1+initialRS)))

	// Calculate RSI for the remaining prices
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

// SMA
func calculateSMA(prices []float64, period int) []float64 {
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

// EMA
func calculateEMA(prices []float64, period int) ([]float64, error) {
	if len(prices) < period {
		return []float64{}, fmt.Errorf("number of prices is less than the defined period")
	}
	multiplier := 2.0 / float64(period+1) // 0.0952381
	ema := make([]float64, len(prices))
	ema[0] = prices[0]

	for i := 1; i < len(prices); i++ {
		ema[i] = ((prices[i] - ema[i-1]) * multiplier) + ema[i-1]
	}

	return ema, nil
}

// DEMA
func calculateDEMA(prices []float64, period int) []float64 {
	if len(prices) < period {
		return []float64{}
	}

	ema1, _ := calculateEMA(prices, period)
	ema2, _ := calculateEMA(ema1, period)

	dema := make([]float64, len(prices))
	for i := range prices {
		dema[i] = 2*ema1[i] - ema2[i]
	}

	return dema
}

// MACD
func calculateMACD(prices []float64, fastPeriod, slowPeriod, signalPeriod int) ([]float64, []float64) {
	fastEMA, _ := calculateEMA(prices, fastPeriod)
	slowEMA, _ := calculateEMA(prices, slowPeriod)

	macdLine := make([]float64, len(prices))
	for i := 0; i < len(prices); i++ {
		macdLine[i] = fastEMA[i] - slowEMA[i]
	}

	signalLine, _ := calculateEMA(macdLine, signalPeriod)

	return macdLine, signalLine
}

// Bollinger Bands

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
		// Slice the period
		window := prices[i : i+period]

		// Calculate SMA
		sma := calculateSMA(window, period)

		// Calculate Standard Deviation
		stdDev := standardDeviation(window, sma[len(sma)-1])

		// Calculate Bands
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

// OHLCV stores full candlestick data
type OHLCV struct {
	Opens   []float64
	Highs   []float64
	Lows    []float64
	Closes  []float64
	Volumes []float64
}

// getHistoricalOHLCV retrieves full OHLCV candlestick data for a period
func getHistoricalOHLCV(client *binance_connector.Client, symbol, interval string, period int) (*OHLCV, error) {
	klines, err := client.NewKlinesService().Symbol(symbol).
		Interval(interval).
		Limit(period).
		Do(context.Background())
	if err != nil {
		return nil, err
	}

	ohlcv := &OHLCV{
		Opens:   make([]float64, 0, len(klines)),
		Highs:   make([]float64, 0, len(klines)),
		Lows:    make([]float64, 0, len(klines)),
		Closes:  make([]float64, 0, len(klines)),
		Volumes: make([]float64, 0, len(klines)),
	}

	for _, k := range klines {
		o, err := strconv.ParseFloat(k.Open, 64)
		if err != nil {
			return nil, fmt.Errorf("price: could not convert open to float: %w", err)
		}
		h, err := strconv.ParseFloat(k.High, 64)
		if err != nil {
			return nil, fmt.Errorf("price: could not convert high to float: %w", err)
		}
		l, err := strconv.ParseFloat(k.Low, 64)
		if err != nil {
			return nil, fmt.Errorf("price: could not convert low to float: %w", err)
		}
		c, err := strconv.ParseFloat(k.Close, 64)
		if err != nil {
			return nil, fmt.Errorf("price: could not convert close to float: %w", err)
		}
		v, err := strconv.ParseFloat(k.Volume, 64)
		if err != nil {
			return nil, fmt.Errorf("price: could not convert volume to float: %w", err)
		}
		ohlcv.Opens = append(ohlcv.Opens, o)
		ohlcv.Highs = append(ohlcv.Highs, h)
		ohlcv.Lows = append(ohlcv.Lows, l)
		ohlcv.Closes = append(ohlcv.Closes, c)
		ohlcv.Volumes = append(ohlcv.Volumes, v)
	}
	return ohlcv, nil
}

// calculateATR computes Average True Range for volatility measurement
func calculateATR(highs, lows, closes []float64, period int) []float64 {
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

// calculateADX computes Average Directional Index for trend strength
func calculateADX(highs, lows, closes []float64, period int) []float64 {
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

// calculateVWAP computes Volume Weighted Average Price
func calculateVWAP(highs, lows, closes, volumes []float64) []float64 {
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
