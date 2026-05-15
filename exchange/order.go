package exchange

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	binance "github.com/binance/binance-connector-go"
	"github.com/wferreirauy/binance-bot/indicator"
)

// SymbolFilters holds the relevant trading filters for a symbol.
type SymbolFilters struct {
	MinNotional float64
	MinQty      float64
	StepSize    float64
	BaseAsset   string
	QuoteAsset  string
}

type ManagedOrderResult struct {
	Order              *binance.GetOrderResponse
	TimedOut           bool
	Canceled           bool
	PartiallyFilled    bool
	ExecutedQty        float64
	CumulativeQuoteQty float64
	PartialHandled     bool
}

// GetSymbolFilters fetches MIN_NOTIONAL and LOT_SIZE filters from Binance exchange info.
func GetSymbolFilters(symbol string) (*SymbolFilters, error) {
	client := binance.NewClient(APIKey, SecretKey, BaseURL)
	info, err := client.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		return nil, fmt.Errorf("exchange info: %w", err)
	}
	for _, s := range info.Symbols {
		if s.Symbol != symbol {
			continue
		}
		sf := &SymbolFilters{
			BaseAsset:  s.BaseAsset,
			QuoteAsset: s.QuoteAsset,
		}
		for _, f := range s.Filters {
			switch f.FilterType {
			case "NOTIONAL":
				if v, err := strconv.ParseFloat(f.MinNotional, 64); err == nil {
					sf.MinNotional = v
				}
			case "LOT_SIZE":
				if v, err := strconv.ParseFloat(f.MinQty, 64); err == nil {
					sf.MinQty = v
				}
				if v, err := strconv.ParseFloat(f.StepSize, 64); err == nil {
					sf.StepSize = v
				}
			}
		}
		return sf, nil
	}
	return nil, fmt.Errorf("symbol %s not found in exchange info", symbol)
}

func requiredBalance(side string, quantity, price float64, filters *SymbolFilters) (string, float64, error) {
	if filters == nil {
		return "", 0, fmt.Errorf("balance: missing symbol filters")
	}
	switch side {
	case "BUY":
		if filters.QuoteAsset == "" {
			return "", 0, fmt.Errorf("balance: missing quote asset")
		}
		if price <= 0 {
			return "", 0, fmt.Errorf("balance: buy price must be greater than 0")
		}
		return filters.QuoteAsset, quantity * price, nil
	case "SELL":
		if filters.BaseAsset == "" {
			return "", 0, fmt.Errorf("balance: missing base asset")
		}
		return filters.BaseAsset, quantity, nil
	default:
		return "", 0, fmt.Errorf("balance: unsupported order side %q", side)
	}
}

func EnsureSufficientBalance(symbol, side string, quantity, price float64) error {
	filters, err := GetSymbolFilters(symbol)
	if err != nil {
		return err
	}
	asset, required, err := requiredBalance(side, quantity, price, filters)
	if err != nil {
		return err
	}
	free, err := GetBalance(asset)
	if err != nil {
		return err
	}
	if free+1e-12 < required {
		return fmt.Errorf("balance: insufficient %s balance for %s %s order: need %.8f, available %.8f",
			asset, symbol, side, required, free)
	}
	return nil
}

// AdjustQuantity ensures the order quantity meets MIN_NOTIONAL and LOT_SIZE filters.
// Returns the adjusted quantity and true if it was modified, or the original and false.
func AdjustQuantity(qty, price float64, filters *SymbolFilters, roundPrecision uint) (float64, bool) {
	adjusted := false
	// Ensure minimum notional: price * qty >= minNotional
	if filters.MinNotional > 0 && price > 0 {
		minQtyForNotional := filters.MinNotional / price
		if qty < minQtyForNotional {
			qty = minQtyForNotional * 1.01 // add 1% buffer to avoid edge cases
			adjusted = true
		}
	}
	// Ensure minimum lot size
	if filters.MinQty > 0 && qty < filters.MinQty {
		qty = filters.MinQty
		adjusted = true
	}
	// Align to step size
	if filters.StepSize > 0 {
		qty = math.Ceil(qty/filters.StepSize) * filters.StepSize
	}
	qty = indicator.RoundFloat(qty, roundPrecision)
	return qty, adjusted
}

// Orders fee = 0.01% (* 0.0001)

func GetAllOrders(symbol string) {
	client := binance.NewClient(APIKey, SecretKey, BaseURL)
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
	client := binance.NewClient(APIKey, SecretKey, BaseURL)
	order, err := client.NewGetOrderService().Symbol(symbol).OrderId(id).Do(context.Background())
	if err != nil {
		return &binance.GetOrderResponse{}, err
	}
	return order, nil
}

func NewOrder(symbol, side string, quantity, price float64) (interface{}, error) {

	client := binance.NewClient(APIKey, SecretKey, BaseURL)
	if err := EnsureSufficientBalance(symbol, side, quantity, price); err != nil {
		return nil, err
	}

	newOrder, err := client.NewCreateOrderService().Symbol(symbol).Side(side).
		Type("LIMIT").TimeInForce("GTC").Quantity(quantity).Price(price).Do(context.Background())
	if err != nil {
		return nil, fmt.Errorf("order: creating new order: %w", err)
	}
	return newOrder, nil

}

func NewMarketOrder(symbol, side string, quantity float64) (interface{}, error) {
	return NewMarketOrderWithPrice(symbol, side, quantity, 0)
}

func NewMarketOrderWithPrice(symbol, side string, quantity, estimatedPrice float64) (interface{}, error) {

	client := binance.NewClient(APIKey, SecretKey, BaseURL)
	price := estimatedPrice
	if side == "BUY" && price <= 0 {
		currentPrice, err := GetPrice(client, symbol)
		if err != nil {
			return nil, err
		}
		price = currentPrice
	}
	if err := EnsureSufficientBalance(symbol, side, quantity, price); err != nil {
		return nil, err
	}

	newOrder, err := client.NewCreateOrderService().Symbol(symbol).Side(side).
		Type("MARKET").Quantity(quantity).Do(context.Background())
	if err != nil {
		return nil, fmt.Errorf("order: creating new market order: %w", err)
	}
	return newOrder, nil

}

func CancelOrder(symbol string, orderID int64) error {
	client := binance.NewClient(APIKey, SecretKey, BaseURL)
	_, err := client.NewCancelOrderService().Symbol(symbol).OrderId(orderID).Do(context.Background())
	if err != nil {
		return fmt.Errorf("order: canceling order %d: %w", orderID, err)
	}
	return nil
}

func GetBalance(asset string) (float64, error) {
	client := binance.NewClient(APIKey, SecretKey, BaseURL)
	account, err := client.NewGetAccountService().Do(context.Background())
	if err != nil {
		return 0, fmt.Errorf("account: %w", err)
	}
	for _, balance := range account.Balances {
		if balance.Asset != asset {
			continue
		}
		free, err := strconv.ParseFloat(balance.Free, 64)
		if err != nil {
			return 0, fmt.Errorf("balance: parse %s: %w", asset, err)
		}
		return free, nil
	}
	return 0, nil
}

func GetTradeFeePct(symbol string, defaultPct float64) float64 {
	client := binance.NewClient(APIKey, SecretKey, BaseURL)
	fees, err := client.NewTradeFeeService().Symbol(symbol).Do(context.Background())
	if err != nil || len(fees) == 0 {
		return defaultPct
	}
	taker, err := strconv.ParseFloat(fees[0].TakerCommission, 64)
	if err != nil {
		return defaultPct
	}
	return taker * 100
}

func NetTargetPct(targetPct, feePct, bufferPct float64) float64 {
	net := targetPct - 2*feePct - bufferPct
	if net < 0 {
		return 0
	}
	return net
}

func WaitForManagedOrder(symbol string, orderID int64, side string, timeout time.Duration, pollInterval time.Duration, partialFillAction string) (*ManagedOrderResult, error) {
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}
	started := time.Now()
	for {
		order, err := GetOrder(symbol, orderID)
		if err != nil {
			return nil, err
		}
		executedQty, _ := strconv.ParseFloat(order.ExecutedQty, 64)
		cumulativeQuoteQty, _ := strconv.ParseFloat(order.CumulativeQuoteQty, 64)
		result := &ManagedOrderResult{
			Order:              order,
			ExecutedQty:        executedQty,
			CumulativeQuoteQty: cumulativeQuoteQty,
			PartiallyFilled:    executedQty > 0 && order.Status != "FILLED",
		}
		switch order.Status {
		case "FILLED":
			return result, nil
		case "CANCELED", "REJECTED", "EXPIRED":
			return result, nil
		}
		if timeout > 0 && time.Since(started) >= timeout {
			result.TimedOut = true
			if err := CancelOrder(symbol, orderID); err != nil {
				return result, err
			}
			result.Canceled = true
			if result.PartiallyFilled && partialFillAction == "reverse" {
				reverseSide := "SELL"
				if side == "SELL" {
					reverseSide = "BUY"
				}
				if _, err := NewMarketOrder(symbol, reverseSide, executedQty); err != nil {
					return result, err
				}
				result.PartialHandled = true
			}
			return result, nil
		}
		time.Sleep(pollInterval)
	}
}
