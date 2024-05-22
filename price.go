package main

import (
	"context"
	"fmt"
	"strconv"

	binance_connector "github.com/binance/binance-connector-go"
)

func GetPrice(t string) (float64, error) {
	client := binance_connector.NewClient(apikey, secretkey, baseurl)

	tickerPrice, err := client.NewTickerPriceService().
		Symbol(t).Do(context.Background())

	if err != nil {
		return 0.0, fmt.Errorf("price: could not get price: %w", err)
	}

	price, err := strconv.ParseFloat(tickerPrice.Price, 64)

	if err != nil {
		return 0.0, fmt.Errorf("price: could not convert price to float: %w", err)
	}

	return price, nil
}
