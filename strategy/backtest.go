package strategy

import (
	"fmt"
	"log"
	"strings"

	binance_connector "github.com/binance/binance-connector-go"
	"github.com/wferreirauy/binance-bot/config"
	"github.com/wferreirauy/binance-bot/exchange"
	"github.com/wferreirauy/binance-bot/indicator"
	"github.com/wferreirauy/binance-bot/storage"
)

type BacktestResult struct {
	Symbol       string
	Strategy     string
	Trades       int
	Wins         int
	Losses       int
	StartBalance float64
	EndBalance   float64
	ReturnPct    float64
	LastClose    float64
}

func Backtest(configFile, symbol, strategyName string) {
	var c config.Config
	cfg, err := c.Read(configFile)
	if err != nil {
		log.Fatal(err)
	}
	if cfg.BaseURL != "" {
		exchange.BaseURL = cfg.BaseURL
	}
	if strategyName == "" {
		strategyName = "classic-bull"
	}
	result, err := RunBacktest(cfg, symbol, strategyName)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Backtest %s using %s\n", result.Symbol, result.Strategy)
	fmt.Printf("Trades: %d | Wins: %d | Losses: %d\n", result.Trades, result.Wins, result.Losses)
	fmt.Printf("Balance: %.4f -> %.4f (return %+.2f%%)\n", result.StartBalance, result.EndBalance, result.ReturnPct)
	fmt.Printf("Last close: %.8f\n", result.LastClose)
}

func RunBacktest(cfg *config.Config, symbol, strategyName string) (*BacktestResult, error) {
	def, err := GetStrategy(strategyName)
	if err != nil {
		return nil, err
	}
	ticker := strings.Replace(symbol, "/", "", -1)
	client := binance_connector.NewClient(exchange.APIKey, exchange.SecretKey, exchange.BaseURL)
	period := cfg.HistoricalPrices.Period
	if period < 60 {
		period = 60
	}
	ohlcv, err := exchange.GetHistoricalOHLCV(client, ticker, cfg.HistoricalPrices.Interval, period)
	if err != nil {
		return nil, fmt.Errorf("backtest: fetch OHLCV: %w", err)
	}
	initial := cfg.Backtest.InitialBalance
	if initial <= 0 {
		initial = 1000
	}
	feePct := cfg.Backtest.FeePct
	if feePct <= 0 {
		feePct = cfg.Fees.DefaultTakerPct
	}
	balance := initial
	var qty, entryPrice float64
	var position bool
	var trades, wins, losses int
	store, _ := storage.New(cfg.DataDir)

	for i := 30; i < len(ohlcv.Closes); i++ {
		tendency := calculateBacktestTendency(ohlcv.Closes[:i+1])
		signal := def.Decide(MarketSnapshot{
			Symbol:     symbol,
			Index:      i,
			OHLCV:      ohlcv,
			Tendency:   tendency,
			Config:     cfg,
			Position:   position,
			EntryPrice: entryPrice,
		})
		price := ohlcv.Closes[i]
		switch signal {
		case SignalBuy:
			if position {
				continue
			}
			qty = (balance * (1 - feePct/100)) / price
			entryPrice = price
			position = true
			trades++
			if store != nil {
				_ = store.AppendTrade(storage.TradeRecord{Symbol: ticker, Side: "BUY", Status: "BACKTEST", Quantity: qty, Price: price, QuoteQuantity: balance, Reason: strategyName, Operation: trades})
			}
		case SignalSell:
			if !position {
				continue
			}
			exitValue := qty * price * (1 - feePct/100)
			if exitValue > balance {
				wins++
			} else {
				losses++
			}
			balance = exitValue
			position = false
			if store != nil {
				_ = store.AppendTrade(storage.TradeRecord{Symbol: ticker, Side: "SELL", Status: "BACKTEST", Quantity: qty, Price: price, QuoteQuantity: balance, Reason: strategyName, Operation: trades})
			}
		}
	}
	if position && len(ohlcv.Closes) > 0 {
		balance = qty * ohlcv.Closes[len(ohlcv.Closes)-1] * (1 - feePct/100)
	}
	return &BacktestResult{
		Symbol:       symbol,
		Strategy:     strategyName,
		Trades:       trades,
		Wins:         wins,
		Losses:       losses,
		StartBalance: initial,
		EndBalance:   balance,
		ReturnPct:    (balance/initial - 1) * 100,
		LastClose:    ohlcv.Closes[len(ohlcv.Closes)-1],
	}, nil
}

func calculateBacktestTendency(closes []float64) string {
	if len(closes) < 10 {
		return "up"
	}
	dema := indicator.CalculateDEMA(closes, 9)
	sma := indicator.CalculateSMA(closes, minInt(30, len(closes)))
	if len(dema) == 0 || len(sma) == 0 {
		return "up"
	}
	if dema[len(dema)-1] >= sma[len(sma)-1] {
		return "up"
	}
	return "down"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
