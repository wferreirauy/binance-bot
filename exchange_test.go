package main

import "testing"

func TestNormalizeOrderUsesExchangeFilters(t *testing.T) {
	rules := SymbolRules{
		Symbol:             "BTCUSDT",
		BaseAssetPrecision: 6,
		QuotePrecision:     2,
		TickSize:           0.10,
		StepSize:           0.001,
		MinQty:             0.001,
		MinNotional:        10,
	}

	qty, price, err := normalizeOrder("BTCUSDT", 0.123456, 43123.987, rules)
	if err != nil {
		t.Fatalf("normalizeOrder returned unexpected error: %v", err)
	}
	if qty != 0.123 {
		t.Fatalf("expected qty 0.123, got %f", qty)
	}
	if price != 43123.90 {
		t.Fatalf("expected price 43123.90, got %f", price)
	}
}

func TestNormalizeOrderRejectsBelowMinNotional(t *testing.T) {
	rules := SymbolRules{
		Symbol:      "ADAUSDT",
		TickSize:    0.0001,
		StepSize:    0.1,
		MinQty:      0.1,
		MinNotional: 10,
	}

	_, _, err := normalizeOrder("ADAUSDT", 5, 1.5, rules)
	if err == nil {
		t.Fatal("expected min notional validation error")
	}
}

func TestValidateBullTradeInputs(t *testing.T) {
	if err := validateBullTradeInputs("BTC/USDT", 10, 3, 2.5, 0.9999, 1.0001, 1); err != nil {
		t.Fatalf("expected valid inputs, got error: %v", err)
	}
	if err := validateBullTradeInputs("BTCUSDT", 10, 3, 2.5, 0.9999, 1.0001, 1); err == nil {
		t.Fatal("expected invalid ticker error")
	}
	if err := validateBullTradeInputs("BTC/USDT", 0, 3, 2.5, 0.9999, 1.0001, 1); err == nil {
		t.Fatal("expected invalid amount error")
	}
}
