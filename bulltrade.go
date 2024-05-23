package main

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/fatih/color"
)

func BullTrade(ticker string, qty float64, buyFactor float64, sellFactor float64, operations int) {

	operation := 0
	var bid int64 = 0
	var sid int64 = 0

	for range operations {

		c := color.New(color.FgCyan, color.Bold)
		c.Printf("Operation #%d\n", operation)

		basePrice, err := GetPrice(ticker)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s PRICE: %.2f\n", ticker, basePrice)

		fmt.Println("New BUY order")
		// buy
		buy := TradeBuy(ticker, qty, basePrice, buyFactor)
		buyOrder := reflect.ValueOf(buy).Elem()
		bid = buyOrder.FieldByName("OrderId").Int()

		if getor, err := GetOrder(ticker, bid); err == nil {
			fmt.Printf("BUY order created. Id: %d - Status: %s\n", getor.OrderId, getor.Status)
			fmt.Printf("SELL order will be for price: %.2f\n", toFixed(basePrice*sellFactor, 2))
		}

		for { // looking at buy order until filled
			if getor, err := GetOrder(ticker, bid); err == nil {
				if getor.Status == "FILLED" {
					fmt.Println("BUY order filled!")
					fmt.Println("New SELL order")
					// sell
					sell := TradeSell(ticker, qty, basePrice, sellFactor)
					sellOrder := reflect.ValueOf(sell).Elem()
					sid = sellOrder.FieldByName("OrderId").Int()
					if getor, err := GetOrder(ticker, sid); err == nil {
						fmt.Printf("SELL order created. Id: %d - Status: %s\n", getor.OrderId, getor.Status)
					}
					break
				}
			}
			time.Sleep(10 * time.Second) // wait 10 secs to take another look
		}

		for { // looking at sell order until FILLED
			if getor, err := GetOrder(ticker, sid); err == nil {
				if getor.Status == "FILLED" {
					fmt.Printf("SELL order filled!\n\n")
					break
				}
			}
			time.Sleep(10 * time.Second) // wait 10 secs to take another look
		}
		operation++
		time.Sleep(10 * time.Second) // wait 10 secs for next operation
	}
}
