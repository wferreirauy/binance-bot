package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SentimentData holds aggregated market sentiment from multiple sources
type SentimentData struct {
	NewsHeadlines  []string
	FearGreedIndex int
	FearGreedLabel string
	FetchedAt      time.Time
}

// newsArticle is a CryptoCompare news API item
type newsArticle struct {
	Title      string `json:"title"`
	Body       string `json:"body"`
	Categories string `json:"categories"`
	Source     string `json:"source"`
}

// FetchSentimentData gathers sentiment data from free public APIs
func FetchSentimentData(ctx context.Context, symbol string) (*SentimentData, error) {
	sd := &SentimentData{FetchedAt: time.Now()}

	client := &http.Client{Timeout: 15 * time.Second}

	// Fetch news headlines from CryptoCompare (free, no key required)
	headlines, err := fetchCryptoNews(ctx, client, symbol)
	if err != nil {
		// non-fatal: continue with empty headlines
		sd.NewsHeadlines = []string{fmt.Sprintf("(news unavailable: %v)", err)}
	} else {
		sd.NewsHeadlines = headlines
	}

	// Fetch Fear & Greed Index from alternative.me (free, no key required)
	fgi, label, err := fetchFearGreedIndex(ctx, client)
	if err != nil {
		sd.FearGreedIndex = -1
		sd.FearGreedLabel = "unavailable"
	} else {
		sd.FearGreedIndex = fgi
		sd.FearGreedLabel = label
	}

	return sd, nil
}

func fetchCryptoNews(ctx context.Context, client *http.Client, symbol string) ([]string, error) {
	// Extract the base coin from symbol (e.g., "BTC/USDT" -> "BTC")
	coin := symbol
	if idx := strings.Index(symbol, "/"); idx > 0 {
		coin = symbol[:idx]
	}

	url := fmt.Sprintf("https://min-api.cryptocompare.com/data/v2/news/?lang=EN&categories=%s", coin)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []newsArticle `json:"Data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse news: %w", err)
	}

	// Take top 15 headlines for the AI to analyze
	limit := 15
	if len(result.Data) < limit {
		limit = len(result.Data)
	}

	headlines := make([]string, 0, limit)
	for _, article := range result.Data[:limit] {
		headlines = append(headlines, fmt.Sprintf("[%s] %s", article.Source, article.Title))
	}

	return headlines, nil
}

func fetchFearGreedIndex(ctx context.Context, client *http.Client) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.alternative.me/fng/", nil)
	if err != nil {
		return 0, "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", err
	}

	var result struct {
		Data []struct {
			Value               string `json:"value"`
			ValueClassification string `json:"value_classification"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, "", fmt.Errorf("parse fear-greed: %w", err)
	}
	if len(result.Data) == 0 {
		return 0, "", fmt.Errorf("no fear-greed data")
	}

	var val int
	fmt.Sscanf(result.Data[0].Value, "%d", &val)
	return val, result.Data[0].ValueClassification, nil
}

// FormatForPrompt returns a text block ready to include in an LLM prompt
func (sd *SentimentData) FormatForPrompt() string {
	var b strings.Builder
	b.WriteString("=== MARKET SENTIMENT DATA ===\n\n")

	b.WriteString(fmt.Sprintf("Fear & Greed Index: %d (%s)\n\n", sd.FearGreedIndex, sd.FearGreedLabel))

	b.WriteString("Recent News Headlines:\n")
	for i, h := range sd.NewsHeadlines {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, h))
	}

	return b.String()
}
