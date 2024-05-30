package main

import (
	"fmt"
	"log"
	"strings"

	color "github.com/fatih/color"
)

var red = color.New(color.FgRed, color.Bold).SprintFunc()
var green = color.New(color.FgGreen, color.Bold).SprintFunc()

func TradeBuy(ticker string, qty, basePrice, buyFactor float64, round int) (interface{}, error) {
	buyPrice := toFixed(basePrice*buyFactor, round)
	total := buyPrice * qty
	tick := strings.Replace(ticker, "/", "", -1)
	scoin, dcoin, found := strings.Cut(ticker, "/")
	if !found {
		log.Fatal("ticker malformed, / is missing ")
	}

	fmt.Printf("%s %f %s - PRICE: %.8f - Total %s: %f\n",
		green("BUY"), qty, scoin, buyPrice, dcoin, total)

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

	fmt.Printf("%s %f %s - PRICE: %.8f - Total %s: %f\n",
		red("SELL"), qty, scoin, sellPrice, dcoin, total)

	order, err := NewOrder(tick, "SELL", qty, sellPrice)
	if err != nil {
		return nil, err
	}
	return order, nil
}
