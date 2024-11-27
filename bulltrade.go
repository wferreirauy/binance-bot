package main

import (
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	binance_connector "github.com/binance/binance-connector-go"
	color "github.com/fatih/color"
	"github.com/gosuri/uilive"
)

var period = 100            // length period for moving average
var interval = "1m"         // time intervals of historical prices for trading
var intervalTendency = "5m" // time intervals for get tendency

func BullTrade(symbol string, qty, stopLoss, takeProfit, buyFactor, sellFactor float64,
	roundPrice, roundAmount, max_ops uint) {

	// initialize binance api client
	client := binance_connector.NewClient(apikey, secretkey, baseurl)

	// define text colors
	cyan := color.New(color.FgHiCyan, color.Bold).SprintFunc()
	red := color.New(color.FgHiRed, color.Bold).SprintFunc()
	green := color.New(color.FgHiGreen, color.Bold).SprintFunc()
	white := color.New(color.FgHiWhite, color.Bold).SprintFunc()

	// parse ticker format symbol
	if re := regexp.MustCompile(`(?m)^[0-9A-Z]{2,8}/[0-9A-Z]{2,8}$`); !re.Match([]byte(symbol)) {
		log.Fatal("error parsing ticker: must match ^[0-9A-Z]{2,8}/[0-9A-Z]{2,8}$")
	}
	scoin, dcoin, found := strings.Cut(symbol, "/")
	if !found {
		log.Fatal("error parsing ticker: \"/\" is missing ")
	}
	ticker := strings.Replace(symbol, "/", "", -1)

	var buyPrice float64
	var operation = 0

	for range max_ops {
		// set tui writers
		cpw := uilive.New() // current price line writer
		cpw.Start()
		odw := uilive.New() // show order details in other line writer
		odw.Start()
		osw := uilive.New() // show order latest status in another line writer
		osw.Start()

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
			tendency, err := getTendency(client, ticker, intervalTendency, period)
			if err != nil {
				log.Printf("Error getting tendency: %v\n", err)
				time.Sleep(10 * time.Second)
				continue
			}
			//rsi
			rsi := calculateRSI(hp, 14)
			// macd
			macdLine, signalLine := calculateMACD(hp, 12, 26, 9)

			// when to buy
			if rsi < 70 && // RSI below 70
				macdLine[len(macdLine)-2] <= signalLine[len(signalLine)-2] &&
				macdLine[len(macdLine)-1] > signalLine[len(signalLine)-1] && // MACD crosses signal
				tendency == "up" { // 15m tendency

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

				fmt.Fprintf(odw, "%s %s %f %s - PRICE: %s - Total %s: %f\n",
					time.Now().Format("02/01/2006 15:04:05"), green("BUY"), qty, scoin, white(buyPrice), dcoin, buyPrice*qty)

				if getor, err := GetOrder(ticker, orderId); err == nil {
					fmt.Fprintf(osw, "%s BUY order created. Id: %d - Status: %s\n",
						time.Now().Format("02/01/2006 15:04:05"), getor.OrderId, getor.Status)
				}

				for { // looking at buy order until is filled
					if getor, err := GetOrder(ticker, orderId); err == nil {
						if getor.Status == "FILLED" {
							osw.Newline()
							fmt.Fprintf(osw, "%s BUY order filled!\n", time.Now().Format("02/01/2006 15:04:05"))
							break // buy filled
						}
					}
					time.Sleep(10 * time.Second) // 10 secs to take another look
				}
				break // indicators conditions meet
			}

			time.Sleep(10 * time.Second)
		}

		time.Sleep(30 * time.Second) // sleep before start selling process

		//// sell ////
		for {
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

			// stop loss
			stopLossPercentage := stopLoss
			stopLossPrice := buyPrice * (1 - stopLossPercentage/100)
			if price <= stopLossPrice { // price reach stop-loss percentage
				sell, err := TradeSell(symbol, roundFloat(qty*0.998, roundAmount), price, 1.0, roundPrice)
				if err != nil {
					log.Fatalf("error creating Stop-Loss SELL order: %s\n", err)
				}
				sellOrder := reflect.ValueOf(sell).Elem()
				orderId := sellOrder.FieldByName("OrderId").Int()

				fmt.Fprintf(odw, "%s %s %f %s - PRICE: %s - Total %s: %f\n",
					time.Now().Format("02/01/2006 15:04:05"), red("SELL"), qty, scoin, white(buyPrice), dcoin, buyPrice*qty)

				if getor, err := GetOrder(ticker, orderId); err == nil {
					fmt.Fprintf(osw, "%s %s order created. Id: %d - Status: %s\n",
						time.Now().Format("02/01/2006 15:04:05"), red("STOP-LOSS SELL"), getor.OrderId, getor.Status)
				}
				for { // looking at sell order until is filled
					if getor, err := GetOrder(ticker, orderId); err == nil {
						if getor.Status == "FILLED" {
							osw.Newline()
							fmt.Fprintf(osw, "%s Stop-Loss SELL order filled!\n", time.Now().Format("02/01/2006 15:04:05"))
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
			macdLine, signalLine := calculateMACD(hp, 12, 26, 9)
			if price >= profitPrice && // price reach take profit percentage
				macdLine[len(macdLine)-2] >= signalLine[len(signalLine)-2] &&
				macdLine[len(macdLine)-1] < signalLine[len(signalLine)-1] { // MACD crosess signal
				sell, err := TradeSell(symbol, roundFloat(qty*0.998, roundAmount), price, sellFactor, roundPrice)
				if err != nil {
					log.Fatalf("error creating SELL order: %s\n", err)
				}
				sellOrder := reflect.ValueOf(sell).Elem()
				orderId := sellOrder.FieldByName("OrderId").Int()

				fmt.Fprintf(odw, "%s %s %f %s - PRICE: %s - Total %s: %f\n",
					time.Now().Format("02/01/2006 15:04:05"), red("SELL"), qty, scoin, white(buyPrice), dcoin, buyPrice*qty)

				if getor, err := GetOrder(ticker, orderId); err == nil {
					fmt.Fprintf(osw, "%s SELL order created. Id: %d - Status: %s\n",
						time.Now().Format("02/01/2006 15:04:05"), getor.OrderId, getor.Status)
				}

				for { // looking at sell order until is filled
					if getor, err := GetOrder(ticker, orderId); err == nil {
						if getor.Status == "FILLED" {
							osw.Newline()
							fmt.Fprintf(osw, "%s SELL order filled!\n\n", time.Now().Format("02/01/2006 15:04:05"))
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
		odw.Stop()
		operation++
		time.Sleep(1 * time.Minute) // 1 minute to start next operation
	}
}
