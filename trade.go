package main

import (
	"fmt"
	"time"
)

func TradeSell(ticker string, qty float64, basePrice float64, sellFactor float64) interface{} {
	dt := time.Now()
	fmt.Println("Order Time: ", dt.Format(time.UnixDate))

	sellPrice := toFixed(basePrice*sellFactor, 2)
	fmt.Printf("%s SELL PRICE: %.2f\n", ticker, sellPrice)

	order := NewOrder(ticker, "SELL", qty, sellPrice)
	return order
}

func TradeBuy(ticker string, qty float64, basePrice float64, buyFactor float64) interface{} {
	dt := time.Now()
	fmt.Println("Order Time: ", dt.Format(time.UnixDate))

	buyPrice := toFixed(basePrice*buyFactor, 2)
	fmt.Printf("%s BUY PRICE: %.2f\n", ticker, buyPrice)

	order := NewOrder(ticker, "BUY", qty, buyPrice)
	return order
}
