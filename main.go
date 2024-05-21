package main

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"time"
)

var apikey string = os.Getenv("BINANCE_API_KEY")
var secretkey string = os.Getenv("BINANCE_SECRET_KEY")
var baseurl string = "https://api1.binance.com"

func main() {

	/* TODO
	accept and define price range
	create orders - fee = 0.01% (* 0.0001)
	*/

	ticker := "SOLUSDT"
	qty := 0.1
	sellFactor := 1.004
	buyFactor := 0.98

	for range 16 {
		// sell
		sellOrder, err := TradeSell(ticker, qty, sellFactor)
		if err != nil {
			log.Fatal(err)
		}
		// fmt.Println(binance_connector.PrettyPrint(sellOrder))

		sorder := reflect.ValueOf(sellOrder).Elem()
		sid := sorder.FieldByName("OrderId").Int()

		if getor, err := GetOrder(ticker, sid); err == nil {
			fmt.Printf("SELL order created. Id: %d - Status: %s\n\n", getor.OrderId, getor.Status)
		}

		// buy
		buyOrder, err := TradeBuy(ticker, qty, buyFactor)
		if err != nil {
			log.Fatal(err)
		}
		// fmt.Println(binance_connector.PrettyPrint(buyOrder))

		border := reflect.ValueOf(buyOrder).Elem()
		bid := border.FieldByName("OrderId").Int()

		if getor, err := GetOrder(ticker, bid); err == nil {
			fmt.Printf("BUY order created. Id: %d - Status: %s\n\n", getor.OrderId, getor.Status)
		}

		time.Sleep(15 * time.Minute)
	}
}
