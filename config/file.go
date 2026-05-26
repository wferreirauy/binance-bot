package config

import (
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	BaseURL          string `yaml:"base-url"`
	DataDir          string `yaml:"data-dir"`
	HistoricalPrices struct {
		Period   int    `yaml:"period"`
		Interval string `yaml:"interval"`
	} `yaml:"historical-prices"`
	OrderManagement struct {
		BuyTimeoutMinutes  int    `yaml:"buy-timeout-minutes"`
		SellTimeoutMinutes int    `yaml:"sell-timeout-minutes"`
		PartialFillAction  string `yaml:"partial-fill-action"`
		PollIntervalSecs   int    `yaml:"poll-interval-secs"`
	} `yaml:"order-management"`
	Fees struct {
		Enabled          bool    `yaml:"enabled"`
		DefaultTakerPct  float64 `yaml:"default-taker-pct"`
		BufferPct        float64 `yaml:"buffer-pct"`
		BuyBackBufferPct float64 `yaml:"buy-back-buffer-pct"`
	} `yaml:"fees"`
	Tendency struct {
		Interval       string `yaml:"interval"`
		Period         int    `yaml:"period"`           // frames fetched for trading-interval tendency (0 → historical-prices.period)
		FastLength     int    `yaml:"fast-length"`      // DEMA length for trading-interval tendency (0 → 9)
		SlowLength     int    `yaml:"slow-length"`      // EMA length for trading-interval tendency (0 → period)
		ConfirmBars    int    `yaml:"confirm-bars"`     // last N bars must agree (0/1 → single-bar, current behavior)
		HTFEnabled     bool   `yaml:"htf-enabled"`      // enable higher-timeframe trend gate
		HTFInterval    string `yaml:"htf-interval"`     // e.g. "5m", "15m" — blocks entry if HTF trend opposes trade direction
		HTFPeriod      int    `yaml:"htf-period"`       // frames fetched for HTF tendency (0 → tendency.period)
		HTFFastLength  int    `yaml:"htf-fast-length"`  // DEMA length for HTF tendency (0 → fast-length)
		HTFSlowLength  int    `yaml:"htf-slow-length"`  // EMA length for HTF tendency (0 → slow-length)
		HTFConfirmBars int    `yaml:"htf-confirm-bars"` // last N HTF bars must agree (0 → confirm-bars)
	} `yaml:"tendency"`
	Indicators struct {
		Rsi struct {
			Interval     string `yaml:"interval"`
			Length       int    `yaml:"length"`
			UpperLimit   int    `yaml:"upper-limit"`
			MiddleLimit  int    `yaml:"middle-limit"`
			LowerLimit   int    `yaml:"lower-limit"`
			SmoothLength int    `yaml:"smooth-length"` // optional EMA pre-smoothing of price before RSI (0/1 = off)
		} `yaml:"rsi"`
		Dema struct {
			Length int `yaml:"length"`
		} `yaml:"dema"`
		Macd struct {
			FastLength      int     `yaml:"fast-length"`
			SlowLength      int     `yaml:"slow-length"`
			SignalLength    int     `yaml:"signal-length"`
			ConsecutiveBars int     `yaml:"consecutive-bars"`        // require N consecutive bars of histogram in direction for scalp (default 1)
			MinSeparation   float64 `yaml:"min-separation"`          // when > 0, require histogram had |hist| >= this within the lookback (meaningful prior MACD/signal divergence) before now closing in
			MinSepLookback  int     `yaml:"min-separation-lookback"` // bars to scan for the prior peak separation (default 20 when min-separation > 0)
		} `yaml:"macd"`
		BollingerBands struct {
			Length     int     `yaml:"length"`
			Multiplier float64 `yaml:"multiplier"`
		} `yaml:"bollinger-bands"`
		Atr struct {
			Period int `yaml:"period"`
		} `yaml:"atr"`
		Adx struct {
			Period    int `yaml:"period"`
			Threshold int `yaml:"threshold"`
		} `yaml:"adx"`
		Volume struct {
			MaPeriod int `yaml:"ma-period"`
		} `yaml:"volume"`
	} `yaml:"indicators"`
	TrailingStop struct {
		Enabled       bool    `yaml:"enabled"`
		ActivationPct float64 `yaml:"activation-pct"`
		TrailingPct   float64 `yaml:"trailing-pct"`
	} `yaml:"trailing-stop"`
	AI struct {
		Enabled   bool `yaml:"enabled"`
		Providers struct {
			OpenAI struct {
				Model string `yaml:"model"`
			} `yaml:"openai"`
			DeepSeek struct {
				Model string `yaml:"model"`
			} `yaml:"deepseek"`
			Claude struct {
				Model string `yaml:"model"`
			} `yaml:"claude"`
		} `yaml:"providers"`
		MinConfidence float64 `yaml:"min-confidence"`
	} `yaml:"ai"`
	RefreshInterval int `yaml:"refresh-interval"`
	ScalpMode       struct {
		Enabled          bool    `yaml:"enabled"`
		MinScore         int     `yaml:"min-score"`          // min bullish signals to trigger entry (compared against weighted or unweighted total)
		PostBuyDelay     int     `yaml:"post-buy-delay"`     // seconds to wait after buy fill before sell monitoring
		InterOpDelay     int     `yaml:"inter-op-delay"`     // seconds to wait between operations
		RequireRSIExit   bool    `yaml:"require-rsi-exit"`   // require RSI declining for take-profit
		SLCooldown       bool    `yaml:"sl-cooldown"`        // enable exponential backoff after consecutive stop-losses
		MaxConsecutiveSL int     `yaml:"max-consecutive-sl"` // SL hits before cooldown kicks in (default: 2)
		CooldownBaseSecs int     `yaml:"cooldown-base-secs"` // base cooldown seconds, doubles each time (default: 60)
		ATRStopLoss      bool    `yaml:"atr-stop-loss"`      // use ATR-based dynamic stop-loss floor
		ATRMultiplier    float64 `yaml:"atr-multiplier"`     // SL = max(configured, atrMultiplier × ATR%) (default: 1.5)

		// --- v0.14 scalp improvements (all opt-in unless noted) ---
		WeightedScoring        bool    `yaml:"weighted-scoring"`         // use weighted signals (MACD=2, RSI=2, BB=2, Divergence=3, others=1)
		BBSqueezeEnabled       bool    `yaml:"bb-squeeze-enabled"`       // add a score point when bands are squeezed (width/avg < bb-squeeze-ratio)
		BBSqueezeRatio         float64 `yaml:"bb-squeeze-ratio"`         // squeeze threshold (default 0.6)
		BBSqueezeWindow        int     `yaml:"bb-squeeze-window"`        // avg-width lookback bars (default 20)
		VolumeStrongMultiplier float64 `yaml:"volume-strong-multiplier"` // required ratio of current/avg volume for vol signal (default 1.5)
		DivergenceEnabled      bool    `yaml:"divergence-enabled"`       // detect RSI bullish/bearish divergence and add high-weight score
		DivergenceLookback     int     `yaml:"divergence-lookback"`      // bars to search for swings (default 30)
		DivergenceSwingPad     int     `yaml:"divergence-swing-pad"`     // bars of confirmation on each side of swing (default 2)
		FastTrendGate          bool    `yaml:"fast-trend-gate"`          // require MACD line above zero for bull / below for bear (faster than DEMA tendency)
		TPATRMultiplier        float64 `yaml:"tp-atr-multiplier"`        // when > 0, take-profit % = (ATR/price * 100) × this (overrides --tp)
		SLATRMultiplier        float64 `yaml:"sl-atr-multiplier"`        // when > 0, stop-loss % = (ATR/price * 100) × this (overrides --sl; orthogonal to ATRStopLoss floor)
		TimeStopBars           int     `yaml:"time-stop-bars"`           // when > 0, exit at-or-above breakeven after N bars without TP
		BreakevenATRMult       float64 `yaml:"breakeven-atr-mult"`       // when > 0, raise SL to entry once price moves this many ATRs in profit
		MinATRPct              float64 `yaml:"min-atr-pct"`              // when > 0, skip entries while ATR% < this (avoid dead-flat regime)
		MaxATRPct              float64 `yaml:"max-atr-pct"`              // when > 0, skip entries while ATR% > this (avoid news-driven chaos)
		MACDPeakExit           bool    `yaml:"macd-peak-exit"`           // exit when MACD histogram peaks in profit, before TP/SL hit
		RecentExtremeBars      int     `yaml:"recent-extreme-bars"`      // when > 0, block entries against a fresh N-bar extreme
	} `yaml:"scalp-mode"`
	TopGainers struct {
		QuoteAsset     string   `yaml:"quote-asset"`
		Limit          int      `yaml:"limit"`
		PollInterval   int      `yaml:"poll-interval"`
		MinVolume      float64  `yaml:"min-volume"`
		ExcludeSymbols []string `yaml:"exclude-symbols"`
	} `yaml:"top-gainers"`
	Rotation struct {
		BridgeAsset       string   `yaml:"bridge-asset"`
		CurrentAsset      string   `yaml:"current-asset"`
		SupportedAssets   []string `yaml:"supported-assets"`
		ScoutMultiplier   float64  `yaml:"scout-multiplier"`
		ScoutMarginPct    float64  `yaml:"scout-margin-pct"`
		UseMargin         bool     `yaml:"use-margin"`
		ScoutSleepSeconds int      `yaml:"scout-sleep-seconds"`
		DryRun            bool     `yaml:"dry-run"`
		MaxJumps          int      `yaml:"max-jumps"`
		MinNotionalBuffer float64  `yaml:"min-notional-buffer"`
	} `yaml:"rotation"`
	Backtest struct {
		InitialBalance float64 `yaml:"initial-balance"`
		FeePct         float64 `yaml:"fee-pct"`
	} `yaml:"backtest"`
	API struct {
		Address string `yaml:"address"`
	} `yaml:"api"`
}

func (c *Config) Read(filePath string) (*Config, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("Config: could not open config file: %w", err)
	}
	defer f.Close()
	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("Config: could not decode the config file: %w", err)
	}
	return &cfg, nil
}
