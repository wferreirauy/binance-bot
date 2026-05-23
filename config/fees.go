package config

// DefaultBuyBackBufferPct is the safety buffer (in percent) used to scale
// down buy-back / round-trip order quantities so the wallet's remaining
// balance — after Binance trading fees — is enough to cover the order.
//
// 0.2% covers a 0.1% taker commission per leg plus a small margin for
// price drift between the price-quote and the order being accepted.
const DefaultBuyBackBufferPct = 0.2

// BuyBackBuffer returns the configured fees.buy-back-buffer-pct or the
// default (0.2%) when the user hasn't set it (zero value). Negative
// values are clamped to 0 by Validate.
func (c *Config) BuyBackBuffer() float64 {
	if c.Fees.BuyBackBufferPct <= 0 {
		return DefaultBuyBackBufferPct
	}
	return c.Fees.BuyBackBufferPct
}
