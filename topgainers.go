package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/wferreirauy/binance-bot/config"
	"github.com/wferreirauy/binance-bot/tui"
)

// ticker24hrResult mirrors Binance's 24hr ticker response with int64 for
// firstId/lastId to handle the -1 values the API may return.
type ticker24hrResult struct {
	Symbol             string `json:"symbol"`
	PriceChangePercent string `json:"priceChangePercent"`
	LastPrice          string `json:"lastPrice"`
	Volume             string `json:"volume"`
	QuoteVolume        string `json:"quoteVolume"`
}

// GainerEntry represents a single top-gainer ticker.
type GainerEntry struct {
	Symbol       string
	LastPrice    float64
	ChangePct    float64
	Volume       float64
	QuoteVolume  float64
}

// TopGainers launches the top gainers monitoring TUI.
func TopGainers(configFile string) {
	var c config.Config
	cfg, err := c.Read(configFile)
	if err != nil {
		log.Fatal(err)
	}

	quoteAsset := cfg.TopGainers.QuoteAsset
	if quoteAsset == "" {
		quoteAsset = "USDT"
	}
	limit := cfg.TopGainers.Limit
	if limit <= 0 {
		limit = 20
	}
	pollSecs := cfg.TopGainers.PollInterval
	if pollSecs <= 0 {
		pollSecs = 60
	}
	pollInterval := time.Duration(pollSecs) * time.Second

	// build exclude set
	excludeSet := make(map[string]bool)
	for _, s := range cfg.TopGainers.ExcludeSymbols {
		excludeSet[strings.ToUpper(s)] = true
	}

	dash := tui.NewGainersDashboard(quoteAsset, limit, pollInterval)

	go func() {
		defer dash.Stop()
		dash.LogInfo(fmt.Sprintf("Monitoring top %d gainers for %s (poll every %ds)", limit, quoteAsset, pollSecs))

		for {
			gainers, err := fetchTopGainers(quoteAsset, cfg.TopGainers.MinVolume, excludeSet, limit)
			if err != nil {
				dash.LogError(fmt.Sprintf("Fetch failed: %v", err))
			} else {
				entries := make([]tui.GainerRow, len(gainers))
				for i, g := range gainers {
					entries[i] = tui.GainerRow{
						Rank:        i + 1,
						Symbol:      g.Symbol,
						Price:       g.LastPrice,
						ChangePct:   g.ChangePct,
						Volume:      g.Volume,
						QuoteVolume: g.QuoteVolume,
					}
				}
				dash.UpdateGainers(entries)
			}
			time.Sleep(pollInterval)
		}
	}()

	if err := dash.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}

// fetchTopGainers fetches 24hr tickers, filters by quote asset and volume,
// then returns the top N sorted by price change percentage (descending).
func fetchTopGainers(
	quoteAsset string,
	minVolume float64,
	excludeSet map[string]bool,
	limit int,
) ([]GainerEntry, error) {
	resp, err := http.Get(baseurl + "/api/v3/ticker/24hr")
	if err != nil {
		return nil, fmt.Errorf("24hr ticker: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("24hr ticker read: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("24hr ticker: HTTP %d", resp.StatusCode)
	}

	var tickers []ticker24hrResult
	if err := json.Unmarshal(body, &tickers); err != nil {
		return nil, fmt.Errorf("24hr ticker decode: %w", err)
	}

	var gainers []GainerEntry
	for _, t := range tickers {
		if !strings.HasSuffix(t.Symbol, quoteAsset) {
			continue
		}
		if excludeSet[t.Symbol] {
			continue
		}

		changePct, err := strconv.ParseFloat(t.PriceChangePercent, 64)
		if err != nil || changePct <= 0 {
			continue
		}
		lastPrice, _ := strconv.ParseFloat(t.LastPrice, 64)
		volume, _ := strconv.ParseFloat(t.Volume, 64)
		quoteVol, _ := strconv.ParseFloat(t.QuoteVolume, 64)

		if minVolume > 0 && quoteVol < minVolume {
			continue
		}

		gainers = append(gainers, GainerEntry{
			Symbol:      t.Symbol,
			LastPrice:   lastPrice,
			ChangePct:   changePct,
			Volume:      volume,
			QuoteVolume: quoteVol,
		})
	}

	sort.Slice(gainers, func(i, j int) bool {
		return gainers[i].ChangePct > gainers[j].ChangePct
	})

	if len(gainers) > limit {
		gainers = gainers[:limit]
	}

	return gainers, nil
}
