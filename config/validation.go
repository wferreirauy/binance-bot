package config

import (
	"fmt"
	"net/url"
	"regexp"
)

var binanceIntervalPattern = regexp.MustCompile(`^(1s|1m|3m|5m|15m|30m|1h|2h|4h|6h|8h|12h|1d|3d|1w|1M)$`)

// Validate returns every configuration issue found, so users can fix them in one pass.
func (c *Config) Validate() []error {
	var errs []error

	if c.BaseURL != "" {
		if parsed, err := url.ParseRequestURI(c.BaseURL); err != nil || parsed.Scheme == "" || parsed.Host == "" {
			errs = append(errs, fmt.Errorf("base-url must be a valid absolute URL"))
		}
	}

	errs = appendPositiveInt(errs, "historical-prices.period", c.HistoricalPrices.Period)
	errs = appendInterval(errs, "historical-prices.interval", c.HistoricalPrices.Interval)
	errs = appendPositiveInt(errs, "refresh-interval", c.RefreshInterval)

	errs = appendInterval(errs, "tendency.interval", c.Tendency.Interval)
	if c.Tendency.Direction != "up" && c.Tendency.Direction != "down" {
		errs = append(errs, fmt.Errorf("tendency.direction must be either \"up\" or \"down\""))
	}
	if c.Tendency.HTFEnabled {
		errs = appendInterval(errs, "tendency.htf-interval", c.Tendency.HTFInterval)
	}

	errs = appendInterval(errs, "indicators.rsi.interval", c.Indicators.Rsi.Interval)
	errs = appendPositiveInt(errs, "indicators.rsi.length", c.Indicators.Rsi.Length)
	if !between(c.Indicators.Rsi.LowerLimit, 0, 100) {
		errs = append(errs, fmt.Errorf("indicators.rsi.lower-limit must be between 0 and 100"))
	}
	if !between(c.Indicators.Rsi.MiddleLimit, 0, 100) {
		errs = append(errs, fmt.Errorf("indicators.rsi.middle-limit must be between 0 and 100"))
	}
	if !between(c.Indicators.Rsi.UpperLimit, 0, 100) {
		errs = append(errs, fmt.Errorf("indicators.rsi.upper-limit must be between 0 and 100"))
	}
	if !(c.Indicators.Rsi.LowerLimit < c.Indicators.Rsi.MiddleLimit && c.Indicators.Rsi.MiddleLimit < c.Indicators.Rsi.UpperLimit) {
		errs = append(errs, fmt.Errorf("indicators.rsi limits must satisfy lower-limit < middle-limit < upper-limit"))
	}

	errs = appendPositiveInt(errs, "indicators.dema.length", c.Indicators.Dema.Length)
	errs = appendPositiveInt(errs, "indicators.macd.fast-length", c.Indicators.Macd.FastLength)
	errs = appendPositiveInt(errs, "indicators.macd.slow-length", c.Indicators.Macd.SlowLength)
	errs = appendPositiveInt(errs, "indicators.macd.signal-length", c.Indicators.Macd.SignalLength)
	if c.Indicators.Macd.FastLength >= c.Indicators.Macd.SlowLength {
		errs = append(errs, fmt.Errorf("indicators.macd.fast-length must be less than slow-length"))
	}
	errs = appendPositiveInt(errs, "indicators.bollinger-bands.length", c.Indicators.BollingerBands.Length)
	if c.Indicators.BollingerBands.Multiplier <= 0 {
		errs = append(errs, fmt.Errorf("indicators.bollinger-bands.multiplier must be greater than 0"))
	}
	errs = appendPositiveInt(errs, "indicators.atr.period", c.Indicators.Atr.Period)
	errs = appendPositiveInt(errs, "indicators.adx.period", c.Indicators.Adx.Period)
	if c.Indicators.Adx.Threshold < 0 {
		errs = append(errs, fmt.Errorf("indicators.adx.threshold must be greater than or equal to 0"))
	}
	errs = appendPositiveInt(errs, "indicators.volume.ma-period", c.Indicators.Volume.MaPeriod)

	if c.TrailingStop.Enabled {
		if c.TrailingStop.ActivationPct <= 0 {
			errs = append(errs, fmt.Errorf("trailing-stop.activation-pct must be greater than 0 when trailing stop is enabled"))
		}
		if c.TrailingStop.TrailingPct <= 0 {
			errs = append(errs, fmt.Errorf("trailing-stop.trailing-pct must be greater than 0 when trailing stop is enabled"))
		}
	}

	if c.ScalpMode.Enabled {
		if !between(c.ScalpMode.MinScore, 1, 6) {
			errs = append(errs, fmt.Errorf("scalp-mode.min-score must be between 1 and 6 when scalp mode is enabled"))
		}
		errs = appendPositiveInt(errs, "scalp-mode.post-buy-delay", c.ScalpMode.PostBuyDelay)
		errs = appendPositiveInt(errs, "scalp-mode.inter-op-delay", c.ScalpMode.InterOpDelay)
		if c.ScalpMode.SLCooldown {
			errs = appendPositiveInt(errs, "scalp-mode.max-consecutive-sl", c.ScalpMode.MaxConsecutiveSL)
			errs = appendPositiveInt(errs, "scalp-mode.cooldown-base-secs", c.ScalpMode.CooldownBaseSecs)
		}
		if c.ScalpMode.ATRStopLoss && c.ScalpMode.ATRMultiplier <= 0 {
			errs = append(errs, fmt.Errorf("scalp-mode.atr-multiplier must be greater than 0 when ATR stop-loss is enabled"))
		}
	}

	if c.AI.Enabled && (c.AI.MinConfidence < 0 || c.AI.MinConfidence > 1) {
		errs = append(errs, fmt.Errorf("ai.min-confidence must be between 0 and 1"))
	}

	if c.TopGainers.QuoteAsset == "" {
		errs = append(errs, fmt.Errorf("top-gainers.quote-asset must not be empty"))
	}
	errs = appendPositiveInt(errs, "top-gainers.limit", c.TopGainers.Limit)
	errs = appendPositiveInt(errs, "top-gainers.poll-interval", c.TopGainers.PollInterval)
	if c.TopGainers.MinVolume < 0 {
		errs = append(errs, fmt.Errorf("top-gainers.min-volume must be greater than or equal to 0"))
	}

	return errs
}

func appendPositiveInt(errs []error, field string, value int) []error {
	if value <= 0 {
		return append(errs, fmt.Errorf("%s must be greater than 0", field))
	}
	return errs
}

func appendInterval(errs []error, field, value string) []error {
	if !binanceIntervalPattern.MatchString(value) {
		return append(errs, fmt.Errorf("%s must be a valid Binance interval", field))
	}
	return errs
}

func between(value, min, max int) bool {
	return value >= min && value <= max
}
