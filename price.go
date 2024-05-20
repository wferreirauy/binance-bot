package main

import (
	"context"
	"fmt"

	binance_connector "github.com/binance/binance-connector-go"
)

func GetPrice(t string) (string, error) {
	client := binance_connector.NewClient(apikey, secretkey, baseurl)

	tickerPrice, err := client.NewTickerPriceService().
		Symbol(t).Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	// fmt.Println(binance_connector.PrettyPrint(tickerPrice))
	return tickerPrice.Price, nil
}
