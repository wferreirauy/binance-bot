package strategy

import (
	"fmt"
	"strings"
	"time"

	"github.com/wferreirauy/binance-bot/ai"
	"github.com/wferreirauy/binance-bot/exchange"
	"github.com/wferreirauy/binance-bot/indicator"
	"github.com/wferreirauy/binance-bot/tui"
)

// TradeBuy places a LIMIT buy order
func TradeBuy(ticker string, qty, basePrice, buyFactor float64, round uint) (any, error) {
	buyPrice := indicator.RoundFloat(basePrice*buyFactor, round)
	tick := strings.Replace(ticker, "/", "", -1)

	filters, err := exchange.GetSymbolFilters(tick)
	if err != nil {
		return nil, fmt.Errorf("buy: %w", err)
	}
	adjQty, adjusted := exchange.AdjustQuantity(qty, buyPrice, filters, round)
	if adjusted {
		fmt.Printf("BUY qty adjusted from %.8f to %.8f to meet exchange filters (minNotional=%.2f)\n", qty, adjQty, filters.MinNotional)
	}

	order, err := exchange.NewOrder(tick, "BUY", adjQty, buyPrice)
	if err != nil {
		return nil, err
	}
	return order, nil
}

// TradeSell places a LIMIT sell order
func TradeSell(ticker string, qty, basePrice, sellFactor float64, round uint) (any, error) {
	sellPrice := indicator.RoundFloat(basePrice*sellFactor, round)
	tick := strings.Replace(ticker, "/", "", -1)

	filters, err := exchange.GetSymbolFilters(tick)
	if err != nil {
		return nil, fmt.Errorf("sell: %w", err)
	}
	adjQty, adjusted := exchange.AdjustQuantity(qty, sellPrice, filters, round)
	if adjusted {
		fmt.Printf("SELL qty adjusted from %.8f to %.8f to meet exchange filters (minNotional=%.2f)\n", qty, adjQty, filters.MinNotional)
	}

	order, err := exchange.NewOrder(tick, "SELL", adjQty, sellPrice)
	if err != nil {
		return nil, err
	}
	return order, nil
}

// TradeMarketBuy places a MARKET buy order
func TradeMarketBuy(ticker string, qty, estimatedPrice float64, round uint) (any, error) {
	tick := strings.Replace(ticker, "/", "", -1)

	filters, err := exchange.GetSymbolFilters(tick)
	if err != nil {
		return nil, fmt.Errorf("market buy: %w", err)
	}
	adjQty, adjusted := exchange.AdjustQuantity(qty, estimatedPrice, filters, round)
	if adjusted {
		fmt.Printf("MARKET BUY qty adjusted from %.8f to %.8f to meet exchange filters (minNotional=%.2f)\n", qty, adjQty, filters.MinNotional)
	}

	order, err := exchange.NewMarketOrder(tick, "BUY", adjQty)
	if err != nil {
		return nil, err
	}
	return order, nil
}

// TradeMarketSell places a MARKET sell order
func TradeMarketSell(ticker string, qty, estimatedPrice float64, round uint) (any, error) {
	tick := strings.Replace(ticker, "/", "", -1)

	filters, err := exchange.GetSymbolFilters(tick)
	if err != nil {
		return nil, fmt.Errorf("market sell: %w", err)
	}
	adjQty, adjusted := exchange.AdjustQuantity(qty, estimatedPrice, filters, round)
	if adjusted {
		fmt.Printf("MARKET SELL qty adjusted from %.8f to %.8f to meet exchange filters (minNotional=%.2f)\n", qty, adjQty, filters.MinNotional)
	}

	order, err := exchange.NewMarketOrder(tick, "SELL", adjQty)
	if err != nil {
		return nil, err
	}
	return order, nil
}

// updateDashAI converts an ai.ConsensusResult into tui.AIConsensusData and updates the dashboard.
func updateDashAI(dash *tui.Dashboard, cr *ai.ConsensusResult) {
	data := &tui.AIConsensusData{
		FinalSignal:    string(cr.FinalSignal),
		AvgConfidence:  cr.AvgConfidence,
		BuyScore:       cr.BuyScore,
		SellScore:      cr.SellScore,
		HoldScore:      cr.HoldScore,
		FearGreed:      -1,
		FearGreedLabel: "",
	}
	if cr.SentimentData != nil {
		data.FearGreed = cr.SentimentData.FearGreedIndex
		data.FearGreedLabel = cr.SentimentData.FearGreedLabel
	}
	for _, d := range cr.Decisions {
		data.Agents = append(data.Agents, tui.AgentResult{
			Provider:   string(d.Provider),
			Signal:     string(d.Signal),
			Confidence: d.Confidence,
			Reasoning:  d.Reasoning,
		})
	}
	dash.UpdateAI(data)
}

// waitOrderFilled polls until an order is filled, logging the result.
func waitOrderFilled(dash *tui.Dashboard, ticker string, orderId int64, filledMsg string, interval time.Duration) {
	for {
		if getor, err := exchange.GetOrder(ticker, orderId); err == nil {
			if getor.Status == "FILLED" {
				dash.LogOrder(filledMsg)
				return
			}
		}
		time.Sleep(interval)
	}
}
