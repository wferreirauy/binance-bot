package main

import (
	"fmt"
	"strings"
)

func TradeBuy(ticker string, qty, basePrice, buyFactor float64, round uint) (any, error) {
	buyPrice := roundFloat(basePrice*buyFactor, round)
	tick := strings.Replace(ticker, "/", "", -1)

	filters, err := GetSymbolFilters(tick)
	if err != nil {
		return nil, fmt.Errorf("buy: %w", err)
	}
	adjQty, adjusted := AdjustQuantity(qty, buyPrice, filters, round)
	if adjusted {
		fmt.Printf("BUY qty adjusted from %.8f to %.8f to meet exchange filters (minNotional=%.2f)\n", qty, adjQty, filters.MinNotional)
	}

	order, err := NewOrder(tick, "BUY", adjQty, buyPrice)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func TradeSell(ticker string, qty, basePrice, sellFactor float64, round uint) (any, error) {
	sellPrice := roundFloat(basePrice*sellFactor, round)
	tick := strings.Replace(ticker, "/", "", -1)

	filters, err := GetSymbolFilters(tick)
	if err != nil {
		return nil, fmt.Errorf("sell: %w", err)
	}
	adjQty, adjusted := AdjustQuantity(qty, sellPrice, filters, round)
	if adjusted {
		fmt.Printf("SELL qty adjusted from %.8f to %.8f to meet exchange filters (minNotional=%.2f)\n", qty, adjQty, filters.MinNotional)
	}

	order, err := NewOrder(tick, "SELL", adjQty, sellPrice)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func TradeMarketBuy(ticker string, qty, estimatedPrice float64, round uint) (any, error) {
	tick := strings.Replace(ticker, "/", "", -1)

	filters, err := GetSymbolFilters(tick)
	if err != nil {
		return nil, fmt.Errorf("market buy: %w", err)
	}
	adjQty, adjusted := AdjustQuantity(qty, estimatedPrice, filters, round)
	if adjusted {
		fmt.Printf("MARKET BUY qty adjusted from %.8f to %.8f to meet exchange filters (minNotional=%.2f)\n", qty, adjQty, filters.MinNotional)
	}

	order, err := NewMarketOrder(tick, "BUY", adjQty)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func TradeMarketSell(ticker string, qty, estimatedPrice float64, round uint) (any, error) {
	tick := strings.Replace(ticker, "/", "", -1)

	filters, err := GetSymbolFilters(tick)
	if err != nil {
		return nil, fmt.Errorf("market sell: %w", err)
	}
	adjQty, adjusted := AdjustQuantity(qty, estimatedPrice, filters, round)
	if adjusted {
		fmt.Printf("MARKET SELL qty adjusted from %.8f to %.8f to meet exchange filters (minNotional=%.2f)\n", qty, adjQty, filters.MinNotional)
	}

	order, err := NewMarketOrder(tick, "SELL", adjQty)
	if err != nil {
		return nil, err
	}
	return order, nil
}
