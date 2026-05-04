package config
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.yml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return p
}

// TestRead_AppliesDefaults verifies that an almost-empty YAML file still
// produces a usable Config (no zero-valued indicator periods that would
// silently disable downstream gates).
func TestRead_AppliesDefaults(t *testing.T) {
	path := writeTempConfig(t, "ai:\n  enabled: false\n")
	var c Config
	cfg, err := c.Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if cfg.HistoricalPrices.Period == 0 {
		t.Error("expected default HistoricalPrices.Period")
	}
	if cfg.Indicators.Rsi.Length == 0 {
		t.Error("expected default RSI length")
	}
	if cfg.Indicators.Macd.SlowLength <= cfg.Indicators.Macd.FastLength {
		t.Error("default MACD slow must be > fast")
	}
	if cfg.RefreshInterval == 0 {
		t.Error("expected default RefreshInterval")
	}
}

// TestRead_RejectsInvertedRSI guards against config typos that would make
// every entry pass the RSI gate trivially.
func TestRead_RejectsInvertedRSI(t *testing.T) {
	body := `
indicators:
  rsi:
    upper-limit: 30
    lower-limit: 70
`
	path := writeTempConfig(t, body)
	var c Config
	if _, err := c.Read(path); err == nil {
		t.Fatal("expected error for inverted RSI bounds")
	}
}

// TestRead_RejectsBadMACD ensures slow <= fast is rejected.
func TestRead_RejectsBadMACD(t *testing.T) {
	body := `
indicators:
  macd:
    fast-length: 26
    slow-length: 12
    signal-length: 9
`
	path := writeTempConfig(t, body)
	var c Config
	if _, err := c.Read(path); err == nil {
		t.Fatal("expected error for slow <= fast MACD periods")
	}
}

// TestRead_RejectsBadDirection ensures tendency.direction is constrained.
func TestRead_RejectsBadDirection(t *testing.T) {
	body := `
tendency:
  direction: "sideways"
`
	path := writeTempConfig(t, body)
	var c Config
	if _, err := c.Read(path); err == nil {
		t.Fatal("expected error for invalid tendency direction")
	}
}

// TestRead_RejectsBadAIConfidence ensures min-confidence is in [0,1].
func TestRead_RejectsBadAIConfidence(t *testing.T) {
	body := `
ai:
  enabled: true
  min-confidence: 1.5
`
	path := writeTempConfig(t, body)
	var c Config
	if _, err := c.Read(path); err == nil {
		t.Fatal("expected error for out-of-range AI min-confidence")
	}
}
