package ai

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Signal represents a trading recommendation
type Signal string

const (
	SignalBuy    Signal = "BUY"
	SignalSell   Signal = "SELL"
	SignalHold   Signal = "HOLD"
	SignalUnknown Signal = "UNKNOWN"
)

// AgentDecision is the output from a single AI agent
type AgentDecision struct {
	Agent      string
	Provider   Provider
	Signal     Signal
	Confidence float64 // 0.0 to 1.0
	Reasoning  string
}

// ConsensusResult aggregates decisions from all agents
type ConsensusResult struct {
	FinalSignal    Signal
	BuyScore       float64
	SellScore      float64
	HoldScore      float64
	AvgConfidence  float64
	Decisions      []AgentDecision
	SentimentData  *SentimentData
}

// TechnicalSnapshot contains the current state of all technical indicators
type TechnicalSnapshot struct {
	Symbol        string
	Price         float64
	PrevPrice     float64
	RSI           float64
	MACDLine      float64
	SignalLine     float64
	PrevMACDLine  float64
	PrevSignalLine float64
	UpperBand     float64
	LowerBand     float64
	DEMA          float64
	Tendency      string
	ADX           float64
	Volume        float64
	AvgVolume     float64
}

// FormatForPrompt returns technical data formatted for LLM consumption
func (ts *TechnicalSnapshot) FormatForPrompt() string {
	var b strings.Builder
	b.WriteString("=== TECHNICAL INDICATORS ===\n\n")
	b.WriteString(fmt.Sprintf("Symbol: %s\n", ts.Symbol))
	b.WriteString(fmt.Sprintf("Current Price: %.8f\n", ts.Price))
	b.WriteString(fmt.Sprintf("Previous Price: %.8f\n", ts.PrevPrice))
	b.WriteString(fmt.Sprintf("Price Change: %.4f%%\n", (ts.Price-ts.PrevPrice)/ts.PrevPrice*100))
	b.WriteString(fmt.Sprintf("RSI(14): %.2f\n", ts.RSI))
	b.WriteString(fmt.Sprintf("MACD Line: %.8f\n", ts.MACDLine))
	b.WriteString(fmt.Sprintf("MACD Signal: %.8f\n", ts.SignalLine))
	b.WriteString(fmt.Sprintf("MACD Crossover: %s\n", describeMACDCross(ts.PrevMACDLine, ts.PrevSignalLine, ts.MACDLine, ts.SignalLine)))
	b.WriteString(fmt.Sprintf("Bollinger Upper: %.8f\n", ts.UpperBand))
	b.WriteString(fmt.Sprintf("Bollinger Lower: %.8f\n", ts.LowerBand))
	b.WriteString(fmt.Sprintf("DEMA: %.8f\n", ts.DEMA))
	b.WriteString(fmt.Sprintf("Tendency: %s\n", ts.Tendency))
	if ts.ADX > 0 {
		b.WriteString(fmt.Sprintf("ADX: %.2f\n", ts.ADX))
	}
	if ts.AvgVolume > 0 {
		b.WriteString(fmt.Sprintf("Volume: %.2f (Avg: %.2f, Ratio: %.2fx)\n", ts.Volume, ts.AvgVolume, ts.Volume/ts.AvgVolume))
	}
	return b.String()
}

func describeMACDCross(prevMACD, prevSignal, curMACD, curSignal float64) string {
	if prevMACD <= prevSignal && curMACD > curSignal {
		return "BULLISH crossover (MACD crossed above signal)"
	}
	if prevMACD >= prevSignal && curMACD < curSignal {
		return "BEARISH crossover (MACD crossed below signal)"
	}
	if curMACD > curSignal {
		return "MACD above signal (bullish)"
	}
	return "MACD below signal (bearish)"
}

// Orchestrator manages multiple AI agents and produces consensus decisions
type Orchestrator struct {
	clients  []*LLMClient
	cacheTTL time.Duration
	mu       sync.Mutex
	lastSentiment *SentimentData
	lastFetch     time.Time
}

// NewOrchestrator builds an orchestrator from the available API keys
func NewOrchestrator(openaiKey, deepseekKey, claudeKey string, openaiModel, deepseekModel, claudeModel string) *Orchestrator {
	o := &Orchestrator{
		cacheTTL: 5 * time.Minute,
	}

	if openaiKey != "" {
		o.clients = append(o.clients, NewLLMClient(ProviderOpenAI, openaiKey, openaiModel))
	}
	if deepseekKey != "" {
		o.clients = append(o.clients, NewLLMClient(ProviderDeepSeek, deepseekKey, deepseekModel))
	}
	if claudeKey != "" {
		o.clients = append(o.clients, NewLLMClient(ProviderClaude, claudeKey, claudeModel))
	}

	return o
}

// IsEnabled returns true if at least one provider is configured
func (o *Orchestrator) IsEnabled() bool {
	return len(o.clients) > 0
}

// Analyze runs the multi-agent pipeline and returns a consensus
func (o *Orchestrator) Analyze(ctx context.Context, ts *TechnicalSnapshot, tradeType string) (*ConsensusResult, error) {
	if !o.IsEnabled() {
		return nil, fmt.Errorf("ai: no providers configured")
	}

	// Fetch or reuse cached sentiment data
	sentiment, err := o.getSentiment(ctx, ts.Symbol)
	if err != nil {
		log.Printf("AI: sentiment fetch error (continuing without): %v\n", err)
		sentiment = &SentimentData{
			NewsHeadlines:  []string{"(unavailable)"},
			FearGreedIndex: -1,
			FearGreedLabel: "unavailable",
			FetchedAt:      time.Now(),
		}
	}

	// Build the analysis prompt
	systemPrompt := buildSystemPrompt(tradeType)
	userPrompt := buildUserPrompt(ts, sentiment)

	// Query all providers concurrently
	var wg sync.WaitGroup
	decisions := make([]AgentDecision, len(o.clients))

	for i, client := range o.clients {
		wg.Add(1)
		go func(idx int, c *LLMClient) {
			defer wg.Done()
			decision := queryAgent(ctx, c, systemPrompt, userPrompt, tradeType)
			decisions[idx] = decision
		}(i, client)
	}
	wg.Wait()

	// Build consensus from all decisions
	result := buildConsensus(decisions, sentiment)
	return result, nil
}

func (o *Orchestrator) getSentiment(ctx context.Context, symbol string) (*SentimentData, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.lastSentiment != nil && time.Since(o.lastFetch) < o.cacheTTL {
		return o.lastSentiment, nil
	}

	sd, err := FetchSentimentData(ctx, symbol)
	if err != nil {
		return nil, err
	}

	o.lastSentiment = sd
	o.lastFetch = time.Now()
	return sd, nil
}

func buildSystemPrompt(tradeType string) string {
	return fmt.Sprintf(`You are an expert cryptocurrency trading analyst AI agent.
Your role is to analyze technical indicators and market sentiment to provide %s trading signals.

RULES:
1. Respond ONLY with a structured analysis in this EXACT format:
   SIGNAL: BUY|SELL|HOLD
   CONFIDENCE: 0.0-1.0
   REASONING: <one paragraph explanation>
2. Be conservative. When uncertain, recommend HOLD.
3. Consider ALL indicators together - no single indicator should dominate.
4. Sentiment from news and Fear & Greed index should influence but not override technical signals.
5. For BULL trades: look for buying opportunities (dips, oversold, bullish crossovers).
6. For BEAR trades: look for selling opportunities (peaks, overbought, bearish crossovers).
7. A Fear & Greed Index below 25 = Extreme Fear (contrarian BUY signal), above 75 = Extreme Greed (contrarian SELL signal).
8. Weight recent news sentiment: negative news in uptrend = caution, positive news in downtrend = caution.`, tradeType)
}

func buildUserPrompt(ts *TechnicalSnapshot, sd *SentimentData) string {
	var b strings.Builder
	b.WriteString("Analyze the following market data and provide your trading signal:\n\n")
	b.WriteString(ts.FormatForPrompt())
	b.WriteString("\n")
	b.WriteString(sd.FormatForPrompt())
	b.WriteString("\nProvide your analysis now.")
	return b.String()
}

func queryAgent(ctx context.Context, client *LLMClient, systemPrompt, userPrompt, tradeType string) AgentDecision {
	agentName := fmt.Sprintf("%s-agent", client.Provider)

	callCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	resp, err := client.Chat(callCtx, systemPrompt, []Message{
		{Role: "user", Content: userPrompt},
	})
	if err != nil {
		log.Printf("AI: %s error: %v\n", agentName, err)
		return AgentDecision{
			Agent:      agentName,
			Provider:   client.Provider,
			Signal:     SignalHold,
			Confidence: 0.0,
			Reasoning:  fmt.Sprintf("Provider error: %v", err),
		}
	}

	return parseAgentResponse(resp.Content, agentName, client.Provider)
}

func parseAgentResponse(content, agentName string, provider Provider) AgentDecision {
	d := AgentDecision{
		Agent:    agentName,
		Provider: provider,
		Signal:   SignalHold,
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		upper := strings.ToUpper(line)

		if strings.HasPrefix(upper, "SIGNAL:") {
			sigStr := strings.TrimSpace(strings.TrimPrefix(upper, "SIGNAL:"))
			switch {
			case strings.Contains(sigStr, "BUY"):
				d.Signal = SignalBuy
			case strings.Contains(sigStr, "SELL"):
				d.Signal = SignalSell
			case strings.Contains(sigStr, "HOLD"):
				d.Signal = SignalHold
			}
		}

		if strings.HasPrefix(upper, "CONFIDENCE:") {
			confStr := strings.TrimSpace(strings.TrimPrefix(upper, "CONFIDENCE:"))
			if val, err := strconv.ParseFloat(confStr, 64); err == nil {
				if val >= 0 && val <= 1 {
					d.Confidence = val
				}
			}
		}

		if strings.HasPrefix(upper, "REASONING:") {
			d.Reasoning = strings.TrimSpace(line[len("REASONING:"):])
		}
	}

	return d
}

func buildConsensus(decisions []AgentDecision, sentiment *SentimentData) *ConsensusResult {
	result := &ConsensusResult{
		Decisions:     decisions,
		SentimentData: sentiment,
	}

	totalWeight := 0.0
	for _, d := range decisions {
		weight := d.Confidence
		if weight <= 0 {
			weight = 0.1 // minimum weight for failed providers
		}
		totalWeight += weight

		switch d.Signal {
		case SignalBuy:
			result.BuyScore += weight
		case SignalSell:
			result.SellScore += weight
		case SignalHold:
			result.HoldScore += weight
		}
	}

	if totalWeight > 0 {
		result.BuyScore /= totalWeight
		result.SellScore /= totalWeight
		result.HoldScore /= totalWeight
	}

	// Determine final signal by weighted majority
	if result.BuyScore > result.SellScore && result.BuyScore > result.HoldScore {
		result.FinalSignal = SignalBuy
	} else if result.SellScore > result.BuyScore && result.SellScore > result.HoldScore {
		result.FinalSignal = SignalSell
	} else {
		result.FinalSignal = SignalHold
	}

	// Calculate average confidence
	confSum := 0.0
	for _, d := range decisions {
		confSum += d.Confidence
	}
	if len(decisions) > 0 {
		result.AvgConfidence = confSum / float64(len(decisions))
	}

	return result
}

// ShouldBuy returns true if AI consensus supports buying
func (cr *ConsensusResult) ShouldBuy() bool {
	return cr.FinalSignal == SignalBuy && cr.AvgConfidence >= 0.5
}

// ShouldSell returns true if AI consensus supports selling
func (cr *ConsensusResult) ShouldSell() bool {
	return cr.FinalSignal == SignalSell && cr.AvgConfidence >= 0.5
}

// String returns a human-readable summary
func (cr *ConsensusResult) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("AI Consensus: %s (confidence: %.0f%%)\n", cr.FinalSignal, cr.AvgConfidence*100))
	b.WriteString(fmt.Sprintf("  Scores - Buy: %.0f%% | Sell: %.0f%% | Hold: %.0f%%\n",
		cr.BuyScore*100, cr.SellScore*100, cr.HoldScore*100))
	for _, d := range cr.Decisions {
		b.WriteString(fmt.Sprintf("  [%s] %s (%.0f%%) - %s\n", d.Provider, d.Signal, d.Confidence*100, d.Reasoning))
	}
	return b.String()
}
