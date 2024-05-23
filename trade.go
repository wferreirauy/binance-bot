package main

import (
	"fmt"
	"log"
	"time"
)

var Reset = "\033[0m"
var Red = "\033[31m"
var Green = "\033[32m"
var Yellow = "\033[33m"
var Blue = "\033[34m"
var Magenta = "\033[35m"
var Cyan = "\033[36m"
var Gray = "\033[37m"
var White = "\033[97m"

func TradeSell(ticker string, qty float64, basePrice float64, sellFactor float64) interface{} {
	dt := time.Now()
	fmt.Println("Order Time: ", dt.Format(time.UnixDate))

	sellPrice := toFixed(basePrice*sellFactor, 2)
	fmt.Printf("%sSELL%s %s - PRICE: %.2f\n", Red, Reset, ticker, sellPrice)

	order, err := NewOrder(ticker, "SELL", qty, sellPrice)
	if err != nil {
		log.Fatal(err)
	}
	return order
}

func TradeBuy(ticker string, qty float64, basePrice float64, buyFactor float64) interface{} {
	dt := time.Now()
	fmt.Println("Order Time: ", dt.Format(time.UnixDate))

	buyPrice := toFixed(basePrice*buyFactor, 2)
	fmt.Printf("%sBUY%s %s - PRICE: %.2f\n", Green, Reset, ticker, buyPrice)

	order, err := NewOrder(ticker, "BUY", qty, buyPrice)
	if err != nil {
		log.Fatal(err)
	}
	return order
}
