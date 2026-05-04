package ai

import "testing"

func TestParseAgentResponseRequiresExactSignal(t *testing.T) {
	decision := parseAgentResponse(`SIGNAL: BUY OR HOLD
CONFIDENCE: 0.9
REASONING: ambiguous`, "test-agent", ProviderOpenAI)

	if decision.Signal != SignalHold {
		t.Fatalf("expected ambiguous signal to default to HOLD, got %s", decision.Signal)
	}
}

func TestParseAgentResponseParsesStructuredFields(t *testing.T) {
	decision := parseAgentResponse(`SIGNAL: SELL.
CONFIDENCE: 0.72
REASONING: bearish crossover with weak volume`, "test-agent", ProviderDeepSeek)

	if decision.Signal != SignalSell {
		t.Fatalf("expected SELL, got %s", decision.Signal)
	}
	if decision.Confidence != 0.72 {
		t.Fatalf("expected confidence 0.72, got %f", decision.Confidence)
	}
	if decision.Reasoning != "bearish crossover with weak volume" {
		t.Fatalf("unexpected reasoning: %q", decision.Reasoning)
	}
}

func TestConsensusMinConfidence(t *testing.T) {
	result := &ConsensusResult{
		FinalSignal:   SignalBuy,
		AvgConfidence: 0.64,
	}

	if !result.ShouldBuyWithMinConfidence(0.6) {
		t.Fatal("expected BUY to pass configured confidence threshold")
	}
	if result.ShouldBuyWithMinConfidence(0.7) {
		t.Fatal("expected BUY to fail higher configured confidence threshold")
	}
}

func TestConsensusAllowsExit(t *testing.T) {
	tests := []struct {
		name        string
		result      ConsensusResult
		exitSignal  Signal
		wantAllowed bool
	}{
		{
			name: "low-confidence opposite signal does not block profit exit",
			result: ConsensusResult{
				FinalSignal:   SignalBuy,
				AvgConfidence: 0.4,
			},
			exitSignal:  SignalSell,
			wantAllowed: true,
		},
		{
			name: "hold allows profit exit",
			result: ConsensusResult{
				FinalSignal:   SignalHold,
				AvgConfidence: 0.9,
			},
			exitSignal:  SignalSell,
			wantAllowed: true,
		},
		{
			name: "confident opposite signal blocks profit exit",
			result: ConsensusResult{
				FinalSignal:   SignalBuy,
				AvgConfidence: 0.8,
			},
			exitSignal:  SignalSell,
			wantAllowed: false,
		},
		{
			name: "matching signal allows profit exit",
			result: ConsensusResult{
				FinalSignal:   SignalSell,
				AvgConfidence: 0.8,
			},
			exitSignal:  SignalSell,
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.AllowsExit(tt.exitSignal, 0.5); got != tt.wantAllowed {
				t.Fatalf("AllowsExit() = %v, want %v", got, tt.wantAllowed)
			}
		})
	}
}
