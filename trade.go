package main

import (
	"context"
	"strings"
)

func TradeBuy(ctx context.Context, ticker string, qty, basePrice, buyFactor float64, round uint) (any, error) {
	buyPrice := roundFloat(basePrice*buyFactor, round)
	tick := strings.Replace(ticker, "/", "", -1)
	order, err := NewOrder(ctx, tick, "BUY", qty, buyPrice)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func TradeSell(ctx context.Context, ticker string, qty, basePrice, sellFactor float64, round uint) (any, error) {
	sellPrice := roundFloat(basePrice*sellFactor, round)
	tick := strings.Replace(ticker, "/", "", -1)
	order, err := NewOrder(ctx, tick, "SELL", qty, sellPrice)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func TradeStopLossSell(ctx context.Context, ticker string, qty float64) (any, error) {
	tick := strings.Replace(ticker, "/", "", -1)
	order, err := NewMarketOrder(ctx, tick, "SELL", qty)
	if err != nil {
		return nil, err
	}
	return order, nil
}
