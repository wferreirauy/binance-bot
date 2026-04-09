# Binance Trade Bot

<img width="1343" height="863" alt="Image" src="https://github.com/user-attachments/assets/10b5cd6f-c80e-4661-b963-9f5b52641330" />


## Features

- **Auto Trade** — Automatically detects market tendency and switches between bull/bear strategies per operation; supports forced strategy mode to wait for a matching tendency
- **Bull Trade** — Buy-low-sell-high strategy for uptrending markets
- **Bear Trade** — Sell-high-buy-low strategy for downtrending markets
- **Scalp Mode** — High-frequency micro-trading using a scoring-based entry system; no longer requires all signals simultaneously
- **Top Gainers Monitor** — Real-time TUI dashboard of the top 24h movers on Binance
- **AI Multi-Agent System** — Concurrent analysis from OpenAI, DeepSeek, and Claude with weighted consensus
- **Sentiment Analysis** — Real-time news headlines and Fear & Greed Index integrated into AI decisions
- **Trailing Stop-Loss** — Dynamically locks in profits as price moves favorably
- **Advanced Indicators** — RSI, MACD, DEMA, Bollinger Bands, ADX, ATR, and volume confirmation
- **Full OHLCV Analysis** — Uses complete candlestick data instead of close-only prices
- **Auto-Notional Adjustment** — Automatically raises order quantity to meet Binance's minimum notional filter
- **File Logging** — All trade events and errors are written to `binance-bot.log` alongside the TUI display

## Download

#### **Download Precompiled Binary**

You can **download the precompiled binary** from the repository's release artifacts.

1. Visit the [Releases](https://github.com/wferreirauy/binance-bot/releases) page of the repository.
2. Download the appropriate binary for your operating system (e.g., Linux, macOS, Windows).
3. Make the binary executable (if required):
   - On Linux or macOS:
     ```bash
     chmod +x binance-bot
     ```
4. Move the binary to a directory in your `$PATH` for global access:
   - On Linux
   ```bash
   sudo mv binance-bot /usr/local/bin/

## Usage

⚠️ **Warning:** This bot is provided as-is. Use it at your own risk. Trading involves financial risks, and you may incur significant losses. Always test in a safe environment (e.g., a testnet and/or with small amounts) before deploying in live markets. The author is not responsible for any financial outcomes.

---

### Prerequisites

Before using the Binance Trade Bot, you need to configure your environment with the Binance API client credentials. These credentials allow the bot to interact securely with your Binance account. Follow these steps to set up:

1. **Obtain your Binance API Key and Secret**
   - Log in to your [Binance account](https://www.binance.com/).
   - Navigate to the [API Management section](https://www.binance.com/en/my/settings/api-management).
   - Create a new API key, choosing HMAC type and providing any label (e.g., `CLI_Bot`).
   - Save the **API Key** and **Secret Key** securely. You will not be able to view the secret again after closing the page.

2. **Set Environment Variables**
   Export the API credentials as environment variables in your terminal before executing the binance-bot cli:

   ```bash
   export BINANCE_API_KEY=<your-api-key>
   export BINANCE_SECRET_KEY=<your-secret-key>
   ```

3. **Set AI Provider API Keys (optional)**
   To enable the AI multi-agent system, export one or more of the following API keys. The system works with any combination — you can use 1, 2, or all 3 providers:

   ```bash
   export OPENAI_API_KEY=<your-openai-api-key>
   export DEEPSEEK_API_KEY=<your-deepseek-api-key>
   export ANTHROPIC_API_KEY=<your-anthropic-api-key>
   ```

   | Variable | Provider | Default Model |
   |----------|----------|---------------|
   | `OPENAI_API_KEY` | OpenAI | `gpt-4o-mini` |
   | `DEEPSEEK_API_KEY` | DeepSeek | `deepseek-chat` |
   | `ANTHROPIC_API_KEY` | Claude | `claude-3-5-haiku-20241022` |

   > If no AI keys are set, the bot runs entirely on technical indicators — AI is fully optional.

4. **Create a config file**
   You can specify a custom configuration file to adjust the bot's parameters of trading indicators. <br />
   See the [sample configuration file](/sample-binance-config.yml).

#### Now you're ready to use the Binance Trade Bot! 🎉

### Run the Bot

#### Auto Trade (automatic tendency detection)

```bash
binance-bot -f binance-config.yml auto-trade -t "BTC/USDT" -a 0.001 -sl 2.0 -tp 2.5 -b 0.9998 -s 1.0003 -rp 2 -ra 5
```

This example:
- Automatically detects whether `BTC/USDT` is trending up or down before each operation.
- Enters **bull mode** (buy low, sell high) when tendency is "up", or **bear mode** (sell high, buy back low) when tendency is "down".
- Re-detects tendency between every operation, adapting to changing market conditions.
- If tendency flips during entry scanning, the bot dynamically switches mode without waiting.
- The TUI header shows the currently active mode (BULL/BEAR) updated in real-time.

#### Auto Trade with forced strategy

```bash
binance-bot -f binance-config.yml auto-trade -t "DOGE/USDT" -a 100 -sl 2.0 -tp 2.5 -b 0.9998 -s 1.0003 -rp 6 -ra 0 --strategy bull
```

This example:
- Forces the bot to only enter **bull** (buy-first) operations — useful when your account only holds USDT.
- The bot monitors the market and **waits** for an "up" tendency before placing any orders.
- If tendency flips away during scanning, the bot returns to waiting instead of switching to bear.
- Use `--strategy bear` to force sell-first operations (when you hold the base coin and want to sell first).
- Use `--strategy auto` (default) for fully automatic tendency detection.

#### Bull Trade (uptrending markets)

```bash
binance-bot -f binance-config.yml bull-trade -t "XRP/USDT" -a 50 -sl 1.5 -tp 2.0 -b 0.9998 -s 1.0003 -rp 4 -ra 0
```

This example:
- Trades the pair `XRP/USDT` with an amount of `50`.
- Sets a stop-loss of `1.5%` and a take-profit of `2%`.
- Adjusts buy and sell factors for the LIMIT order target price.
- Rounds the price to 4 decimals and the amount to 0 decimals.

#### Bear Trade (downtrending markets)

```bash
binance-bot -f binance-config.yml bear-trade -t "BTC/USDT" -a 0.001 -sl 2.0 -tp 3.0 -b 0.9998 -s 1.0003 -rp 2 -ra 5
```

This example:
- Sells `0.001 BTC` when bearish signals are detected.
- Sets a stop-loss of `2%` (price rises above entry) and take-profit of `3%` (price drops below entry).
- Buys back at a lower price to capture the difference as profit.

#### Scalp Mode (high-frequency micro-trading)

```bash
binance-bot -f sample-scalp-config.yml bull-trade -t "PEPE/USDT" -a 50 --sl 0.6 --tp 1.0 -b 0.9999 -s 1.0001 -rp 8 -ra 0 -o 500
```

This example:
- Uses 1-minute candles and a scoring-based entry (any 3 of 6 signals bullish).
- Sets tight stop-loss / take-profit suitable for volatile low-cap tokens.
- Runs up to 500 operations with only 5s between them for maximum trade frequency.
- See [sample-scalp-config.yml](/sample-scalp-config.yml) for the full config.

#### Top Gainers Monitor

```bash
binance-bot -f binance-config.yml top-gainers
```

Launches a real-time TUI listing the top 24h price-change gainers on Binance, filtered by quote asset, minimum volume, and an exclude list. Refreshes on the configured `poll-interval`. Press `q` to quit.

Modify these parameters based on your specific trading requirements.

---

#### Explanation of Command Arguments

These arguments apply to the `auto-trade`, `bull-trade`, and `bear-trade` commands:

| Option               | Short | Description                                                                                 | Default       |
|----------------------|-------|---------------------------------------------------------------------------------------------|---------------|
| `--ticker`           | `-t`  | The trading pair ticker in the format `ABC/USD` (e.g., `BTC/USDT`).                         | **Required**  |
| `--amount`           | `-a`  | Amount to trade.                                                                            | **Required**  |
| `--stop-loss`        | `-sl` | Stop-loss percentage (e.g., `1.5` for 1.5%).                                                | `3`           |
| `--take-profit`      | `-tp` | Take-profit percentage (e.g., `3.0` for 3%).                                                | `2.5`         |
| `--buy-factor`       | `-b`  | Factor to determine the target price for a LIMIT buy order.                                 | `0.9999`      |
| `--sell-factor`      | `-s`  | Factor to determine the target price for a LIMIT sell order.                                | `1.0001`      |
| `--round-price`      | `-rp` | Decimal precision for rounding price values.                                                | **Required**  |
| `--round-amount`     | `-ra` | Decimal precision for rounding amount values.                                               | **Required**  |
| `--operations`       | `-o`  | Number of operations to execute during the trading session.                                 | `100`         |
| `--strategy`         | `-st` | *(auto-trade only)* Force entry strategy: `bull`, `bear`, or `auto`.                       | `auto`        |
| `--help`             | `-h`  | Show help for the command.                                                                  | -             |

### Help Commands

- For general help on the bot:
  ```bash
  binance-bot --help
  ```

  Output:
  ```
  NAME:
     binance-bot - A program bot to trade in Binance

  USAGE:
     binance-bot [global options] command <command args>

  VERSION:
     v0.6.0

  AUTHOR:
     Walter Ferreira <wferreirauy@gmail.com>

  COMMANDS:
     bull-trade, bt    Start a bull trade run
     bear-trade, brt   Start a bear trade run (sell high, buy back low)
     auto-trade, at    Automatically detect market tendency and trade accordingly (bull or bear)
     top-gainers, tg   Monitor top market gainers in real-time
     help, h           Shows a list of commands or help for one command

  GLOBAL OPTIONS:
     --config-file FILE, -f FILE  Load configuration from FILE (default: $HOME/binance-config.yml)
     --help, -h     show help
     --version, -v  print the version
  ```

- For help with the `bull-trade` command:
  ```bash
  binance-bot bull-trade --help
  ```

- For help with the `bear-trade` command:
  ```bash
  binance-bot bear-trade --help
  ```

- For help with the `auto-trade` command:
  ```bash
  binance-bot auto-trade --help
  ```

- For help with the `top-gainers` command:
  ```bash
  binance-bot top-gainers --help
  ```

### TUI Keyboard Shortcuts

While the bot is running, the following keys are available inside the TUI:

| Key | Action |
|-----|--------|
| `q` / `Ctrl+C` | Quit the application |
| `h` | Toggle the help / keyboard shortcuts popup |
| `Esc` | Close any open popup |

---

## Configuration

The bot is configured through a YAML file. See [sample-binance-config.yml](/sample-binance-config.yml) for a complete example.

### Indicators Configuration

```yaml
historical-prices:
  period: 100           # number of candlesticks to fetch
  interval: "1m"        # candlestick interval (1m, 3m, 5m, 15m, 1h, etc.)

tendency:
  interval: "3m"        # interval for tendency calculation
  direction: "up"       # expected direction for bull-trade

indicators:
  rsi:
    interval: "5m"
    length: 14
    upper-limit: 70     # overbought threshold
    middle-limit: 50
    lower-limit: 30     # oversold threshold
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
    period: 14          # Average True Range for volatility
  adx:
    period: 14
    threshold: 25       # minimum ADX value to confirm trend strength
  volume:
    ma-period: 20       # volume moving average period
```

### Trailing Stop Configuration

The trailing stop-loss dynamically adjusts to lock in profits as the price moves favorably:

```yaml
trailing-stop:
  enabled: true
  activation-pct: 1.5  # activate after price moves 1.5% in your favor
  trailing-pct: 1.0    # trail by 1.0% from the peak/trough
```

- For **bull trades**: once the price rises by `activation-pct` above buy price, the stop tracks from the highest price and triggers if the price drops `trailing-pct` from that peak.
- For **bear trades**: once the price drops by `activation-pct` below sell price, the stop tracks from the lowest price and triggers if the price rises `trailing-pct` from that trough.

### Scalp Mode Configuration

Scalp mode is optimized for **high-frequency micro-trading** on volatile tickers. Instead of requiring all 6 entry signals simultaneously, it scores each signal and enters when `min-score` are bullish.

```yaml
scalp-mode:
  enabled: true
  min-score: 3           # minimum bullish signals out of 6 to trigger entry
  post-buy-delay: 5      # seconds to wait after fill before exit monitoring
  inter-op-delay: 10     # seconds to wait between completed operations
  require-rsi-exit: false # require RSI momentum confirmation before take-profit
```

**Scoring signals (6 total):**

| # | Signal | Bullish condition |
|---|--------|-------------------|
| 1 | RSI | Below upper limit |
| 2 | MACD | MACD line above signal line |
| 3 | Tendency | Matches configured direction |
| 4 | Bollinger | DEMA closer to lower band than upper band |
| 5 | ADX | Above configured threshold |
| 6 | Volume | Current volume above its moving average |

With `min-score: 3` the bot enters if any 3 of 6 signals are bullish. Raise to `4` or `5` for more selective, lower-frequency entries.

> See [sample-scalp-config.yml](/sample-scalp-config.yml) for a complete high-frequency configuration tuned for 1-minute candles.

**Recommended stop-loss / take-profit for scalping (after 0.2% round-trip fees):**

| Scenario | `--sl` | `--tp` | Net gain/loss |
|----------|--------|--------|---------------|
| Ultra-tight | `0.4` | `0.7` | +0.5% / -0.4% |
| Balanced ✓ | `0.6` | `1.0` | +0.8% / -0.6% |
| Conservative | `1.0` | `1.8` | +1.6% / -1.0% |

### Top Gainers Configuration

```yaml
top-gainers:
  quote-asset: "USDT"      # filter pairs ending with this asset
  limit: 20                # number of top gainers to display
  poll-interval: 60        # seconds between each refresh
  min-volume: 1000000      # minimum 24h quote volume to include
  exclude-symbols:         # symbols to always exclude
    - "USDCUSDT"
```

### AI Configuration

The AI multi-agent system is optional. When enabled, it queries multiple LLM providers concurrently with technical indicators and market sentiment data to produce a consensus trading signal.

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
  min-confidence: 0.5   # minimum consensus confidence to act (0.0 - 1.0)
```

**How it works:**
1. Each configured provider receives a structured prompt with all technical indicators plus real-time sentiment data.
2. Providers are queried **concurrently** for speed.
3. Each agent returns a signal (`BUY`, `SELL`, or `HOLD`), a confidence score (0.0-1.0), and reasoning.
4. A **weighted consensus** algorithm aggregates all votes to produce a final decision.
5. The AI must approve (or at least not contradict) technical signals before trades are executed.

**Sentiment sources (free, no API key required):**
- **CryptoCompare** — latest news headlines for the traded coin
- **Alternative.me Fear & Greed Index** — overall crypto market mood

> Set `ai.enabled: false` or omit all provider API keys to disable AI and run on technical indicators only.

### File Logging

All log levels (orders, info, errors) are automatically written to `binance-bot.log` in the working directory, alongside the TUI display. Color tags are stripped before writing. The file is opened in append mode so logs accumulate across sessions.

```
2026-04-07 12:30:01 [ORDER] BUY 50.000000 PEPE @ 0.00001234 USDT = 0.000617 USDT
2026-04-07 12:30:05 [INFO]  BUY order filled!
2026-04-07 12:30:06 [ERROR] RSI prices: context deadline exceeded
```

### Auto-Notional Adjustment

Binance enforces a minimum notional value (`price × quantity`) per symbol — typically 5 USDT. If the configured `--amount` would produce a notional below this threshold (common with very cheap tokens like PEPE or SHIB), the bot automatically raises the quantity to meet the exchange's `NOTIONAL` and `LOT_SIZE` filters before placing the order. A message is logged when an adjustment occurs:

```
BUY qty adjusted from 50.00000000 to 405210.00000000 to meet exchange filters (minNotional=5.00)
```

No manual intervention is required — the adjustment is transparent and logged.

---

## Trading Strategy Logic

### Bull-Trade

The `bull-trade` command is designed to operate during **bull market trends**, leveraging upward momentum to execute profitable trades.

#### **Buy Conditions**

In **classic mode**, the bot places a buy order when **all** of the following conditions are true simultaneously. In **scalp mode**, the conditions are scored and entry triggers when `min-score` out of 6 are bullish (see [Scalp Mode Configuration](#scalp-mode-configuration)).

1. **RSI**: Value is below the configured `upper-limit` (default 70), indicating the market is not overbought.
2. **MACD Crossover**: The MACD line crosses above the Signal line (classic) or is above the Signal line (scalp), suggesting upward momentum.
3. **Tendency Confirmation**: The trend direction matches the configured direction (DEMA above EMA = "up").
4. **DEMA Proximity to Bollinger Bands**: The current DEMA is closer to the Lower Band than the Upper Band, suggesting a potential reversal from oversold conditions.
5. **ADX Trend Strength** *(if configured)*: ADX is above the threshold (default 25), confirming a strong trend.
6. **Volume Confirmation** *(if configured)*: Current volume exceeds its moving average, avoiding false breakouts.
7. **AI Consensus** *(if enabled)*: The multi-agent system approves the entry or does not contradict it.

#### **Sell Conditions**
The bot will exit a position through one of three mechanisms:

1. **Trailing Stop-Loss** *(if enabled)*: After the price rises by `activation-pct` above buy price, the stop trails from the highest price. Triggers when price drops by `trailing-pct` from the peak.
2. **Fixed Stop-Loss**: The price drops to the stop-loss percentage below buy price. Executes immediately (no AI delay on protective exits).
3. **Take Profit**: The price reaches the take-profit percentage AND RSI is declining (skipped in scalp mode when `require-rsi-exit: false`) AND the AI supports the exit (if enabled).

---

### Bear-Trade

The `bear-trade` command is designed to operate during **bear market trends**, profiting from downward price movement by selling high and buying back low.

#### **Sell Entry Conditions**

In **classic mode**, all conditions must be met simultaneously. In **scalp mode**, `min-score` out of 6 signals must be bearish.

The bot will open a short position (sell) when:

1. **RSI**: Value is above the configured `lower-limit` (default 30), indicating the market is not oversold.
2. **MACD Crossover**: The MACD line crosses below the Signal line, suggesting downward momentum.
3. **Tendency**: The trend direction is "down" (DEMA below EMA).
4. **DEMA Proximity to Bollinger Bands**: The current DEMA is closer to the Upper Band than the Lower Band, suggesting a potential reversal from overbought conditions.
5. **ADX Trend Strength** *(if configured)*: ADX confirms the trend has strength.
6. **Volume Confirmation** *(if configured)*: Current volume exceeds its moving average.
7. **AI Consensus** *(if enabled)*: The multi-agent system approves the entry.

#### **Buy-Back Exit Conditions**
The bot will exit the bear position (buy back) through one of three mechanisms:

1. **Trailing Stop** *(if enabled)*: After the price drops by `activation-pct` below sell price, the stop trails from the lowest price. Triggers when price rises by `trailing-pct` from the trough.
2. **Fixed Stop-Loss**: The price rises to the stop-loss percentage above sell price. Executes immediately.
3. **Take Profit**: The price drops to the take-profit percentage AND RSI is rising (skipped in scalp mode when `require-rsi-exit: false`) AND the AI supports the exit (if enabled).

---

### Auto-Trade (Dynamic Tendency Detection)

The `auto-trade` command removes the need to manually choose between bull and bear strategies. Before each operation, the bot evaluates the current market tendency using the same DEMA-vs-EMA analysis used by the individual modes.

#### **How It Works**

1. **Strategy Selection**: The `--strategy` flag determines behavior:
   - `auto` (default): Detects tendency automatically and trades in whichever direction the market is trending.
   - `bull`: Forces buy-first operations — the bot waits until tendency is "up" before entering. Ideal when you only hold the quote asset (e.g., USDT).
   - `bear`: Forces sell-first operations — the bot waits until tendency is "down" before entering. Ideal when you hold the base asset and want to sell first.
2. **Tendency Detection**: At the start of each operation, the bot fetches historical prices on the configured `tendency.interval` and compares DEMA to EMA. If DEMA > EMA the tendency is "up" (bull); otherwise "down" (bear).
3. **Waiting for Match**: When a strategy is forced (`bull` or `bear`), the bot continuously monitors tendency and only proceeds when it matches the required direction. The TUI shows the mode with "(waiting)" until tendency aligns.
4. **Mode Selection**: Based on the detected/matched tendency, the bot switches to the appropriate strategy — bull (buy low, sell high) or bear (sell high, buy back low).
5. **Live Re-detection**: During entry scanning in `auto` mode, if the tendency flips, the bot immediately adapts and switches to the opposite mode. In forced strategy mode, a tendency flip causes the bot to return to waiting.
6. **Entry & Exit**: Once a mode is selected, the exact same entry conditions (classic or scalp scoring) and exit mechanisms (trailing stop, stop-loss, take-profit, AI confirmation) apply as in the standalone `bull-trade` or `bear-trade` commands.
7. **Per-Operation Adaptation**: After each completed operation (entry + exit), the bot re-detects tendency before the next one.

#### **TUI Display**

The TUI header dynamically shows the current mode:
- `BULL (waiting)` or `BEAR (waiting)` when a forced strategy is waiting for matching tendency
- `AUTO MODE` in cyan at startup (when strategy is auto)
- Switches to `BULL MODE` (green) or `BEAR MODE` (red) once tendency is detected/matched
- Updates in real-time if tendency flips during scanning

> **Tip**: The `auto-trade` command uses the same config file and flags as `bull-trade` / `bear-trade`. The `tendency.direction` config field is ignored — the bot determines direction automatically.

---

### AI Multi-Agent Decision Flow

When AI is enabled, the decision flow operates as follows:

```
Technical Indicators ──┐
                       ├──> AI Agents (concurrent) ──> Weighted Consensus ──> Trade Decision
Sentiment Data ────────┘       │         │         │
                          OpenAI    DeepSeek    Claude
```

- **Entry signals**: Technical conditions must pass first, then AI must approve (or at least HOLD).
- **Stop-loss / trailing-stop exits**: Execute **immediately** without waiting for AI — safety first.
- **Take-profit exits**: Require AI confirmation to avoid exiting too early in strong trends.

---

⚠️ **Note:** Always test the bot in a safe environment (e.g., testnet or small amounts) before live trading. Ensure you understand the risks and implications of using automated trading strategies.

---

## Build from Source

To build the `binance-bot` from the source code, ensure you have the following prerequisites installed:

#### **Prerequisites**
1. **Go (Golang):**
   - Install Go from the [official website](https://go.dev/).
   - Ensure your Go version is at least **1.19** by running:
     ```bash
     go version
     ```

2. **Git:**
   - Clone the repository using Git. Install Git from [here](https://git-scm.com/) if you don't already have it.

#### **Steps to Build**

1. Clone the repository:
   ```bash
   git clone https://github.com/wferreirauy/binance-bot.git
   cd binance-bot
   ```

2. Build the project:
   ```bash
   go build -o binance-bot
   ```

3. Verify the executable:
   ```bash
   ./binance-bot --help
   ```

If the build succeeds, you should see the general help menu displayed, indicating that the bot has been built successfully.

## References

### Binance API documentation

https://binance-docs.github.io/apidocs/spot/en/#general-info

### Binance GO library

https://github.com/binance/binance-connector-go

### AI Provider APIs

- [OpenAI API](https://platform.openai.com/docs)
- [DeepSeek API](https://platform.deepseek.com/api-docs)
- [Anthropic Claude API](https://docs.anthropic.com/en/docs)

### Sentiment Data Sources

- [CryptoCompare News API](https://min-api.cryptocompare.com/) — Free, no API key required
- [Alternative.me Fear & Greed Index](https://alternative.me/crypto/fear-and-greed-index/) — Free, no API key required
