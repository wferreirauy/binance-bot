package strategy

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/wferreirauy/binance-bot/ai"
	"github.com/wferreirauy/binance-bot/config"
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

// entryCondition represents a single scored entry condition with its name and result.
type entryCondition struct {
	Name string
	Met  bool
}

// logEntryConditions logs detailed entry condition breakdown to the orders panel.
func logEntryConditions(dash *tui.Dashboard, mode string, conditions []entryCondition, score, total, minScore int, scalp bool) {
	label := "Entry"
	if scalp {
		label = "Scalp entry"
	}
	dash.LogInfo(fmt.Sprintf("[yellow]%s: score %d/%d (min %d)[-]", label, score, total, minScore))
	for _, c := range conditions {
		tag := "[green]✓[-]"
		if !c.Met {
			tag = "[red]✗[-]"
		}
		dash.LogInfo(fmt.Sprintf("  %s %s", tag, c.Name))
	}
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

// logStartupStatus logs warnings about missing API keys to the Activity Log
// and updates the AI panel status when keys are not configured.
func logStartupStatus(dash *tui.Dashboard, cfg *config.Config, aiOrch *ai.Orchestrator) {
	// Check Binance API keys
	if exchange.APIKey == "" {
		dash.LogError("[yellow]BINANCE_API_KEY[-] environment variable is not set")
	}
	if exchange.SecretKey == "" {
		dash.LogError("[yellow]BINANCE_SECRET_KEY[-] environment variable is not set")
	}

	// Check AI agent keys and update AI panel
	if cfg.AI.Enabled {
		var missingKeys []string
		if os.Getenv("OPENAI_API_KEY") == "" {
			missingKeys = append(missingKeys, "OPENAI_API_KEY")
		}
		if os.Getenv("DEEPSEEK_API_KEY") == "" {
			missingKeys = append(missingKeys, "DEEPSEEK_API_KEY")
		}
		if os.Getenv("ANTHROPIC_API_KEY") == "" {
			missingKeys = append(missingKeys, "ANTHROPIC_API_KEY")
		}

		if len(missingKeys) > 0 {
			for _, key := range missingKeys {
				dash.LogInfo(fmt.Sprintf("[yellow]⚠[-] %s not set — provider disabled", key))
			}
		}

		if aiOrch != nil {
			dash.LogInfo("AI Agents: [green]ENABLED[-]")
		} else {
			dash.UpdateAIStatus(true, missingKeys)
		}
	} else {
		dash.UpdateAIStatus(false, nil)
	}
}
