package main

import (
	"log"
	"os"
	"time"

	"github.com/urfave/cli/v2"
)

var apikey string = os.Getenv("BINANCE_API_KEY")
var secretkey string = os.Getenv("BINANCE_SECRET_KEY")
var baseurl string = "https://api1.binance.com"

func main() {
	/* TODO
	accept and define price range
	*/

	app := &cli.App{
		Name:     "binance-bot",
		Version:  "v0.0.1",
		Compiled: time.Now(),
		Authors: []*cli.Author{
			{
				Name:  "Walter Ferreira",
				Email: "wferreirauy@gmail.com",
			},
		},
		HelpName:             "binance-bot",
		Usage:                "A program bot to trade in Binance",
		UsageText:            "binance-bot [global options] command <command args>",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			&cli.Command{
				Name:    "bull-trade",
				Usage:   "Start a bull trade run",
				Aliases: []string{"bt"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "ticker",
						Usage:   "ticker to trade, format ABC/USD eg. BTC/USDT",
						Aliases: []string{"t"},
					},
					&cli.Float64Flag{
						Name:    "amount",
						Usage:   "how much to trade",
						Aliases: []string{"a"},
					},
					&cli.Float64Flag{
						Name:    "buy-factor",
						Usage:   "target factor for LIMIT buy",
						Aliases: []string{"b"},
					},
					&cli.Float64Flag{
						Name:    "sell-factor",
						Usage:   "target factor for LIMIT sell",
						Aliases: []string{"s"},
					},
					&cli.IntFlag{
						Name:    "round-price",
						Usage:   "price decimals round",
						Aliases: []string{"rp"},
					},
					&cli.IntFlag{
						Name:    "round-amount",
						Usage:   "price decimals round",
						Aliases: []string{"ra"},
					},
					&cli.IntFlag{
						Name:    "operations",
						Usage:   "number of operations",
						Aliases: []string{"o"},
					},
				},
				Action: func(cCtx *cli.Context) error {
					BullTrade(cCtx.String("ticker"), cCtx.Float64("amount"), cCtx.Float64("buy-factor"),
						cCtx.Float64("sell-factor"), cCtx.Int("round-price"), cCtx.Int("round-amount"),
						cCtx.Int("operations"))
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
