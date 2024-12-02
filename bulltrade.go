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
)

func BullTrade(symbol string, qty, stopLoss, takeProfit, buyFactor, sellFactor float64,
	roundPrice, roundAmount, max_ops uint) {

	cyan := color.New(color.FgHiCyan, color.Bold).SprintFunc()
	red := color.New(color.FgHiRed, color.Bold).SprintFunc()
	green := color.New(color.FgHiGreen, color.Bold).SprintFunc()
	yellow := color.New(color.FgHiYellow, color.Bold).SprintFunc()
	white := color.New(color.FgHiWhite, color.Bold).SprintFunc()

	client := binance_connector.NewClient(apikey, secretkey, baseurl)

	period := 100 // period for moving average

	// parse symbol
	if re := regexp.MustCompile(`(?m)^[0-9A-Z]{2,8}/[0-9A-Z]{2,8}$`); !re.Match([]byte(symbol)) {
		log.Fatal("error parsing ticker: must match ^[0-9A-Z]{2,8}/[0-9A-Z]{2,8}$")
	}
	scoin, dcoin, found := strings.Cut(symbol, "/")
	if !found {
		log.Fatal("error parsing ticker: \"/\" is missing ")
	}
	ticker := strings.Replace(symbol, "/", "", -1)

	var buyPrice float64

	operation := 0
	for range max_ops {
		fmt.Println(white("Operation"), cyan("#"+strconv.Itoa(operation)))
		qty = roundFloat(qty, roundAmount)

		//// buy
		fmt.Print("\033[s") // save the cursor position

		for {
			var tendency string
			hp15m, err := getHistoricalPrices(client, ticker, "15m", period)
			if err != nil {
				log.Printf("Error getting historical prices with 15m interval: %v\n", err)
				time.Sleep(10 * time.Second)
				continue
			}
			dema15 := calculateDEMA(hp15m, 9)
			if ema15, _ := calculateEMA(hp15m, 100); len(ema15) > 0 {
				if dema15[len(dema15)-1] > ema15[len(ema15)-1] {
					tendency = "up"
				} else if dema15[len(dema15)-1] < ema15[len(ema15)-1] {
					tendency = "down"
				}
			} else {
				tendency = "up"
			}

			hp1m, err := getHistoricalPrices(client, ticker, "1m", period)
			if err != nil {
				log.Printf("Error getting historical prices with 1m interval: %v\n", err)
				time.Sleep(10 * time.Second)
				continue
			}
			price := hp1m[len(hp1m)-1]
			prevPrice := hp1m[len(hp1m)-2]
			fmt.Print("\033[u\033[K") // restore the cursor position and clear the line
			switch {
			case price < prevPrice:
				log.Printf("%s PRICE is %s %s ", yellow(scoin), red(strconv.FormatFloat(price, 'f', int(roundPrice), 64)), dcoin)
			case price > prevPrice:
				log.Printf("%s PRICE is %s %s ", yellow(scoin), green(strconv.FormatFloat(price, 'f', int(roundPrice), 64)), dcoin)
			default:
				log.Printf("%s PRICE is %s %s ", yellow(scoin), white(strconv.FormatFloat(price, 'f', int(roundPrice), 64)), dcoin)
			}
			macdLine, signalLine := calculateMACD(hp1m, 12, 26, 9)
			rsi := calculateRSI(hp1m, 14)

			// where to enter?
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

				if getor, err := GetOrder(ticker, orderId); err == nil {
					log.Printf("BUY order created. Id: %d - Status: %s\n", getor.OrderId, getor.Status)
				}

				for { // looking at buy order until filled
					if getor, err := GetOrder(ticker, orderId); err == nil {
						if getor.Status == "FILLED" {
							log.Println("BUY order filled!")
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

		// sell
		fmt.Print("\033[s") // save the cursor position
		for {
			hp1m, err := getHistoricalPrices(client, ticker, "1m", period)
			if err != nil {
				log.Printf("Error getting historical prices with 1m interval: %v\n", err)
				time.Sleep(10 * time.Second)
				continue
			}
			price := hp1m[len(hp1m)-1]
			prevPrice := hp1m[len(hp1m)-2]
			fmt.Print("\033[u\033[K") // restore the cursor position and clear the line
			switch {
			case price < prevPrice:
				log.Printf("%s PRICE is %s %s ", yellow(scoin), red(strconv.FormatFloat(price, 'f', int(roundPrice), 64)), dcoin)
			case price > prevPrice:
				log.Printf("%s PRICE is %s %s ", yellow(scoin), green(strconv.FormatFloat(price, 'f', int(roundPrice), 64)), dcoin)
			default:
				log.Printf("%s PRICE is %s %s ", yellow(scoin), white(strconv.FormatFloat(price, 'f', int(roundPrice), 64)), dcoin)
			}

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
				if getor, err := GetOrder(ticker, orderId); err == nil {
					log.Printf("%s order created. Id: %d - Status: %s\n",
						red("STOP-LOSS SELL"), getor.OrderId, getor.Status)
				}
				for { // looking at sell order until FILLED
					if getor, err := GetOrder(ticker, orderId); err == nil {
						if getor.Status == "FILLED" {
							log.Printf("Stop-Loss SELL order filled!\n\n")
							break // sell filled
						}
					}
					time.Sleep(10 * time.Second) // 10 secs to take another look
				}
				break // stop loss order sold
			}

			// take profit
			profitPercentage := takeProfit
			profitPrice := buyPrice * (1 + profitPercentage/100)
			macdLine, signalLine := calculateMACD(hp1m, 12, 26, 9)
			if price >= profitPrice && // price reach take profit percentage
				macdLine[len(macdLine)-2] >= signalLine[len(signalLine)-2] &&
				macdLine[len(macdLine)-1] < signalLine[len(signalLine)-1] { // MACD crosess signal
				sell, err := TradeSell(symbol, roundFloat(qty*0.998, roundAmount), price, sellFactor, roundPrice)
				if err != nil {
					log.Fatalf("error creating SELL order: %s\n", err)
				}
				sellOrder := reflect.ValueOf(sell).Elem()
				orderId := sellOrder.FieldByName("OrderId").Int()
				if getor, err := GetOrder(ticker, orderId); err == nil {
					log.Printf("SELL order created. Id: %d - Status: %s\n", getor.OrderId, getor.Status)
				}
				for { // looking at sell order until FILLED
					if getor, err := GetOrder(ticker, orderId); err == nil {
						if getor.Status == "FILLED" {
							log.Printf("SELL order filled!\n\n")
							break // sell filled
						}
					}
					time.Sleep(10 * time.Second) // 10 secs to take another look
				}
				break // take profit order sold
			}
			time.Sleep(10 * time.Second)
		}
		operation++
		time.Sleep(1 * time.Minute) // 1 minute to start next operation
	}
}
