package main

import (
	"context"
	"fmt"

	binance "github.com/binance/binance-connector-go"
)

// Orders fee = 0.01% (* 0.0001)

func GetAllOrders(symbol string) {
	client := binance.NewClient(apikey, secretkey, baseurl)
	// Binance Get all account orders; active, canceled, or filled - GET /api/v3/allOrders
	getAllOrders, err := client.NewGetAllOrdersService().Symbol(symbol).
		Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(binance.PrettyPrint(getAllOrders))
}

func GetOrder(symbol string, id int64) (res *binance.GetOrderResponse, err error) {
	client := binance.NewClient(apikey, secretkey, baseurl)
	order, err := client.NewGetOrderService().Symbol(symbol).OrderId(id).Do(context.Background())
	if err != nil {
		return &binance.GetOrderResponse{}, err
	}
	return order, nil
}

func NewOrder(symbol, side string, quantity, price float64) (interface{}, error) {

	client := binance.NewClient(apikey, secretkey, baseurl)

	newOrder, err := client.NewCreateOrderService().Symbol(symbol).Side(side).
		Type("LIMIT").TimeInForce("GTC").Quantity(quantity).Price(price).Do(context.Background())
	if err != nil {
		return nil, fmt.Errorf("order: creating new order: %w", err)
	}
	return newOrder, nil

}

/* func placeStopLossOrder(client *binance.Client, symbol string, quantity, stopPrice, limitPrice float64) error {
	order, err := client.NewCreateOrderService().
		Symbol(symbol).
		Side(binance.SideTypeSell).
		Type(binance.OrderTypeStopLossLimit).
		Quantity(fmt.Sprintf("%f", quantity)).
		StopPrice(fmt.Sprintf("%f", stopPrice)).
		Price(fmt.Sprintf("%f", limitPrice)).
		Do(context.Background())
	if err != nil {
		return err
	}
	log.Printf("Orden de stop-loss creada: %v", order)
	return nil
} */
