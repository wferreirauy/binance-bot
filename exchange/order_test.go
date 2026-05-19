package exchange

import "testing"

func TestRequiredBalanceBuyUsesQuoteAsset(t *testing.T) {
	asset, required, err := requiredBalance("BUY", 2.5, 10, &SymbolFilters{
		BaseAsset:  "XRP",
		QuoteAsset: "USDT",
	})
	if err != nil {
		t.Fatalf("requiredBalance returned error: %v", err)
	}
	if asset != "USDT" {
		t.Fatalf("expected quote asset USDT, got %s", asset)
	}
	if required != 25 {
		t.Fatalf("expected required quote balance 25, got %f", required)
	}
}

func TestRequiredBalanceSellUsesBaseAsset(t *testing.T) {
	asset, required, err := requiredBalance("SELL", 22.5771, 10, &SymbolFilters{
		BaseAsset:  "XRP",
		QuoteAsset: "USDT",
	})
	if err != nil {
		t.Fatalf("requiredBalance returned error: %v", err)
	}
	if asset != "XRP" {
		t.Fatalf("expected base asset XRP, got %s", asset)
	}
	if required != 22.5771 {
		t.Fatalf("expected required base balance 22.5771, got %f", required)
	}
}

func TestRequiredBalanceRejectsBuyWithoutPrice(t *testing.T) {
	_, _, err := requiredBalance("BUY", 1, 0, &SymbolFilters{
		BaseAsset:  "BTC",
		QuoteAsset: "USDT",
	})
	if err == nil {
		t.Fatal("expected BUY without price to fail")
	}
}

func TestOrderBalanceCheckSufficientAllowsTinyFloatDrift(t *testing.T) {
	check := &OrderBalanceCheck{
		Required:  10,
		Available: 10 - 5e-13,
	}
	if !check.Sufficient() {
		t.Fatal("expected tiny float drift to still be sufficient")
	}
}

func TestOrderBalanceCheckSufficientRejectsShortBalance(t *testing.T) {
	check := &OrderBalanceCheck{
		Required:  10,
		Available: 9.99,
	}
	if check.Sufficient() {
		t.Fatal("expected short balance to be insufficient")
	}
}
