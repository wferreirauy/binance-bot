package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
)

const (
	defaultRequestTimeout = 15 * time.Second
	defaultOrderFillWait  = 2 * time.Minute
	defaultPollInterval   = 10 * time.Second
)

type SymbolRules struct {
	Symbol             string
	BaseAsset          string
	QuoteAsset         string
	BaseAssetPrecision uint
	QuotePrecision     uint
	TickSize           float64
	StepSize           float64
	MinQty             float64
	MinNotional        float64
}

type exchangeInfoResponse struct {
	Symbols []struct {
		Symbol             string `json:"symbol"`
		BaseAsset          string `json:"baseAsset"`
		QuoteAsset         string `json:"quoteAsset"`
		BaseAssetPrecision uint   `json:"baseAssetPrecision"`
		QuotePrecision     uint   `json:"quotePrecision"`
		Filters            []struct {
			FilterType  string `json:"filterType"`
			TickSize    string `json:"tickSize"`
			StepSize    string `json:"stepSize"`
			MinQty      string `json:"minQty"`
			MinNotional string `json:"minNotional"`
		} `json:"filters"`
	} `json:"symbols"`
}

func fetchSymbolRules(ctx context.Context, symbol string) (SymbolRules, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	url := fmt.Sprintf("%s/api/v3/exchangeInfo?symbol=%s", strings.TrimRight(baseurl, "/"), symbol)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return SymbolRules{}, fmt.Errorf("exchange info request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return SymbolRules{}, fmt.Errorf("exchange info fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return SymbolRules{}, fmt.Errorf("exchange info fetch: unexpected status %s", resp.Status)
	}

	var payload exchangeInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return SymbolRules{}, fmt.Errorf("exchange info decode: %w", err)
	}
	if len(payload.Symbols) != 1 {
		return SymbolRules{}, fmt.Errorf("symbol %s not found on Binance", symbol)
	}

	raw := payload.Symbols[0]
	rules := SymbolRules{
		Symbol:             raw.Symbol,
		BaseAsset:          raw.BaseAsset,
		QuoteAsset:         raw.QuoteAsset,
		BaseAssetPrecision: raw.BaseAssetPrecision,
		QuotePrecision:     raw.QuotePrecision,
	}

	for _, filter := range raw.Filters {
		switch filter.FilterType {
		case "PRICE_FILTER":
			rules.TickSize = parseFloat(filter.TickSize)
		case "LOT_SIZE":
			rules.StepSize = parseFloat(filter.StepSize)
			rules.MinQty = parseFloat(filter.MinQty)
		case "MIN_NOTIONAL", "NOTIONAL":
			if rules.MinNotional == 0 {
				rules.MinNotional = parseFloat(filter.MinNotional)
			}
		}
	}

	if rules.TickSize == 0 || rules.StepSize == 0 {
		return SymbolRules{}, fmt.Errorf("symbol %s is missing required Binance filters", symbol)
	}
	return rules, nil
}

func floorToStep(value, step float64) float64 {
	if step <= 0 {
		return value
	}
	return math.Floor((value/step)+1e-9) * step
}

func decimalPlacesFromStep(step float64, fallback uint) uint {
	if step <= 0 {
		return fallback
	}
	for precision := uint(0); precision <= 12; precision++ {
		scaled := step * math.Pow(10, float64(precision))
		if math.Abs(scaled-math.Round(scaled)) < 1e-9 {
			return precision
		}
	}
	return fallback
}

func normalizeOrder(symbol string, quantity, price float64, rules SymbolRules) (float64, float64, error) {
	adjQty := floorToStep(quantity, rules.StepSize)
	adjPrice := floorToStep(price, rules.TickSize)
	qtyPrecision := decimalPlacesFromStep(rules.StepSize, rules.BaseAssetPrecision)
	pricePrecision := decimalPlacesFromStep(rules.TickSize, rules.QuotePrecision)
	adjQty = roundFloat(adjQty, qtyPrecision)
	adjPrice = roundFloat(adjPrice, pricePrecision)

	if adjQty <= 0 {
		return 0, 0, fmt.Errorf("%s quantity %.12f becomes invalid after applying step size %.12f", symbol, quantity, rules.StepSize)
	}
	if rules.MinQty > 0 && adjQty < rules.MinQty {
		return 0, 0, fmt.Errorf("%s quantity %.12f is below Binance minimum %.12f", symbol, adjQty, rules.MinQty)
	}
	if adjPrice <= 0 {
		return 0, 0, fmt.Errorf("%s price %.12f becomes invalid after applying tick size %.12f", symbol, price, rules.TickSize)
	}
	if rules.MinNotional > 0 && (adjQty*adjPrice) < rules.MinNotional {
		return 0, 0, fmt.Errorf("%s notional %.12f is below Binance minimum %.12f", symbol, adjQty*adjPrice, rules.MinNotional)
	}
	return adjQty, adjPrice, nil
}

func normalizeMarketQuantity(symbol string, quantity float64, rules SymbolRules) (float64, error) {
	adjQty := floorToStep(quantity, rules.StepSize)
	qtyPrecision := decimalPlacesFromStep(rules.StepSize, rules.BaseAssetPrecision)
	adjQty = roundFloat(adjQty, qtyPrecision)
	if adjQty <= 0 {
		return 0, fmt.Errorf("%s quantity %.12f becomes invalid after applying step size %.12f", symbol, quantity, rules.StepSize)
	}
	if rules.MinQty > 0 && adjQty < rules.MinQty {
		return 0, fmt.Errorf("%s quantity %.12f is below Binance minimum %.12f", symbol, adjQty, rules.MinQty)
	}
	return adjQty, nil
}
