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
		UsageText:            "binance-bot [global options] command <command options>",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			&cli.Command{
				Name:    "bull-trade",
				Usage:   "start a bull trade run",
				Aliases: []string{"bt"},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "ticker",
						Usage:   "ticker to trade, eg. BTCUSDT",
						Aliases: []string{"t"},
					},
					&cli.Float64Flag{
						Name:    "amount",
						Usage:   "how much to trade",
						Aliases: []string{"a"},
					},
					&cli.Float64Flag{
						Name:    "buyFactor",
						Usage:   "target factor for LIMIT buy",
						Aliases: []string{"b"},
					},
					&cli.Float64Flag{
						Name:    "sellFactor",
						Usage:   "target factor for LIMIT sell",
						Aliases: []string{"s"},
					},
					&cli.IntFlag{
						Name:    "operations",
						Usage:   "number of operations",
						Aliases: []string{"o"},
					},
				},
				Action: func(cCtx *cli.Context) error {
					BullTrade(cCtx.String("ticker"), cCtx.Float64("amount"), cCtx.Float64("buyFactor"),
						cCtx.Float64("sellFactor"), cCtx.Int("operations"))
					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
