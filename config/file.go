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
		Enabled         bool    `yaml:"enabled"`
		DefaultTakerPct float64 `yaml:"default-taker-pct"`
		BufferPct       float64 `yaml:"buffer-pct"`
	} `yaml:"fees"`
	Tendency struct {
		Interval    string `yaml:"interval"`
		HTFEnabled  bool   `yaml:"htf-enabled"`  // enable higher-timeframe trend gate
		HTFInterval string `yaml:"htf-interval"` // e.g. "5m", "15m" — blocks entry if HTF trend opposes trade direction
	} `yaml:"tendency"`
	Indicators struct {
		Rsi struct {
			Interval    string `yaml:"interval"`
			Length      int    `yaml:"length"`
			UpperLimit  int    `yaml:"upper-limit"`
			MiddleLimit int    `yaml:"middle-limit"`
			LowerLimit  int    `yaml:"lower-limit"`
		} `yaml:"rsi"`
		Dema struct {
			Length int `yaml:"length"`
		} `yaml:"dema"`
		Macd struct {
			FastLength   int `yaml:"fast-length"`
			SlowLength   int `yaml:"slow-length"`
			SignalLength int `yaml:"signal-length"`
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
		MinScore         int     `yaml:"min-score"`          // min bullish signals out of 6 to trigger entry
		PostBuyDelay     int     `yaml:"post-buy-delay"`     // seconds to wait after buy fill before sell monitoring
		InterOpDelay     int     `yaml:"inter-op-delay"`     // seconds to wait between operations
		RequireRSIExit   bool    `yaml:"require-rsi-exit"`   // require RSI declining for take-profit
		SLCooldown       bool    `yaml:"sl-cooldown"`        // enable exponential backoff after consecutive stop-losses
		MaxConsecutiveSL int     `yaml:"max-consecutive-sl"` // SL hits before cooldown kicks in (default: 2)
		CooldownBaseSecs int     `yaml:"cooldown-base-secs"` // base cooldown seconds, doubles each time (default: 60)
		ATRStopLoss      bool    `yaml:"atr-stop-loss"`      // use ATR-based dynamic stop-loss floor
		ATRMultiplier    float64 `yaml:"atr-multiplier"`     // SL = max(configured, atrMultiplier × ATR%) (default: 1.5)
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
