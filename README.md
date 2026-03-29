# Binance Trade Bot

<img width="1470" height="956" alt="Image" src="https://github.com/user-attachments/assets/bc3c2602-bac7-47a8-badf-7bd5987c9c4e" />

![Captura desde 2024-12-07 03-23-41](https://github.com/user-attachments/assets/4d251761-ed43-4649-842f-fd016bc62532)

## Features

- **Bull Trade** — Buy-low-sell-high strategy for uptrending markets
- **Bear Trade** — Sell-high-buy-low strategy for downtrending markets
- **AI Multi-Agent System** — Concurrent analysis from OpenAI, DeepSeek, and Claude with weighted consensus
- **Sentiment Analysis** — Real-time news headlines and Fear & Greed Index integrated into AI decisions
- **Trailing Stop-Loss** — Dynamically locks in profits as price moves favorably
- **Advanced Indicators** — RSI, MACD, DEMA, Bollinger Bands, ADX, ATR, VWAP, and volume confirmation
- **Full OHLCV Analysis** — Uses complete candlestick data instead of close-only prices

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

Modify these parameters based on your specific trading requirements.

---

#### Explanation of Command Arguments

These arguments apply to both `bull-trade` and `bear-trade` commands:

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
     v0.0.5

  AUTHOR:
     Walter Ferreira <wferreirauy@gmail.com>

  COMMANDS:
     bull-trade, bt   Start a bull trade run
     bear-trade, brt  Start a bear trade run (sell high, buy back low)
     help, h          Shows a list of commands or help for one command

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

---

## Trading Strategy Logic

### Bull-Trade

The `bull-trade` command is designed to operate during **bull market trends**, leveraging upward momentum to execute profitable trades.

#### **Buy Conditions**
The bot will place a buy order when **all** of these conditions are met:

1. **RSI**: Value is below the configured `upper-limit` (default 70), indicating the market is not overbought.
2. **MACD Crossover**: The MACD line crosses above the Signal line, suggesting upward momentum.
3. **Tendency Confirmation**: The trend direction matches the configured direction (DEMA above EMA = "up").
4. **DEMA Proximity to Bollinger Bands**: The current DEMA is closer to the Lower Band than the Upper Band, suggesting a potential reversal from oversold conditions.
5. **ADX Trend Strength** *(if configured)*: ADX is above the threshold (default 25), confirming a strong trend.
6. **Volume Confirmation** *(if configured)*: Current volume exceeds its moving average, avoiding false breakouts.
7. **AI Consensus** *(if enabled)*: The multi-agent system approves the entry or does not contradict it.

#### **Sell Conditions**
The bot will exit a position through one of three mechanisms:

1. **Trailing Stop-Loss** *(if enabled)*: After the price rises by `activation-pct` above buy price, the stop trails from the highest price. Triggers when price drops by `trailing-pct` from the peak.
2. **Fixed Stop-Loss**: The price drops to the stop-loss percentage below buy price. Executes immediately (no AI delay on protective exits).
3. **Take Profit**: The price reaches the take-profit percentage AND the RSI turns downward AND the AI supports the exit (if enabled).

---

### Bear-Trade

The `bear-trade` command is designed to operate during **bear market trends**, profiting from downward price movement by selling high and buying back low.

#### **Sell Entry Conditions**
The bot will open a short position (sell) when **all** of these conditions are met:

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
3. **Take Profit**: The price drops to the take-profit percentage AND the RSI turns upward AND the AI supports the exit (if enabled).

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
