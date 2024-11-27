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
	if ema, _ := calculateEMA(hp, 100); len(ema) > 0 {
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
func calculateRSI(prices []float64, period int) float64 {
	var gains, losses float64
	for i := 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}
	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
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
