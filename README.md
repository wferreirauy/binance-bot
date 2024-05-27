# Binance Trade Bot

## Usage

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
   --buy-factor value, -b value      target factor for LIMIT buy (default: 0)
   --sell-factor value, -s value     target factor for LIMIT sell (default: 0)
   --round-price value, --rp value   price decimals round (default: 0)
   --round-amount value, --ra value  price decimals round (default: 0)
   --operations value, -o value      number of operations (default: 0)
   --help, -h                        show help
```

## References

### Binance API documentation

https://binance-docs.github.io/apidocs/spot/en/#general-info

### Binance GO library

https://github.com/binance/binance-connector-go
