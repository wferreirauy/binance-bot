package main

import (
	"context"
	"fmt"
	"strconv"

	binance_connector "github.com/binance/binance-connector-go"
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

// Get Historical Prices for a period
func getHistoricalPrices(client *binance_connector.Client, symbol string, period int) ([]float64, error) {
	klines, err := client.NewKlinesService().Symbol(symbol).
		Interval("1m").
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
func calculateEMA(prices []float64, period int) []float64 {
	multiplier := 2.0 / float64(period+1) // 0.0952381
	ema := make([]float64, len(prices))
	ema[0] = prices[0]

	for i := 1; i < len(prices); i++ {
		ema[i] = ((prices[i] - ema[i-1]) * multiplier) + ema[i-1]
	}
	return ema
}

// DEMA
func calculateDEMA(prices []float64, period int) []float64 {
	if len(prices) < period {
		return []float64{}
	}

	ema1 := calculateEMA(prices, period)
	ema2 := calculateEMA(ema1, period)

	dema := make([]float64, len(prices))
	for i := range prices {
		dema[i] = 2*ema1[i] - ema2[i]
	}

	return dema
}

// MACD
func calculateMACD(prices []float64, fastPeriod, slowPeriod, signalPeriod int) (float64, float64, []float64, []float64) {
	fastEMA := calculateEMA(prices, fastPeriod)
	slowEMA := calculateEMA(prices, slowPeriod)

	macdLine := make([]float64, len(prices))
	for i := 0; i < len(prices); i++ {
		macdLine[i] = fastEMA[i] - slowEMA[i]
	}

	signalLine := calculateEMA(macdLine, signalPeriod)

	return macdLine[len(macdLine)-1], signalLine[len(signalLine)-1], macdLine, signalLine
}
