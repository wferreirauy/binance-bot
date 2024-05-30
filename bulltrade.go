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

func BullTrade(symbol string, qty, buyFactor, sellFactor float64, roundPrice, roundAmount, max_ops int) {
	cyan := color.New(color.FgCyan, color.Bold).SprintFunc()
	red := color.New(color.FgRed, color.Bold).SprintFunc()
	green := color.New(color.FgGreen, color.Bold).SprintFunc()

	client := binance_connector.NewClient(apikey, secretkey, baseurl)

	period := 26 // period for moving average

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
		fmt.Println("Operation", cyan("#"+strconv.Itoa(operation)))
		qty = toFixed(qty, roundAmount)

		// buy
		fmt.Print("\033[s") // save the cursor position
		for {
			historicalPrices, err := getHistoricalPrices(client, ticker, period+26)
			if err != nil {
				log.Printf("Error getting historical prices: %v\n", err)
				continue
			}
			price := historicalPrices[len(historicalPrices)-1]
			fmt.Print("\033[u\033[K") // restore the cursor position and clear the line
			log.Printf("%s PRICE is %.8f %s\n", scoin, price, dcoin)
			sma := calculateSMA(historicalPrices, period)
			ema := calculateEMA(historicalPrices, period)
			lastMacd, lastSignal, _, _ := calculateMACD(historicalPrices, 12, 26, 9)
			rsi := calculateRSI(historicalPrices, period)

			if rsi < 70 && ema[len(ema)-1] > sma[len(sma)-1] &&
				ema[len(ema)-2] <= sma[len(sma)-2] && lastMacd > lastSignal {
				log.Printf("Creating new %s order\n", green("BUY"))
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
			time.Sleep(20 * time.Second)
		}

		time.Sleep(30 * time.Second)

		// sell
		fmt.Print("\033[s") // save the cursor position
		for {
			historicalPrices, err := getHistoricalPrices(client, ticker, period+26)
			if err != nil {
				log.Printf("Error getting historical prices: %v\n", err)
				continue
			}
			price := historicalPrices[len(historicalPrices)-1]
			fmt.Print("\033[u\033[K") // restore the cursor position and clear the line
			log.Printf("%s PRICE is %.8f %s\n", scoin, price, dcoin)
			sma := calculateSMA(historicalPrices, period)
			ema := calculateEMA(historicalPrices, period)
			lastMacd, lastSignal, _, _ := calculateMACD(historicalPrices, 12, 26, 9)
			if ema[len(ema)-1] < sma[len(sma)-1] && ema[len(ema)-2] >= sma[len(sma)-2] && lastMacd < lastSignal && price > buyPrice {
				log.Printf("Creating new %s order\n", red("SELL"))
				sell, err := TradeSell(symbol, qty, price, sellFactor, roundPrice)
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
				break // indicators conditions meet
			}
			time.Sleep(20 * time.Second)
		}
		operation++
		time.Sleep(1 * time.Minute) // 1 minute to start next operation
	}
}
