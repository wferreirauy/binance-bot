package main

import (
	"context"
	"fmt"
	"log"

	binance_connector "github.com/binance/binance-connector-go"
)

// get all orders
// GetAllOrders("SOLUSDT")
func GetAllOrders(symbol string) {
	client := binance_connector.NewClient(apikey, secretkey, baseurl)
	// Binance Get all account orders; active, canceled, or filled - GET /api/v3/allOrders
	getAllOrders, err := client.NewGetAllOrdersService().Symbol(symbol).
		Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(binance_connector.PrettyPrint(getAllOrders))
}

// get order by symbol and id
/* if order, err := GetOrder("SOLUSDT", 6148978765); err == nil {
	fmt.Println(binance_connector.PrettyPrint(order))
} */
func GetOrder(symbol string, id int64) (res *binance_connector.GetOrderResponse, err error) {
	client := binance_connector.NewClient(apikey, secretkey, baseurl)
	order, err := client.NewGetOrderService().Symbol(symbol).OrderId(id).Do(context.Background())
	if err != nil {
		return &binance_connector.GetOrderResponse{}, err
	}
	return order, nil
}

// create new order, side BUY or SELL, with ammount in usdt
/*
	 if order, err := NewOrder("SOLUSDT", "SELL", 0.5, 178.05); err == nil {
		fmt.Println(binance_connector.PrettyPrint(order))
*/
func NewOrder(symbol string, side string, quantity float64, price float64) interface{} {

	client := binance_connector.NewClient(apikey, secretkey, baseurl)

	newOrder, err := client.NewCreateOrderService().Symbol(symbol).Side(side).
		Type("LIMIT").TimeInForce("GTC").Quantity(quantity).Price(price).Do(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	return newOrder

}

func TestNewOrder(symbol string, side string, quantity float64, price float64) interface{} {

	client := binance_connector.NewClient(apikey, secretkey, baseurl)

	newOrder, err := client.NewTestNewOrder().Symbol(symbol).Side(side).
		OrderType("LIMIT").TimeInForce("GTC").Quantity(quantity).Price(price).Do(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	return newOrder

}
