# Configuration Reference

This document describes every parameter available in the YAML configuration file used by the binance-bot.

## Table of Contents

- [Base URL](#base-url)
- [Historical Prices](#historical-prices)
- [Refresh Interval](#refresh-interval)
- [Tendency](#tendency)
- [Indicators](#indicators)
  - [RSI](#rsi)
  - [DEMA](#dema)
  - [MACD](#macd)
  - [Bollinger Bands](#bollinger-bands)
  - [ATR](#atr)
  - [ADX](#adx)
  - [Volume](#volume)
- [Trailing Stop](#trailing-stop)
- [Scalp Mode](#scalp-mode)
- [AI Agents](#ai-agents)
- [Top Gainers](#top-gainers)
- [Recommended Configurations](#recommended-configurations)
  - [Scalping (High-Frequency)](#scalping-high-frequency)
  - [Mid-Term Trading (Swing)](#mid-term-trading-swing)
  - [Long-Term Trading (Position)](#long-term-trading-position)

---

## Base URL

Overrides the Binance API base URL. Useful for switching between production and testnet environments.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `base-url` | string | `"https://api1.binance.com"` | Binance API base URL. If omitted or empty, the default production URL is used. |

| Environment | URL |
|-------------|-----|
| Production (default) | `https://api1.binance.com` |
| Testnet | `https://testnet.binance.vision` |

```yaml
base-url: "https://testnet.binance.vision"
```

---

## Historical Prices

Controls the OHLCV data fetched for indicator calculations.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `period` | int | - | Number of candles to fetch for moving average and indicator calculations. Higher values give more historical context but slower response to changes. |
| `interval` | string | - | Candlestick interval for price data. Valid values: `"1m"`, `"3m"`, `"5m"`, `"15m"`, `"30m"`, `"1h"`, `"4h"`, `"1d"`. |

```yaml
historical-prices:
  period: 100
  interval: "1m"
```

---

## Refresh Interval

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `refresh-interval` | int | 10 | Seconds between each price poll and indicator recalculation. Lower values mean faster reaction but more API calls. |

```yaml
refresh-interval: 10
```

---

## Tendency

Controls market direction detection and the higher-timeframe trend gate.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `interval` | string | - | Candlestick interval used to determine the current market tendency (e.g., `"3m"`, `"5m"`). |
| `htf-enabled` | bool | false | When `true`, enables the Higher-Timeframe (HTF) trend gate that blocks entries when the longer timeframe opposes the trade direction. |
| `htf-interval` | string | - | The interval for the HTF tendency check (e.g., `"15m"`, `"1h"`). Should be higher than the trading `interval`. |

```yaml
tendency:
  interval: "3m"
  htf-enabled: false
  htf-interval: "15m"
```

---

## Indicators

### RSI

Relative Strength Index — measures momentum to detect overbought/oversold conditions.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `interval` | string | - | Candlestick interval for RSI price data (can differ from the main trading interval). |
| `length` | int | - | Number of periods for RSI calculation. Standard is 14; lower values are more reactive. |
| `upper-limit` | int | - | RSI above this value is considered overbought. In bull mode, entry is blocked above this. |
| `middle-limit` | int | - | Neutral RSI level (informational). |
| `lower-limit` | int | - | RSI below this value is considered oversold. In bear mode, entry is blocked below this. |

```yaml
indicators:
  rsi:
    interval: "5m"
    length: 14
    upper-limit: 70
    middle-limit: 50
    lower-limit: 30
```

### DEMA

Double Exponential Moving Average — a faster-responding trend indicator than standard EMA.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `length` | int | - | Number of periods for DEMA calculation. Lower values track price more closely. |

```yaml
  dema:
    length: 9
```

### MACD

Moving Average Convergence Divergence — detects trend changes via crossovers.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `fast-length` | int | - | Period for the fast EMA. Standard: 12. |
| `slow-length` | int | - | Period for the slow EMA. Standard: 26. |
| `signal-length` | int | - | Period for the signal line EMA. Standard: 9. |

Entry is triggered on MACD crossovers: bullish (MACD crosses above signal) for buy, bearish (MACD crosses below signal) for sell.

```yaml
  macd:
    fast-length: 12
    slow-length: 26
    signal-length: 9
```

### Bollinger Bands

Volatility bands around a moving average — used to determine relative price position.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `length` | int | - | Period for the middle band (SMA). Standard: 20. |
| `multiplier` | float | - | Standard deviation multiplier for band width. Standard: 2.0. Lower values produce tighter bands. |

Entry logic checks whether price is closer to the lower band (bull) or upper band (bear).

```yaml
  bollinger-bands:
    length: 20
    multiplier: 2.0
```

### ATR

Average True Range — measures market volatility. Used for dynamic stop-loss calculations.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `period` | int | - | Number of periods for ATR calculation. Standard: 14. |

```yaml
  atr:
    period: 14
```

### ADX

Average Directional Index — measures trend strength regardless of direction.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `period` | int | - | Number of periods for ADX calculation. Standard: 14. |
| `threshold` | int | - | Minimum ADX value required to enter a trade. Higher values require stronger trends. Typical: 20-25. |

```yaml
  adx:
    period: 14
    threshold: 25
```

### Volume

Volume-based confirmation — ensures trades occur during active market participation.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `ma-period` | int | - | Period for the volume moving average. Current volume must exceed this average for entry confirmation. |

```yaml
  volume:
    ma-period: 20
```

---

## Trailing Stop

A dynamic exit mechanism that locks in profits by following price movement.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | - | Enable or disable trailing stop. |
| `activation-pct` | float | - | Percentage profit required before the trailing stop activates. E.g., `1.5` means trail starts after 1.5% unrealized profit. |
| `trailing-pct` | float | - | Percentage from peak (bull) or trough (bear) that triggers the exit. E.g., `1.0` means sell if price drops 1% from highest point after activation. |

```yaml
trailing-stop:
  enabled: true
  activation-pct: 1.5
  trailing-pct: 1.0
```

---

## Scalp Mode

Relaxed entry conditions for high-frequency micro-trading. Uses a scoring system instead of requiring all entry conditions simultaneously.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | false | Enable scalp mode scoring system. |
| `min-score` | int | 3 | Minimum bullish/bearish signals (out of 6) required to trigger entry. The 6 signals are: RSI, MACD, tendency, Bollinger position, ADX strength, and volume confirmation. |
| `post-buy-delay` | int | 30 | Seconds to wait after order fill before starting exit monitoring. Prevents immediate stop-loss on volatile fills. |
| `inter-op-delay` | int | 60 | Seconds to wait between completed operations before starting the next one. |
| `require-rsi-exit` | bool | true | When `true`, take-profit requires RSI to be declining (bull) or rising (bear) in addition to hitting the TP target. Set `false` for faster exits. |
| `sl-cooldown` | bool | false | Enable exponential backoff after consecutive stop-losses to avoid overtrading in choppy markets. |
| `max-consecutive-sl` | int | 2 | Number of consecutive stop-losses before cooldown kicks in. |
| `cooldown-base-secs` | int | 60 | Base cooldown in seconds. Doubles with each additional consecutive SL. Capped at 600s. |
| `atr-stop-loss` | bool | false | Dynamically widen stop-loss based on current ATR (volatility). Uses `max(configured_SL, atr-multiplier * ATR%)`. |
| `atr-multiplier` | float | 1.5 | Multiplier applied to ATR percentage for dynamic stop-loss floor. |

```yaml
scalp-mode:
  enabled: false
  min-score: 3
  post-buy-delay: 30
  inter-op-delay: 60
  require-rsi-exit: true
  sl-cooldown: false
  max-consecutive-sl: 2
  cooldown-base-secs: 60
  atr-stop-loss: false
  atr-multiplier: 1.5
```

---

## AI Agents

Multi-agent AI consensus system. Three AI providers analyze technical data and vote on trade signals.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | false | Enable AI-assisted trading decisions. Requires at least one provider API key set via environment variables. |
| `providers.openai.model` | string | - | OpenAI model to use. Requires `OPENAI_API_KEY` env var. |
| `providers.deepseek.model` | string | - | DeepSeek model to use. Requires `DEEPSEEK_API_KEY` env var. |
| `providers.claude.model` | string | - | Anthropic Claude model to use. Requires `ANTHROPIC_API_KEY` env var. |
| `min-confidence` | float | 0.5 | Minimum average confidence (0.0-1.0) from AI agents to act on their signal. |

```yaml
ai:
  enabled: true
  providers:
    openai:
      model: "gpt-4o-mini"
    deepseek:
      model: "deepseek-chat"
    claude:
      model: "claude-3-5-haiku-20241022"
  min-confidence: 0.5
```

---

## Top Gainers

Configuration for the `top-gainers` command that monitors top 24h price-change symbols.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `quote-asset` | string | - | Filter pairs ending with this asset (e.g., `"USDT"`, `"BTC"`). |
| `limit` | int | - | Number of top gainers to display. |
| `poll-interval` | int | - | Seconds between each data refresh. |
| `min-volume` | float | - | Minimum 24h quote volume to include a symbol. Filters out illiquid pairs. |
| `exclude-symbols` | []string | - | List of symbols to exclude from results (e.g., stablecoins). |

```yaml
top-gainers:
  quote-asset: "USDT"
  limit: 20
  poll-interval: 60
  min-volume: 1000000
  exclude-symbols:
    - "USDCUSDT"
```

---

## Recommended Configurations

### Scalping (High-Frequency)

For multiple micro-transactions per day on volatile pairs (e.g., PEPE/USDT, SHIB/USDT, DOGE/USDT).

**Goals:** Fast entries and exits, many small profits, tight risk management.

**CLI example:**
```bash
binance-bot bt -t PEPE/USDT -a 50 --sl 1.0 --tp 0.5 -b 0.9999 -s 1.0001 -rp 8 -ra 0 -o 500 -f scalp-config.yml
```

```yaml
historical-prices:
  period: 50
  interval: "1m"

refresh-interval: 10

tendency:
  interval: "1m"
  htf-enabled: true
  htf-interval: "5m"

indicators:
  rsi:
    interval: "1m"
    length: 7
    upper-limit: 65
    middle-limit: 50
    lower-limit: 35
  dema:
    length: 5
  macd:
    fast-length: 6
    slow-length: 13
    signal-length: 5
  bollinger-bands:
    length: 10
    multiplier: 1.5
  atr:
    period: 7
  adx:
    period: 7
    threshold: 15
  volume:
    ma-period: 10

trailing-stop:
  enabled: true
  activation-pct: 0.5
  trailing-pct: 0.3

scalp-mode:
  enabled: true
  min-score: 3
  post-buy-delay: 5
  inter-op-delay: 10
  require-rsi-exit: false
  sl-cooldown: true
  max-consecutive-sl: 2
  cooldown-base-secs: 60
  atr-stop-loss: true
  atr-multiplier: 1.5

ai:
  enabled: true
  providers:
    openai:
      model: "gpt-4o-mini"
    deepseek:
      model: "deepseek-chat"
    claude:
      model: "claude-3-5-haiku-20241022"
  min-confidence: 0.3
```

**Key characteristics:**
- Short indicator periods (7-13) for fast signals
- Relaxed entry via scoring (`min-score: 3` of 6)
- Tight trailing stop (0.3% from peak after 0.5% profit)
- Low ADX threshold (15) — accepts weaker trends
- HTF gate on 5m to avoid counter-trend entries
- ATR-based dynamic SL to adapt to volatility spikes
- Low AI confidence threshold (0.3) — AI is advisory, not blocking

---

### Mid-Term Trading (Swing)

For trades held minutes to hours on moderate-volatility pairs (e.g., BTC/USDT, ETH/USDT, SOL/USDT).

**Goals:** Capture larger moves, fewer trades per day, balanced risk/reward.

**CLI example:**
```bash
binance-bot at -t BTC/USDT -a 0.001 --sl 3.0 --tp 2.0 -b 0.9995 -s 1.0005 -rp 2 -ra 5 -o 20 --strategy auto -f swing-config.yml
```

```yaml
historical-prices:
  period: 100
  interval: "5m"

refresh-interval: 30

tendency:
  interval: "15m"
  htf-enabled: true
  htf-interval: "1h"

indicators:
  rsi:
    interval: "15m"
    length: 14
    upper-limit: 70
    middle-limit: 50
    lower-limit: 30
  dema:
    length: 9
  macd:
    fast-length: 12
    slow-length: 26
    signal-length: 9
  bollinger-bands:
    length: 20
    multiplier: 2.0
  atr:
    period: 14
  adx:
    period: 14
    threshold: 25
  volume:
    ma-period: 20

trailing-stop:
  enabled: true
  activation-pct: 1.5
  trailing-pct: 1.0

scalp-mode:
  enabled: false

ai:
  enabled: true
  providers:
    openai:
      model: "gpt-4o-mini"
    deepseek:
      model: "deepseek-chat"
    claude:
      model: "claude-3-5-haiku-20241022"
  min-confidence: 0.5
```

**Key characteristics:**
- Standard indicator periods (14, 26) — proven defaults
- Strict entry requiring ALL conditions (scalp mode off)
- 15m tendency with 1h HTF gate — filters noise
- Trailing stop activates after 1.5% profit, trails 1% from peak
- Higher AI confidence threshold (0.5) — AI must be more certain
- 30s refresh interval — less aggressive polling

---

### Long-Term Trading (Position)

For trades held hours to days on high-cap assets (e.g., BTC/USDT, ETH/USDT).

**Goals:** Capture major trend moves, very few trades, wide stops to avoid noise.

**CLI example:**
```bash
binance-bot at -t BTC/USDT -a 0.01 --sl 5.0 --tp 4.0 -b 0.999 -s 1.001 -rp 2 -ra 5 -o 5 --strategy auto -f position-config.yml
```

```yaml
historical-prices:
  period: 200
  interval: "15m"

refresh-interval: 60

tendency:
  interval: "1h"
  htf-enabled: true
  htf-interval: "4h"

indicators:
  rsi:
    interval: "1h"
    length: 14
    upper-limit: 75
    middle-limit: 50
    lower-limit: 25
  dema:
    length: 14
  macd:
    fast-length: 12
    slow-length: 26
    signal-length: 9
  bollinger-bands:
    length: 20
    multiplier: 2.5
  atr:
    period: 14
  adx:
    period: 14
    threshold: 30
  volume:
    ma-period: 30

trailing-stop:
  enabled: true
  activation-pct: 3.0
  trailing-pct: 2.0

scalp-mode:
  enabled: false

ai:
  enabled: true
  providers:
    openai:
      model: "gpt-4o-mini"
    deepseek:
      model: "deepseek-chat"
    claude:
      model: "claude-3-5-haiku-20241022"
  min-confidence: 0.6
```

**Key characteristics:**
- Long indicator periods and high timeframes (1h tendency, 4h HTF gate)
- Wide Bollinger bands (2.5x multiplier) — only signals extremes
- High ADX threshold (30) — only enters strong, confirmed trends
- Wide trailing stop (3% activation, 2% trail) — gives room to breathe
- Extended RSI limits (25/75) — only trades at clear overbought/oversold
- 60s refresh — minimal API load
- Highest AI confidence (0.6) — conservative decision making
- Long historical lookback (200 candles) — more data for indicator accuracy
