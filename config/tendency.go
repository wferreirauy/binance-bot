package config

// TendencyParams holds the resolved parameters for a single tendency evaluation.
type TendencyParams struct {
	Interval    string
	Frames      int
	FastLength  int
	SlowLength  int
	ConfirmBars int
}

// TradingTendencyParams returns the parameters used for trading-interval tendency
// detection, applying backward-compatible defaults when fields are omitted.
func (c *Config) TradingTendencyParams() TendencyParams {
	frames := c.Tendency.Period
	if frames <= 0 {
		frames = c.HistoricalPrices.Period
	}
	fast := c.Tendency.FastLength
	if fast <= 0 {
		fast = 9
	}
	slow := c.Tendency.SlowLength
	if slow <= 0 {
		slow = frames
	}
	confirm := c.Tendency.ConfirmBars
	if confirm <= 0 {
		confirm = 1
	}
	return TendencyParams{
		Interval:    c.Tendency.Interval,
		Frames:      frames,
		FastLength:  fast,
		SlowLength:  slow,
		ConfirmBars: confirm,
	}
}

// HTFTendencyParams returns the parameters used for the higher-timeframe trend gate,
// falling back to the trading-interval params when HTF-specific fields are omitted.
func (c *Config) HTFTendencyParams() TendencyParams {
	base := c.TradingTendencyParams()
	p := TendencyParams{
		Interval:    c.Tendency.HTFInterval,
		Frames:      base.Frames,
		FastLength:  base.FastLength,
		SlowLength:  base.SlowLength,
		ConfirmBars: base.ConfirmBars,
	}
	if c.Tendency.HTFPeriod > 0 {
		p.Frames = c.Tendency.HTFPeriod
	}
	if c.Tendency.HTFFastLength > 0 {
		p.FastLength = c.Tendency.HTFFastLength
	}
	if c.Tendency.HTFSlowLength > 0 {
		p.SlowLength = c.Tendency.HTFSlowLength
	}
	if c.Tendency.HTFConfirmBars > 0 {
		p.ConfirmBars = c.Tendency.HTFConfirmBars
	}
	return p
}
