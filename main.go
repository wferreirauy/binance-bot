package main

import (
	"os"
)

var apikey string = os.Getenv("BINANCE_API_KEY")
var secretkey string = os.Getenv("BINANCE_SECRET_KEY")
var baseurl string = "https://api1.binance.com"

func main() {

	/* TODO
	accept and define price range
	create orders - fee = 0.01% (* 0.0001)
	*/

	ticker := "BTCUSDT"
	qty := 0.0003
	sellFactor := 1.002
	buyFactor := 0.999

	BullTrade(ticker, qty, sellFactor, buyFactor, 10)
}
