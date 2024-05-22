package main

import (
	"fmt"
	"log"
	"reflect"
	"time"
)

func BullTrade(ticker string, qty float64, sellFactor float64, buyFactor float64, operations int) {

	basePrice, err := GetPrice(ticker)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s PRICE: %.2f\n", ticker, basePrice)

	var bid int64 = 0
	var sid int64 = 0

	operation := 0

	for range operations {
		fmt.Printf("Operation: #%d\n", operation)

		// buy
		buy := TradeBuy(ticker, qty, basePrice, buyFactor)
		buyOrder := reflect.ValueOf(buy).Elem()
		bid = buyOrder.FieldByName("OrderId").Int()

		if getor, err := GetOrder(ticker, bid); err == nil {
			fmt.Printf("BUY order created. Id: %d - Status: %s\n\n", getor.OrderId, getor.Status)
		}

		for { // looking at buy order until filled
			if getor, err := GetOrder(ticker, bid); err == nil {
				if getor.Status == "FILLED" {
					// sell
					sell := TradeSell(ticker, qty, basePrice, sellFactor)
					sellOrder := reflect.ValueOf(sell).Elem()
					sid = sellOrder.FieldByName("OrderId").Int()
					fmt.Printf("BUY order filled! Creating new SELL order...\n")
					if getor, err := GetOrder(ticker, sid); err == nil {
						fmt.Printf("SELL order created. Id: %d - Status: %s\n\n", getor.OrderId, getor.Status)
					}
					break
				}
			}
			time.Sleep(10 * time.Second)
		}

		for { // looking at sell order until FILLED
			if getor, err := GetOrder(ticker, sid); err == nil {
				if getor.Status == "FILLED" {
					fmt.Printf("SELL order filled!\n")
					break
				}
			}
		}

		time.Sleep(10 * time.Second)
	}
}
