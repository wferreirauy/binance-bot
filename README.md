# Binance Trade Bot

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
