package main

import (
	"fmt"
	"log"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	binance_connector "github.com/binance/binance-connector-go"
	color "github.com/fatih/color"
	"github.com/gosuri/uilive"
	"github.com/wferreirauy/binance-bot/config"
)

func BullTrade(
	configFile string,
	symbol string,
	qty float64,
	stopLoss float64,
	takeProfit float64,
	buyFactor float64,
	sellFactor float64,
	roundPrice uint,
	roundAmount uint,
	max_ops uint,
) {

	// read config.yml file
	var c config.Config
	cfg, err := c.Read(configFile)
	if err != nil {
		log.Fatal(err)
	}
	period := cfg.HistoricalPrices.Period     // length period for moving average
	interval := cfg.HistoricalPrices.Interval // time intervals of historical prices for trading

	// initialize binance api client
	client := binance_connector.NewClient(apikey, secretkey, baseurl)

	// define text colors
	cyan := color.New(color.FgHiCyan, color.Bold).SprintFunc()
	red := color.New(color.FgHiRed, color.Bold).SprintFunc()
	green := color.New(color.FgHiGreen, color.Bold).SprintFunc()
	white := color.New(color.FgHiWhite, color.Bold).SprintFunc()

	// validate symbol in format 0-9A-Z/0-9A-Z
	if re := regexp.MustCompile(`(?m)^[0-9A-Z]{2,8}/[0-9A-Z]{2,8}$`); !re.Match([]byte(symbol)) {
		log.Fatal("error parsing ticker: must match ^[0-9A-Z]{2,8}/[0-9A-Z]{2,8}$")
	}
	scoin, dcoin, found := strings.Cut(symbol, "/")
	if !found {
		log.Fatal("error parsing ticker: \"/\" is missing ")
	}
	ticker := strings.Replace(symbol, "/", "", -1)

	var buyPrice float64
	var operation = 1

	for range max_ops {
		// set tui writers
		cpw := uilive.New() // current price line writer
		cpw.Start()

		fmt.Println(white("Operation"), cyan("#"+strconv.Itoa(operation)))
		qty = roundFloat(qty, roundAmount)

		//// buy ////
		for {
			// get historical prices
			hp, err := getHistoricalPrices(client, ticker, interval, period)
			if err != nil {
				log.Printf("Error getting historical prices with %s interval: %v\n", interval, err)
				time.Sleep(10 * time.Second)
				continue
			}

			price := hp[len(hp)-1]
			prevPrice := hp[len(hp)-2]

			// print current price
			printPrice(cpw, symbol, price, prevPrice, roundPrice)

			// indicators
			// tendency "up" or "down"
			tendency, err := getTendency(client, ticker, cfg.Tendency.Interval, period)
			if err != nil {
				log.Printf("Error getting tendency: %v\n", err)
				time.Sleep(10 * time.Second)
				continue
			}
			// dema
			dema := calculateDEMA(hp, cfg.Indicators.Dema.Length)
			currentDema := dema[len(dema)-1]
			//rsi
			rsi := calculateRSI(hp, cfg.Indicators.Rsi.Length)
			// macd
			macdLine, signalLine := calculateMACD(
				hp,
				cfg.Indicators.Macd.FastLength,
				cfg.Indicators.Macd.SlowLength,
				cfg.Indicators.Macd.SignalLength,
			)
			// bollingerbands
			bb, err := CalculateBollingerBands(
				hp,
				cfg.Indicators.BollingerBands.Length,
				cfg.Indicators.BollingerBands.Multiplier,
			)
			if err != nil {
				log.Printf("Error getting BollingerBands: %v\n", err)
			}
			lowerBand := bb.LowerBand[len(bb.LowerBand)-1]
			upperBand := bb.UpperBand[len(bb.UpperBand)-1]
			distanceToUpper := math.Abs(currentDema - upperBand)
			distanceToLower := math.Abs(currentDema - lowerBand)

			// when to buy
			if rsi[len(rsi)-1] < float64(cfg.Indicators.Rsi.UpperLimit) && // RSI below upper limit
				macdLine[len(macdLine)-2] <= signalLine[len(signalLine)-2] &&
				macdLine[len(macdLine)-1] > signalLine[len(signalLine)-1] && // MACD crosses signal
				tendency == cfg.Tendency.Direction && // 3m tendency
				distanceToLower < distanceToUpper { // dema closer than lower band
				buy, err := TradeBuy(symbol, qty, price, buyFactor, roundPrice)
				if err != nil {
					log.Fatalf("error creating BUY order: %s\n", err)
				}
				buyOrder := reflect.ValueOf(buy).Elem()
				orderId := buyOrder.FieldByName("OrderId").Int()
				orderPrice := buyOrder.FieldByName("Price").String()
				buyPrice, err = strconv.ParseFloat(orderPrice, 64)
				if err != nil {
					log.Printf("could not convert price on buy order to float: %s\n", err)
				}

				fmt.Printf("%s %s %f %s - PRICE: %s - Total %s: %f\n",
					time.Now().Format("02/01/2006 15:04:05"), green("BUY"), qty, scoin, white(buyPrice), dcoin, buyPrice*qty)

				if getor, err := GetOrder(ticker, orderId); err == nil {
					fmt.Printf("%s BUY order created. Id: %d - Status: %s\n",
						time.Now().Format("02/01/2006 15:04:05"), getor.OrderId, getor.Status)
				}

				for { // looking at buy order until is filled
					if getor, err := GetOrder(ticker, orderId); err == nil {
						if getor.Status == "FILLED" {
							fmt.Printf("%s BUY order filled!\n\n", time.Now().Format("02/01/2006 15:04:05"))
							break // buy filled
						}
					}
					time.Sleep(10 * time.Second) // 10 secs to take another look
				}
				break // indicators conditions meet
			}

			time.Sleep(10 * time.Second)
		}
		cpw.Stop()
		time.Sleep(30 * time.Second) // sleep before start selling process

		//// sell ////
		cpw.Start()
		for {
			hp, err := getHistoricalPrices(client, ticker, interval, period)
			if err != nil {
				log.Printf("Error getting historical prices with %s interval: %v\n", interval, err)
				time.Sleep(10 * time.Second)
				continue
			}
			rsiprices, err := getHistoricalPrices(client, ticker, cfg.Indicators.Rsi.Interval, period)
			if err != nil {
				log.Printf("Error getting historical prices with %s interval: %v\n", interval, err)
				time.Sleep(10 * time.Second)
				continue
			}

			price := hp[len(hp)-1]
			prevPrice := hp[len(hp)-2]
			rsi := calculateRSI(rsiprices, cfg.Indicators.Rsi.Length)

			// print current price
			printPrice(cpw, symbol, price, prevPrice, roundPrice)

			// stop loss
			stopLossPercentage := stopLoss
			stopLossPrice := buyPrice * (1 - stopLossPercentage/100)
			if price <= stopLossPrice { // price reach stop-loss percentage
				sell, err := TradeSell(symbol, roundFloat(qty*0.998, roundAmount), price, 1.0, roundPrice)
				if err != nil {
					log.Fatalf("error creating Stop-Loss SELL order with amount %f: %s\n",
						roundFloat(qty*0.998, roundAmount), err)
				}
				sellOrder := reflect.ValueOf(sell).Elem()
				orderId := sellOrder.FieldByName("OrderId").Int()

				fmt.Printf("%s %s %f %s - PRICE: %s - Total %s: %f\n",
					time.Now().Format("02/01/2006 15:04:05"), red("SELL"), qty, scoin, white(price), dcoin, price*qty)

				if getor, err := GetOrder(ticker, orderId); err == nil {
					fmt.Printf("%s %s order created. Id: %d - Status: %s\n",
						time.Now().Format("02/01/2006 15:04:05"), red("STOP-LOSS SELL"), getor.OrderId, getor.Status)
				}
				for { // looking at sell order until is filled
					if getor, err := GetOrder(ticker, orderId); err == nil {
						if getor.Status == "FILLED" {
							fmt.Printf("%s %s order filled!\n\n", time.Now().Format("02/01/2006 15:04:05"), red("STOP-LOSS SELL"))
							break // sell filled
						}
					}
					time.Sleep(10 * time.Second) // 10 secs to take another look
				}
				break // sold (stop loss)
			}

			// take profit
			profitPercentage := takeProfit
			profitPrice := buyPrice * (1 + profitPercentage/100)
			if price >= profitPrice && // price reach take profit percentage
				rsi[len(rsi)-1] < rsi[len(rsi)-2] { // and rsi turns down
				sell, err := TradeSell(symbol, roundFloat(qty*0.998, roundAmount), price, sellFactor, roundPrice)
				if err != nil {
					log.Fatalf("error creating SELL order with amount %f: %s\n",
						roundFloat(qty*0.998, roundAmount), err)
				}
				sellOrder := reflect.ValueOf(sell).Elem()
				orderId := sellOrder.FieldByName("OrderId").Int()

				fmt.Printf("%s %s %f %s - PRICE: %s - Total %s: %f\n",
					time.Now().Format("02/01/2006 15:04:05"), red("SELL"), qty, scoin, white(price), dcoin, price*qty)

				if getor, err := GetOrder(ticker, orderId); err == nil {
					fmt.Printf("%s SELL order created. Id: %d - Status: %s\n",
						time.Now().Format("02/01/2006 15:04:05"), getor.OrderId, getor.Status)
				}

				for { // looking at sell order until is filled
					if getor, err := GetOrder(ticker, orderId); err == nil {
						if getor.Status == "FILLED" {
							fmt.Printf("%s SELL order filled!\n\n", time.Now().Format("02/01/2006 15:04:05"))
							break // sell filled
						}
					}
					time.Sleep(10 * time.Second) // 10 secs to take another look
				}
				break // sold
			}
			time.Sleep(10 * time.Second)
		}
		cpw.Stop()
		operation++
		time.Sleep(1 * time.Minute) // 1 minute to start next operation
	}
}
