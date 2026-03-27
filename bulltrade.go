package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	binance_connector "github.com/binance/binance-connector-go"
	"github.com/wferreirauy/binance-bot/ai"
	"github.com/wferreirauy/binance-bot/config"
	"github.com/wferreirauy/binance-bot/tui"
)

func BullTrade(
	configFile string,
	symbol string,
	qty float64,
	stopLoss float64,
	takeProfit float64,
	buyFactor float64,
	sellFactor float64,
	roundPrice uint,
	roundAmount uint,
	max_ops uint,
) {

	// read config.yml file
	var c config.Config
	cfg, err := c.Read(configFile)
	if err != nil {
		log.Fatal(err)
	}
	period := cfg.HistoricalPrices.Period     // length period for moving average
	interval := cfg.HistoricalPrices.Interval // time intervals of historical prices for trading

	// refresh interval for price polling (default 10 seconds)
	refreshSecs := cfg.RefreshInterval
	if refreshSecs <= 0 {
		refreshSecs = 10
	}
	refreshInterval := time.Duration(refreshSecs) * time.Second

	// initialize binance api client
	client := binance_connector.NewClient(apikey, secretkey, baseurl)

	// validate symbol in format 0-9A-Z/0-9A-Z
	if re := regexp.MustCompile(`(?m)^[0-9A-Z]{2,8}/[0-9A-Z]{2,8}$`); !re.Match([]byte(symbol)) {
		log.Fatal("error parsing ticker: must match ^[0-9A-Z]{2,8}/[0-9A-Z]{2,8}$")
	}
	scoin, dcoin, found := strings.Cut(symbol, "/")
	if !found {
		log.Fatal("error parsing ticker: \"/\" is missing ")
	}
	ticker := strings.Replace(symbol, "/", "", -1)

	// initialize AI orchestrator
	var aiOrch *ai.Orchestrator
	if cfg.AI.Enabled {
		aiOrch = ai.NewOrchestrator(
			os.Getenv("OPENAI_API_KEY"),
			os.Getenv("DEEPSEEK_API_KEY"),
			os.Getenv("ANTHROPIC_API_KEY"),
			cfg.AI.Providers.OpenAI.Model,
			cfg.AI.Providers.DeepSeek.Model,
			cfg.AI.Providers.Claude.Model,
		)
		if !aiOrch.IsEnabled() {
			aiOrch = nil
		}
	}

	// initialize TUI dashboard
	dash := tui.NewDashboard("BULL", symbol)

	// run trade logic in a goroutine, TUI runs on main thread
	go func() {
		defer dash.Stop()
		if aiOrch != nil {
			dash.LogInfo("AI Agents: [green]ENABLED[-]")
		}
		bullTradeLoop(dash, client, cfg, aiOrch, symbol, ticker, scoin, dcoin, qty, stopLoss, takeProfit, buyFactor, sellFactor, roundPrice, roundAmount, max_ops, period, interval, refreshInterval)
	}()

	if err := dash.Run(); err != nil {
		log.Fatalf("TUI error: %v", err)
	}
}

func bullTradeLoop(
	dash *tui.Dashboard,
	client *binance_connector.Client,
	cfg *config.Config,
	aiOrch *ai.Orchestrator,
	symbol, ticker, scoin, dcoin string,
	qty, stopLoss, takeProfit, buyFactor, sellFactor float64,
	roundPrice, roundAmount, max_ops uint,
	period int,
	interval string,
	refreshInterval time.Duration,
) {
	var buyPrice float64
	var operation = 1

	for range max_ops {
		dash.SetOperation(operation)
		qty = roundFloat(qty, roundAmount)

		//// buy ////
		dash.SetPhase("SCANNING BUY")
		for {
			ohlcv, err := getHistoricalOHLCV(client, ticker, interval, period)
			if err != nil {
				dash.LogError(fmt.Sprintf("OHLCV fetch: %v", err))
				time.Sleep(refreshInterval)
				continue
			}

			price := ohlcv.Closes[len(ohlcv.Closes)-1]
			prevPrice := ohlcv.Closes[len(ohlcv.Closes)-2]
			dash.UpdatePrice(price, prevPrice, roundPrice)

			// tendency
			tendency, err := getTendency(client, ticker, cfg.Tendency.Interval, period)
			if err != nil {
				dash.LogError(fmt.Sprintf("Tendency: %v", err))
				time.Sleep(refreshInterval)
				continue
			}

			// indicators
			dema := calculateDEMA(ohlcv.Closes, cfg.Indicators.Dema.Length)
			currentDema := dema[len(dema)-1]
			rsi := calculateRSI(ohlcv.Closes, cfg.Indicators.Rsi.Length)
			macdLine, signalLine := calculateMACD(ohlcv.Closes, cfg.Indicators.Macd.FastLength, cfg.Indicators.Macd.SlowLength, cfg.Indicators.Macd.SignalLength)
			bb, err := CalculateBollingerBands(ohlcv.Closes, cfg.Indicators.BollingerBands.Length, cfg.Indicators.BollingerBands.Multiplier)
			if err != nil {
				dash.LogError(fmt.Sprintf("BollingerBands: %v", err))
			}
			lowerBand := bb.LowerBand[len(bb.LowerBand)-1]
			upperBand := bb.UpperBand[len(bb.UpperBand)-1]
			distanceToUpper := math.Abs(currentDema - upperBand)
			distanceToLower := math.Abs(currentDema - lowerBand)

			// MACD cross description
			macdCross := "BEARISH"
			if macdLine[len(macdLine)-2] <= signalLine[len(signalLine)-2] && macdLine[len(macdLine)-1] > signalLine[len(signalLine)-1] {
				macdCross = "BULLISH"
			}

			// ADX
			var adxVal float64
			var adxStrong bool
			if cfg.Indicators.Adx.Period > 0 {
				adx := calculateADX(ohlcv.Highs, ohlcv.Lows, ohlcv.Closes, cfg.Indicators.Adx.Period)
				if len(adx) > 0 {
					adxVal = adx[len(adx)-1]
				}
				adxStrong = len(adx) == 0 || adxVal > float64(cfg.Indicators.Adx.Threshold)
			} else {
				adxStrong = true
			}

			// Volume
			var currentVolume, avgVolume float64
			var volumeConfirmed bool
			if cfg.Indicators.Volume.MaPeriod > 0 {
				volumeMA := calculateSMA(ohlcv.Volumes, cfg.Indicators.Volume.MaPeriod)
				currentVolume = ohlcv.Volumes[len(ohlcv.Volumes)-1]
				if len(volumeMA) > 0 {
					avgVolume = volumeMA[len(volumeMA)-1]
				}
				volumeConfirmed = len(volumeMA) == 0 || currentVolume > avgVolume
			} else {
				volumeConfirmed = true
			}

			// Update indicators panel
			dash.UpdateIndicators(&tui.IndicatorData{
				RSI: rsi[len(rsi)-1], RSIUpperLimit: cfg.Indicators.Rsi.UpperLimit, RSILowerLimit: cfg.Indicators.Rsi.LowerLimit,
				MACDLine: macdLine[len(macdLine)-1], SignalLine: signalLine[len(signalLine)-1], MACDCross: macdCross,
				DEMA: currentDema, UpperBand: upperBand, LowerBand: lowerBand,
				Tendency: tendency, ADX: adxVal, ADXThreshold: cfg.Indicators.Adx.Threshold,
				Volume: currentVolume, AvgVolume: avgVolume,
			})

			// AI analysis
			var aiApproved = true
			if aiOrch != nil {
				snapshot := &ai.TechnicalSnapshot{
					Symbol: symbol, Price: price, PrevPrice: prevPrice,
					RSI: rsi[len(rsi)-1], MACDLine: macdLine[len(macdLine)-1], SignalLine: signalLine[len(signalLine)-1],
					PrevMACDLine: macdLine[len(macdLine)-2], PrevSignalLine: signalLine[len(signalLine)-2],
					UpperBand: upperBand, LowerBand: lowerBand, DEMA: currentDema, Tendency: tendency,
					ADX: adxVal, Volume: currentVolume, AvgVolume: avgVolume,
				}
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				consensus, err := aiOrch.Analyze(ctx, snapshot, "BULL")
				cancel()
				if err != nil {
					dash.LogError(fmt.Sprintf("AI: %v", err))
				} else {
					updateDashAI(dash, consensus)
					aiApproved = consensus.ShouldBuy() || consensus.FinalSignal == ai.SignalHold
				}
			}

			// when to buy
			if rsi[len(rsi)-1] < float64(cfg.Indicators.Rsi.UpperLimit) &&
				macdLine[len(macdLine)-2] <= signalLine[len(signalLine)-2] &&
				macdLine[len(macdLine)-1] > signalLine[len(signalLine)-1] &&
				tendency == cfg.Tendency.Direction &&
				distanceToLower < distanceToUpper &&
				adxStrong && volumeConfirmed && aiApproved {

				dash.SetPhase("BUYING")
				buy, err := TradeBuy(symbol, qty, price, buyFactor, roundPrice)
				if err != nil {
					dash.LogError(fmt.Sprintf("BUY order failed: %v", err))
					return
				}
				buyOrder := reflect.ValueOf(buy).Elem()
				orderId := buyOrder.FieldByName("OrderId").Int()
				orderPrice := buyOrder.FieldByName("Price").String()
				buyPrice, _ = strconv.ParseFloat(orderPrice, 64)

				dash.LogOrder(fmt.Sprintf("[green::b]BUY[-] %f %s @ [white::b]%.*f[-] %s = %.*f %s",
					qty, scoin, roundPrice, buyPrice, dcoin, roundPrice, buyPrice*qty, dcoin))

				if getor, err := GetOrder(ticker, orderId); err == nil {
					dash.LogInfo(fmt.Sprintf("BUY order #%d - Status: %s", getor.OrderId, getor.Status))
				}

				for {
					if getor, err := GetOrder(ticker, orderId); err == nil {
						if getor.Status == "FILLED" {
							dash.LogOrder("[green::b]BUY order filled![-]")
							break
						}
					}
					time.Sleep(refreshInterval)
				}
				break
			}
			time.Sleep(refreshInterval)
		}

		time.Sleep(30 * time.Second)

		//// sell ////
		dash.SetPhase("MONITORING SELL")
		highestPrice := buyPrice

		for {
			ohlcv, err := getHistoricalOHLCV(client, ticker, interval, period)
			if err != nil {
				dash.LogError(fmt.Sprintf("OHLCV fetch: %v", err))
				time.Sleep(refreshInterval)
				continue
			}
			rsiprices, err := getHistoricalPrices(client, ticker, cfg.Indicators.Rsi.Interval, period)
			if err != nil {
				dash.LogError(fmt.Sprintf("RSI prices: %v", err))
				time.Sleep(refreshInterval)
				continue
			}

			price := ohlcv.Closes[len(ohlcv.Closes)-1]
			prevPrice := ohlcv.Closes[len(ohlcv.Closes)-2]
			rsi := calculateRSI(rsiprices, cfg.Indicators.Rsi.Length)
			dash.UpdatePrice(price, prevPrice, roundPrice)

			// update indicators panel with sell-phase data
			pnl := (price - buyPrice) / buyPrice * 100
			dash.UpdateIndicators(&tui.IndicatorData{
				RSI: rsi[len(rsi)-1], RSIUpperLimit: cfg.Indicators.Rsi.UpperLimit, RSILowerLimit: cfg.Indicators.Rsi.LowerLimit,
				Tendency: fmt.Sprintf("P&L: %+.2f%%", pnl),
			})

			if price > highestPrice {
				highestPrice = price
			}

			// trailing stop-loss
			if cfg.TrailingStop.Enabled {
				activationPrice := buyPrice * (1 + cfg.TrailingStop.ActivationPct/100)
				if highestPrice >= activationPrice {
					trailingStopPrice := highestPrice * (1 - cfg.TrailingStop.TrailingPct/100)
					if price <= trailingStopPrice {
						dash.SetPhase("TRAILING STOP")
						sell, err := TradeSell(symbol, roundFloat(qty*0.998, roundAmount), price, 1.0, roundPrice)
						if err != nil {
							dash.LogError(fmt.Sprintf("Trailing-Stop SELL failed: %v", err))
							return
						}
						sellOrder := reflect.ValueOf(sell).Elem()
						orderId := sellOrder.FieldByName("OrderId").Int()
						dash.LogOrder(fmt.Sprintf("[fuchsia::b]TRAILING-STOP SELL[-] %f %s @ [white::b]%.*f[-] %s",
							qty, scoin, roundPrice, price, dcoin))
						waitOrderFilled(dash, ticker, orderId, "[fuchsia::b]TRAILING-STOP SELL[-] filled!", refreshInterval)
						break
					}
				}
			}

			// fixed stop loss
			stopLossPrice := buyPrice * (1 - stopLoss/100)
			if price <= stopLossPrice {
				dash.SetPhase("STOP LOSS")
				sell, err := TradeSell(symbol, roundFloat(qty*0.998, roundAmount), price, 1.0, roundPrice)
				if err != nil {
					dash.LogError(fmt.Sprintf("Stop-Loss SELL failed: %v", err))
					return
				}
				sellOrder := reflect.ValueOf(sell).Elem()
				orderId := sellOrder.FieldByName("OrderId").Int()
				dash.LogOrder(fmt.Sprintf("[red::b]STOP-LOSS SELL[-] %f %s @ [white::b]%.*f[-] %s",
					qty, scoin, roundPrice, price, dcoin))
				waitOrderFilled(dash, ticker, orderId, "[red::b]STOP-LOSS SELL[-] filled!", refreshInterval)
				break
			}

			// take profit with AI exit confirmation
			profitPrice := buyPrice * (1 + takeProfit/100)
			var aiSellApproved = true
			if price >= profitPrice && aiOrch != nil {
				snapshot := &ai.TechnicalSnapshot{
					Symbol: symbol, Price: price, PrevPrice: prevPrice,
					RSI: rsi[len(rsi)-1], Tendency: "sell-exit",
				}
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				consensus, err := aiOrch.Analyze(ctx, snapshot, "BULL")
				cancel()
				if err != nil {
					dash.LogError(fmt.Sprintf("AI sell: %v", err))
				} else {
					updateDashAI(dash, consensus)
					aiSellApproved = consensus.ShouldSell() || consensus.FinalSignal == ai.SignalHold
				}
			}
			if price >= profitPrice && rsi[len(rsi)-1] < rsi[len(rsi)-2] && aiSellApproved {
				dash.SetPhase("TAKE PROFIT")
				sell, err := TradeSell(symbol, roundFloat(qty*0.998, roundAmount), price, sellFactor, roundPrice)
				if err != nil {
					dash.LogError(fmt.Sprintf("SELL order failed: %v", err))
					return
				}
				sellOrder := reflect.ValueOf(sell).Elem()
				orderId := sellOrder.FieldByName("OrderId").Int()
				dash.LogOrder(fmt.Sprintf("[red::b]SELL[-] %f %s @ [white::b]%.*f[-] %s = %.*f %s",
					qty, scoin, roundPrice, price, dcoin, roundPrice, price*qty, dcoin))
				waitOrderFilled(dash, ticker, orderId, "[red::b]SELL[-] order filled!", refreshInterval)
				break
			}
			time.Sleep(refreshInterval)
		}

		operation++
		dash.LogInfo(fmt.Sprintf("Operation #%d complete. Next in 1 min...", operation-1))
		time.Sleep(1 * time.Minute)
	}
}

// updateDashAI converts an ai.ConsensusResult into tui.AIConsensusData and updates the dashboard.
func updateDashAI(dash *tui.Dashboard, cr *ai.ConsensusResult) {
	data := &tui.AIConsensusData{
		FinalSignal:    string(cr.FinalSignal),
		AvgConfidence:  cr.AvgConfidence,
		BuyScore:       cr.BuyScore,
		SellScore:      cr.SellScore,
		HoldScore:      cr.HoldScore,
		FearGreed:      -1,
		FearGreedLabel: "",
	}
	if cr.SentimentData != nil {
		data.FearGreed = cr.SentimentData.FearGreedIndex
		data.FearGreedLabel = cr.SentimentData.FearGreedLabel
	}
	for _, d := range cr.Decisions {
		data.Agents = append(data.Agents, tui.AgentResult{
			Provider:   string(d.Provider),
			Signal:     string(d.Signal),
			Confidence: d.Confidence,
			Reasoning:  d.Reasoning,
		})
	}
	dash.UpdateAI(data)
}

// waitOrderFilled polls until an order is filled, logging the result.
func waitOrderFilled(dash *tui.Dashboard, ticker string, orderId int64, filledMsg string, interval time.Duration) {
	for {
		if getor, err := GetOrder(ticker, orderId); err == nil {
			if getor.Status == "FILLED" {
				dash.LogOrder(filledMsg)
				return
			}
		}
		time.Sleep(interval)
	}
}
