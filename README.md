# Binance Trade Bot

![Captura desde 2024-11-18 00-51-42](https://github.com/user-attachments/assets/50f62e72-7cda-45a9-844c-88b70dbbd772)


## Usage

⚠️ **Warning:** This bot is provided as-is. Use it at your own risk. Trading involves financial risks, and you may incur significant losses. Always test in a safe environment (e.g., a testnet and/or with small amounts) before deploying in live markets. The author is not responsible for any financial outcomes.

---

### Prerequisites

Before using the Binance Trade Bot, you need to configure your environment with the Binance API client credentials. These credentials allow the bot to interact securely with your Binance account. Follow these steps to set up:

1. **Obtain your Binance API Key and Secret**
   - Log in to your [Binance account](https://www.binance.com/).
   - Navigate to the API Management section.
   - Create a new API key, providing a label for it (e.g., `CLI_Bot`).
   - Save the **API Key** and **Secret Key** securely. You will not be able to view the secret again after closing the page.

2. **Set Environment Variables**
   Export the API credentials as environment variables in your terminal before executing the binance-bot cli:

   ```bash
   export BINANCE_API_KEY=<your-api-key>
   export BINANCE_API_SECRET=<your-secret-key>
   ```

Now you're ready to use the Binance Trade Bot! 🎉

### Run the Bot

To start the bot, use the following **example command**:

```bash
binance-bot bull-trade -t "XRP/USDT" -a 50 -sl 1.5 -tp 3.0 -b 0.9995 -s 1.0005 -rp 4 -ra 0
```

#### Example Command Details

The example above demonstrates a configuration with:
- Trading the pair `XRP/USDT`.
- Trading an amount of `50`.
- A stop-loss of `1.5%` and a take-profit of `3%`.
- Adjusted buy and sell factors, rounded the price to 4 decimals and the amount to 0 decimals.

Modify these parameters based on your specific trading requirements.

---

#### Explanation of Command Arguments

| Option               | Short | Description                                                                                 | Default       |
|----------------------|-------|---------------------------------------------------------------------------------------------|---------------|
| `--ticker`           | `-t`  | The trading pair ticker in the format `ABC/USD` (e.g., `BTC/USDT`).                         | **Required**  |
| `--amount`           | `-a`  | Amount to trade.                                                                            | `0`           |
| `--stop-loss`        | `-sl` | Stop-loss percentage (e.g., `1.5` for 1.5%).                                                | `3`           |
| `--take-profit`      | `-tp` | Take-profit percentage (e.g., `3.0` for 3%).                                                | `2.5`         |
| `--buy-factor`       | `-b`  | Factor to determine the target price for a LIMIT buy order.                                 | `0.9999`      |
| `--sell-factor`      | `-s`  | Factor to determine the target price for a LIMIT sell order.                                | `1.0001`      |
| `--round-price`      | `--p` | Decimal precision for rounding price values.                                                | `0`           |
| `--round-amount`     | `-ra` | Decimal precision for rounding amount values.                                               | `0`           |
| `--operations`       | `-o`  | Number of operations to execute during the trading session.                                 | `100`         |
| `--help`             | `-h`  | Show help for the `bull-trade` command.                                                     | -             |

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
     v0.0.1

  AUTHOR:
     Walter Ferreira <wferreirauy@gmail.com>

  COMMANDS:
     bull-trade, bt  Start a bull trade run
     help, h         Shows a list of commands or help for one command

  GLOBAL OPTIONS:
     --help, -h     show help
     --version, -v  print the version
  ```

- For help with the `bull-trade` command:
  ```bash
  binance-bot bull-trade --help
  ```

  Output:
  ```
  NAME:
     binance-bot bull-trade - Start a bull trade run

  USAGE:
     binance-bot bull-trade [command options]

  OPTIONS:
     --ticker value, -t value          ticker to trade, format ABC/USD eg. BTC/USDT
     --amount value, -a value          how much to trade (default: 0)
     --stop-loss value, -sl value     Stop-Loss percentage float, eg. 3.0 (default: 3)
     --take-profit value, -tp value   Take profit percentage float, eg. 2.5 (default: 2.5)
     --buy-factor value, -b value      target factor for LIMIT buy (default: 0.9999)
     --sell-factor value, -s value     target factor for LIMIT sell (default: 1.0001)
     --round-price value, -rp value   price decimals round (default: 0)
     --round-amount value, -ra value  amount decimals round (default: 0)
     --operations value, -o value      number of operations (default: 100)
     --help, -h                        show help
  ```

---

## Trading Strategy Logic

### Bull-Trade

The `bull-trade` command is specifically designed to operate during **bull market trends**, leveraging upward momentum to execute profitable trades. It is optimized for market conditions where prices are generally increasing, making it less effective during bearish or sideways markets.


#### **Buy Conditions**
The bot will place a buy order when these conditions are met
1. **Relative Strength Index (RSI)**:
   - The RSI value must be **less than 70**, indicating that the market is not overbought.

2. **MACD Crossover**:
   - A buy signal is generated when the **MACD line crosses above the Signal line**, suggesting upward momentum.

3. **DEMA & EMA Confirmation**:
   - The market is considered to have an upward trend when the **15-period Double Exponential Moving Average (DEMA)** is **above** the **15-period Exponential Moving Average (EMA)**.

#### **Sell Conditions**
The bot will place a sell order when these conditions are met
1. **Take Profit Factor**:
    - The price reaches the **take-profit percentage** specified in the command.
    - The **MACD line crosses below the Signal line**, indicating potential downward momentum.
2. **Stop Loss Factor**:
   - The price drops to the **stop-loss percentage** specified in the command.

---

This strategy combines momentum indicators, trend confirmation, and predefined risk/reward factors to maximize trading opportunities during bull market conditions. However, the strategy is not designed to handle bearish or flat market trends and may result in losses under such conditions.

⚠️ **Note:** Always test the bot in a safe environment (e.g., testnet or small amounts) before live trading. Ensure you understand the risks and implications of using automated trading strategies.

---

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
