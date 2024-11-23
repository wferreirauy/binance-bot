package main

import (
	"strings"
)

func TradeBuy(ticker string, qty, basePrice, buyFactor float64, round uint) (any, error) {
	buyPrice := roundFloat(basePrice*buyFactor, round)
	tick := strings.Replace(ticker, "/", "", -1)
	order, err := NewOrder(tick, "BUY", qty, buyPrice)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func TradeSell(ticker string, qty, basePrice, sellFactor float64, round uint) (any, error) {
	sellPrice := roundFloat(basePrice*sellFactor, round)
	tick := strings.Replace(ticker, "/", "", -1)
	order, err := NewOrder(tick, "SELL", qty, sellPrice)
	if err != nil {
		return nil, err
	}
	return order, nil
}
