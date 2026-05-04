package config

import (
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	BaseURL          string `yaml:"base-url"`
	HistoricalPrices struct {
		Period   int    `yaml:"period"`
		Interval string `yaml:"interval"`
	} `yaml:"historical-prices"`
	Tendency struct {
		Interval    string `yaml:"interval"`
		Direction   string `yaml:"direction"`
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
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("Config: invalid configuration: %w", err)
	}
	return &cfg, nil
}

// applyDefaults fills in safe values for keys that were omitted from the YAML
// so that downstream code never sees a zero-valued indicator period (which
// would silently disable gates such as ADX or volume confirmation).
func (c *Config) applyDefaults() {
	if c.HistoricalPrices.Period == 0 {
		c.HistoricalPrices.Period = 100
	}
	if c.HistoricalPrices.Interval == "" {
		c.HistoricalPrices.Interval = "1m"
	}
	if c.Tendency.Interval == "" {
		c.Tendency.Interval = "3m"
	}
	if c.Tendency.Direction == "" {
		c.Tendency.Direction = "up"
	}
	if c.Indicators.Rsi.Length == 0 {
		c.Indicators.Rsi.Length = 14
	}
	if c.Indicators.Rsi.UpperLimit == 0 {
		c.Indicators.Rsi.UpperLimit = 70
	}
	if c.Indicators.Rsi.LowerLimit == 0 {
		c.Indicators.Rsi.LowerLimit = 30
	}
	if c.Indicators.Rsi.Interval == "" {
		c.Indicators.Rsi.Interval = c.HistoricalPrices.Interval
	}
	if c.Indicators.Dema.Length == 0 {
		c.Indicators.Dema.Length = 9
	}
	if c.Indicators.Macd.FastLength == 0 {
		c.Indicators.Macd.FastLength = 12
	}
	if c.Indicators.Macd.SlowLength == 0 {
		c.Indicators.Macd.SlowLength = 26
	}
	if c.Indicators.Macd.SignalLength == 0 {
		c.Indicators.Macd.SignalLength = 9
	}
	if c.Indicators.BollingerBands.Length == 0 {
		c.Indicators.BollingerBands.Length = 20
	}
	if c.Indicators.BollingerBands.Multiplier == 0 {
		c.Indicators.BollingerBands.Multiplier = 2.0
	}
	if c.RefreshInterval == 0 {
		c.RefreshInterval = 10
	}
}

// validate rejects configurations that would produce undefined trading
// behaviour (negative periods, RSI bounds inverted, etc.). It is intentionally
// strict because misconfiguration here can manifest as silently disabled
// guard-rails rather than visible errors.
func (c *Config) validate() error {
	if c.HistoricalPrices.Period < 0 {
		return fmt.Errorf("historical-prices.period must be >= 0")
	}
	if c.Indicators.Rsi.Length < 0 || c.Indicators.Macd.FastLength < 0 ||
		c.Indicators.Macd.SlowLength < 0 || c.Indicators.Macd.SignalLength < 0 ||
		c.Indicators.BollingerBands.Length < 0 ||
		c.Indicators.Atr.Period < 0 || c.Indicators.Adx.Period < 0 ||
		c.Indicators.Volume.MaPeriod < 0 || c.Indicators.Dema.Length < 0 {
		return fmt.Errorf("indicator periods must be >= 0")
	}
	if c.Indicators.Macd.SlowLength > 0 && c.Indicators.Macd.FastLength > 0 &&
		c.Indicators.Macd.SlowLength <= c.Indicators.Macd.FastLength {
		return fmt.Errorf("indicators.macd.slow-length must be > fast-length")
	}
	if c.Indicators.Rsi.UpperLimit > 0 && c.Indicators.Rsi.LowerLimit > 0 &&
		c.Indicators.Rsi.UpperLimit <= c.Indicators.Rsi.LowerLimit {
		return fmt.Errorf("indicators.rsi.upper-limit must be > lower-limit")
	}
	if c.Tendency.Direction != "" && c.Tendency.Direction != "up" && c.Tendency.Direction != "down" {
		return fmt.Errorf("tendency.direction must be 'up' or 'down'")
	}
	if c.RefreshInterval < 0 {
		return fmt.Errorf("refresh-interval must be >= 0")
	}
	if c.AI.MinConfidence < 0 || c.AI.MinConfidence > 1 {
		return fmt.Errorf("ai.min-confidence must be in [0, 1]")
	}
	if c.TrailingStop.Enabled {
		if c.TrailingStop.ActivationPct < 0 || c.TrailingStop.TrailingPct < 0 {
			return fmt.Errorf("trailing-stop pct values must be >= 0")
		}
	}
	return nil
}
