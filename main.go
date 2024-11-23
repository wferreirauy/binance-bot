package main

import (
	"log"
	"os"
	"time"

	"github.com/urfave/cli/v2"
)

var (
	apikey    string = os.Getenv("BINANCE_API_KEY")
	secretkey string = os.Getenv("BINANCE_SECRET_KEY")
	baseurl   string = "https://api1.binance.com"
)

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
			{
				Name:    "bull-trade",
				Usage:   "Start a bull trade run",
				Aliases: []string{"bt"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "ticker",
						Usage:    "ticker to trade, format ABC/USD eg. BTC/USDT",
						Aliases:  []string{"t"},
						Required: true,
					},
					&cli.Float64Flag{
						Name:     "amount",
						Usage:    "how much to trade",
						Aliases:  []string{"a"},
						Required: true,
					},
					&cli.Float64Flag{
						Name:    "stop-loss",
						Usage:   "Stop-Loss percentage float, eg. 3.0",
						Value:   3.0,
						Aliases: []string{"sl"},
					},
					&cli.Float64Flag{
						Name:    "take-profit",
						Usage:   "Take profit percentage float, eg. 2.5",
						Value:   2.5,
						Aliases: []string{"tp"},
					},
					&cli.Float64Flag{
						Name:    "buy-factor",
						Usage:   "target factor for LIMIT buy",
						Value:   0.9999,
						Aliases: []string{"b"},
					},
					&cli.Float64Flag{
						Name:    "sell-factor",
						Usage:   "target factor for LIMIT sell",
						Value:   1.0001,
						Aliases: []string{"s"},
					},
					&cli.IntFlag{
						Name:     "round-price",
						Usage:    "price decimals round",
						Aliases:  []string{"rp"},
						Required: true,
					},
					&cli.IntFlag{
						Name:     "round-amount",
						Usage:    "amount decimals round",
						Aliases:  []string{"ra"},
						Required: true,
					},
					&cli.IntFlag{
						Name:    "operations",
						Usage:   "number of operations",
						Value:   100,
						Aliases: []string{"o"},
					},
				},
				Action: func(cCtx *cli.Context) error {
					BullTrade(cCtx.String("ticker"), cCtx.Float64("amount"), cCtx.Float64("stop-loss"), cCtx.Float64("take-profit"),
						cCtx.Float64("buy-factor"), cCtx.Float64("sell-factor"), cCtx.Uint("round-price"), cCtx.Uint("round-amount"),
						cCtx.Uint("operations"))
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
