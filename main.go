package main

import (
	"os"
)

/* Binance API doc
   https://binance-docs.github.io/apidocs/spot/en/#general-info

   Binance GO library
   https://github.com/binance/binance-connector-go
*/

var apikey string = os.Getenv("BINANCE_API_KEY")
var secretkey string = os.Getenv("BINANCE_SECRET_KEY")
var baseurl string = "https://api1.binance.com"

func main() {

	/* TODO
	accept and define price range
	*/

	ticker := "BTCUSDT"
	qty := 0.0003
	sellFactor := 1.002
	buyFactor := 0.999

	BullTrade(ticker, qty, sellFactor, buyFactor, 10)
}
