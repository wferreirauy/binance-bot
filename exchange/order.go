package exchange

import (
	"context"
	"fmt"
	"math"
	"strconv"

	binance "github.com/binance/binance-connector-go"
	"github.com/wferreirauy/binance-bot/indicator"
)

// SymbolFilters holds the relevant trading filters for a symbol.
type SymbolFilters struct {
	MinNotional float64
	MinQty      float64
	StepSize    float64
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
		sf := &SymbolFilters{}
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

	newOrder, err := client.NewCreateOrderService().Symbol(symbol).Side(side).
		Type("LIMIT").TimeInForce("GTC").Quantity(quantity).Price(price).Do(context.Background())
	if err != nil {
		return nil, fmt.Errorf("order: creating new order: %w", err)
	}
	return newOrder, nil

}

func NewMarketOrder(symbol, side string, quantity float64) (interface{}, error) {

	client := binance.NewClient(APIKey, SecretKey, BaseURL)

	newOrder, err := client.NewCreateOrderService().Symbol(symbol).Side(side).
		Type("MARKET").Quantity(quantity).Do(context.Background())
	if err != nil {
		return nil, fmt.Errorf("order: creating new market order: %w", err)
	}
	return newOrder, nil

}
