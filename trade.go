package main

import (
	"fmt"
	"log"
	"time"
)

func TradeSell(ticker string, qty float64, sellFactor float64) (interface{}, error) {
	currentPrice, err := GetPrice(ticker)
	if err != nil {
		log.Fatal(err)
	}

	dt := time.Now()
	fmt.Println("Order Time: ", dt.String())
	fmt.Printf("%s PRICE: %.2f\n", ticker, currentPrice)

	sellPrice := toFixed(currentPrice*sellFactor, 2)
	fmt.Printf("%s SELL PRICE: %.2f\n", ticker, sellPrice)

	order, err := NewOrder(ticker, "SELL", qty, sellPrice)
	if err != nil {
		return nil, err
	}

	return order, nil
}

func TradeBuy(ticker string, qty float64, buyFactor float64) (interface{}, error) {
	currentPrice, err := GetPrice(ticker)
	if err != nil {
		log.Fatal(err)
	}

	dt := time.Now()
	fmt.Println("Order Time: ", dt.String())
	fmt.Printf("%s PRICE: %.2f\n", ticker, currentPrice)

	buyPrice := toFixed(currentPrice*buyFactor, 2)
	fmt.Printf("%s BUY PRICE: %.2f\n", ticker, buyPrice)

	order, err := NewOrder(ticker, "BUY", qty, buyPrice)
	if err != nil {
		return nil, err
	}

	return order, nil
}
