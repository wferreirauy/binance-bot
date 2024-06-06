package main

import (
	"log"
	"strconv"
	"strings"

	color "github.com/fatih/color"
)

var red = color.New(color.FgHiRed, color.Bold).SprintFunc()
var green = color.New(color.FgHiGreen, color.Bold).SprintFunc()
var white = color.New(color.FgHiWhite, color.Bold).SprintFunc()

func TradeBuy(ticker string, qty, basePrice, buyFactor float64, round uint) (interface{}, error) {
	buyPrice := roundFloat(basePrice*buyFactor, round)
	total := buyPrice * qty
	tick := strings.Replace(ticker, "/", "", -1)
	scoin, dcoin, found := strings.Cut(ticker, "/")
	if !found {
		log.Fatal("ticker malformed, \"/\" is missing ")
	}

	log.Printf("%s %f %s - PRICE: %s - Total %s: %f\n",
		green("BUY"), qty, scoin, white(strconv.FormatFloat(buyPrice, 'f', int(round), 64)), dcoin, total)

	order, err := NewOrder(tick, "BUY", qty, buyPrice)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func TradeSell(ticker string, qty, basePrice, sellFactor float64, round uint) (interface{}, error) {
	sellPrice := roundFloat(basePrice*sellFactor, round)
	total := sellPrice * qty
	tick := strings.Replace(ticker, "/", "", -1)
	scoin, dcoin, found := strings.Cut(ticker, "/")
	if !found {
		log.Fatal("ticker malformed, \"/\" is missing ")
	}

	log.Printf("%s %f %s - PRICE: %s - Total %s: %f\n",
		red("SELL"), qty, scoin, white(strconv.FormatFloat(sellPrice, 'f', int(round), 64)), dcoin, total)

	order, err := NewOrder(tick, "SELL", qty, sellPrice)
	if err != nil {
		return nil, err
	}
	return order, nil
}
