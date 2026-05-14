package config

import (
	"strings"
	"testing"
)

func TestValidateAcceptsSampleConfig(t *testing.T) {
	var c Config
	cfg, err := c.Read("../sample-binance-config.yml")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	errs := cfg.Validate()
	if len(errs) > 0 {
		t.Fatalf("expected sample config to be valid, got %v", errs)
	}
}

func TestValidateReportsMultipleIssues(t *testing.T) {
	var c Config
	cfg, err := c.Read("../sample-binance-config.yml")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	cfg.HistoricalPrices.Interval = "2m"
	cfg.Tendency.Direction = "sideways"
	cfg.Indicators.Rsi.LowerLimit = 80
	cfg.Indicators.Macd.FastLength = 26
	cfg.Indicators.Macd.SlowLength = 12
	cfg.AI.Enabled = true
	cfg.AI.MinConfidence = 1.2
	cfg.TopGainers.Limit = 0

	errs := cfg.Validate()
	got := errorsText(errs)

	want := []string{
		"historical-prices.interval must be a valid Binance interval",
		"tendency.direction must be either \"up\" or \"down\"",
		"indicators.rsi limits must satisfy lower-limit < middle-limit < upper-limit",
		"indicators.macd.fast-length must be less than slow-length",
		"ai.min-confidence must be between 0 and 1",
		"top-gainers.limit must be greater than 0",
	}
	for _, msg := range want {
		if !strings.Contains(got, msg) {
			t.Fatalf("expected validation error %q in:\n%s", msg, got)
		}
	}
}

func errorsText(errs []error) string {
	var b strings.Builder
	for _, err := range errs {
		b.WriteString(err.Error())
		b.WriteByte('\n')
	}
	return b.String()
}
