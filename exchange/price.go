package exchange

import (
	"context"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	binance_connector "github.com/binance/binance-connector-go"
	color "github.com/fatih/color"
	"github.com/wferreirauy/binance-bot/indicator"
)

// GetPrice gets the price of a ticker symbol
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

// PrintPrice prints the current ticker price with color formatting
func PrintPrice(writer io.Writer, ticker string, price, prevPrice float64, round uint) {
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

// GetHistoricalPrices retrieves closing prices for a period
func GetHistoricalPrices(client *binance_connector.Client, symbol, interval string, period int) ([]float64, error) {
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

// GetTendency determines market tendency (up/down) using DEMA vs EMA
func GetTendency(client *binance_connector.Client, ticker, timePeriod string, period int) (string, error) {
	hp, err := GetHistoricalPrices(client, ticker, timePeriod, period)
	if err != nil {
		return "", err
	}
	var tendency string
	dema := indicator.CalculateDEMA(hp, 9)
	if ema, _ := indicator.CalculateEMA(hp, period); len(ema) > 0 {
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

// OHLCV stores full candlestick data
type OHLCV struct {
	Opens   []float64
	Highs   []float64
	Lows    []float64
	Closes  []float64
	Volumes []float64
}

// GetHistoricalOHLCV retrieves full OHLCV candlestick data for a period
func GetHistoricalOHLCV(client *binance_connector.Client, symbol, interval string, period int) (*OHLCV, error) {
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
