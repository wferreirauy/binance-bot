package main

import (
	"fmt"
	"log"
	"strings"
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

func TradeBuy(ticker string, qty, basePrice, buyFactor float64, round int) (interface{}, error) {
	buyPrice := toFixed(basePrice*buyFactor, round)
	total := buyPrice * qty
	tick := strings.Replace(ticker, "/", "", -1)
	scoin, dcoin, found := strings.Cut(ticker, "/")
	if !found {
		log.Fatal("ticker malformed, / is missing ")
	}

	fmt.Printf("%sBUY%s %f %s - PRICE: %.8f - Total %s: %f\n",
		Green, Reset, qty, scoin, buyPrice, dcoin, total)

	order, err := NewOrder(tick, "BUY", qty, buyPrice)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func TradeSell(ticker string, qty, basePrice, sellFactor float64, round int) (interface{}, error) {
	sellPrice := toFixed(basePrice*sellFactor, round)
	total := sellPrice * qty
	tick := strings.Replace(ticker, "/", "", -1)
	scoin, dcoin, found := strings.Cut(ticker, "/")
	if !found {
		log.Fatal("ticker malformed, / is missing ")
	}

	fmt.Printf("%sSELL%s %f %s - PRICE: %.8f - Total %s: %f\n",
		Red, Reset, qty, scoin, sellPrice, dcoin, total)

	order, err := NewOrder(tick, "SELL", qty, sellPrice)
	if err != nil {
		return nil, err
	}
	return order, nil
}
