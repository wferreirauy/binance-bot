# Binance Trade Bot

### Prerequisites

Before using the Binance CLI bot, you need to configure your environment with the Binance API client credentials. These credentials allow the bot to interact securely with your Binance account. Follow these steps to set up:

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

Now you're ready to use the Binance CLI bot! 🎉

## Usage

![Captura desde 2024-11-18 00-51-42](https://github.com/user-attachments/assets/50f62e72-7cda-45a9-844c-88b70dbbd772)


Run `binance-bot --help` for general help.

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

Run `binance-bot bull-trade --help` for bull-trade command help.

```
NAME:
   binance-bot bull-trade - Start a bull trade run

USAGE:
   binance-bot bull-trade [command options]

OPTIONS:
   --ticker value, -t value          ticker to trade, format ABC/USD eg. BTC/USDT
   --amount value, -a value          how much to trade (default: 0)
   --stop-loss value, --sl value     Stop-Loss percentage float, eg. 3.0 (default: 3)
   --take-profit value, --tp value   Take profit percentage float, eg. 2.5 (default: 2.5)
   --buy-factor value, -b value      target factor for LIMIT buy (default: 0.9999)
   --sell-factor value, -s value     target factor for LIMIT sell (default: 1.0001)
   --round-price value, --rp value   price decimals round (default: 0)
   --round-amount value, --ra value  price decimals round (default: 0)
   --operations value, -o value      number of operations (default: 100)
   --help, -h                        show help
```




## References

### Binance API documentation

https://binance-docs.github.io/apidocs/spot/en/#general-info

### Binance GO library

https://github.com/binance/binance-connector-go
