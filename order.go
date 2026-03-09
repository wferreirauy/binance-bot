package main

import (
	"context"
	"fmt"
	"time"

	binance "github.com/binance/binance-connector-go"
)

// Orders fee = 0.01% (* 0.0001)

func GetAllOrders(symbol string) {
	client := binance.NewClient(apikey, secretkey, baseurl)
	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()
	getAllOrders, err := client.NewGetAllOrdersService().Symbol(symbol).Do(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(binance.PrettyPrint(getAllOrders))
}

func GetOrder(ctx context.Context, symbol string, id int64) (res *binance.GetOrderResponse, err error) {
	client := binance.NewClient(apikey, secretkey, baseurl)
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()
	order, err := client.NewGetOrderService().Symbol(symbol).OrderId(id).Do(ctx)
	if err != nil {
		return &binance.GetOrderResponse{}, err
	}
	return order, nil
}

func CancelOrder(ctx context.Context, symbol string, id int64) error {
	client := binance.NewClient(apikey, secretkey, baseurl)
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()
	_, err := client.NewCancelOrderService().Symbol(symbol).OrderId(id).Do(ctx)
	if err != nil {
		return fmt.Errorf("cancel order %d for %s: %w", id, symbol, err)
	}
	return nil
}

func NewOrder(ctx context.Context, symbol, side string, quantity, price float64) (interface{}, error) {
	rules, err := fetchSymbolRules(ctx, symbol)
	if err != nil {
		return nil, err
	}
	adjQty, adjPrice, err := normalizeOrder(symbol, quantity, price, rules)
	if err != nil {
		return nil, err
	}

	client := binance.NewClient(apikey, secretkey, baseurl)
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	newOrder, err := client.NewCreateOrderService().Symbol(symbol).Side(side).
		Type("LIMIT").TimeInForce("GTC").Quantity(adjQty).Price(adjPrice).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("order: creating new %s limit order on %s with qty %.12f and price %.12f: %w", side, symbol, adjQty, adjPrice, err)
	}
	return newOrder, nil
}

func NewMarketOrder(ctx context.Context, symbol, side string, quantity float64) (interface{}, error) {
	rules, err := fetchSymbolRules(ctx, symbol)
	if err != nil {
		return nil, err
	}
	adjQty, err := normalizeMarketQuantity(symbol, quantity, rules)
	if err != nil {
		return nil, err
	}

	client := binance.NewClient(apikey, secretkey, baseurl)
	ctx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	newOrder, err := client.NewCreateOrderService().Symbol(symbol).Side(side).
		Type("MARKET").Quantity(adjQty).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("order: creating new %s market order on %s with qty %.12f: %w", side, symbol, adjQty, err)
	}
	return newOrder, nil
}

func WaitForOrderFill(ctx context.Context, symbol string, orderID int64, timeout time.Duration) (*binance.GetOrderResponse, error) {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		order, err := GetOrder(waitCtx, symbol, orderID)
		if err == nil {
			switch order.Status {
			case "FILLED":
				return order, nil
			case "CANCELED", "EXPIRED", "REJECTED":
				return order, fmt.Errorf("order %d ended with status %s", orderID, order.Status)
			}
		}

		select {
		case <-waitCtx.Done():
			return nil, fmt.Errorf("timed out waiting for order %d fill: %w", orderID, waitCtx.Err())
		case <-time.After(defaultPollInterval):
		}
	}
}

func WaitForOrderFillOrCancel(ctx context.Context, symbol string, orderID int64, timeout time.Duration) (*binance.GetOrderResponse, error) {
	order, err := WaitForOrderFill(ctx, symbol, orderID, timeout)
	if err == nil {
		return order, nil
	}
	cancelErr := CancelOrder(context.Background(), symbol, orderID)
	if cancelErr != nil {
		return nil, fmt.Errorf("%v; additionally failed to cancel order: %w", err, cancelErr)
	}
	return nil, fmt.Errorf("%v; order cancellation requested", err)
}
