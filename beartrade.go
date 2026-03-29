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

// BearTrade implements a sell-high-buy-low strategy for bearish markets.
func BearTrade(
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
	period := cfg.HistoricalPrices.Period
	interval := cfg.HistoricalPrices.Interval

	// refresh interval for price polling (default 10 seconds)
	refreshSecs := cfg.RefreshInterval
	if refreshSecs <= 0 {
		refreshSecs = 10
	}
	refreshInterval := time.Duration(refreshSecs) * time.Second

	// initialize binance api client
	client := binance_connector.NewClient(apikey, secretkey, baseurl)

	// validate symbol
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
	dash := tui.NewDashboard("BEAR", symbol)

	// run trade logic in a goroutine, TUI runs on main thread
	go func() {
		defer dash.Stop()
		dash.SetRefreshInterval(refreshInterval)
		if aiOrch != nil {
			dash.LogInfo("AI Agents: [green]ENABLED[-]")
		}
		bearTradeLoop(dash, client, cfg, aiOrch, symbol, ticker, scoin, dcoin, qty, stopLoss, takeProfit, buyFactor, sellFactor, roundPrice, roundAmount, max_ops, period, interval, refreshInterval)
	}()

	if err := dash.Run(); err != nil {
		log.Fatalf("TUI error: %v", err)
	}
}

func bearTradeLoop(
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
	var sellPrice float64
	var operation = 1

	for range max_ops {
		dash.SetOperation(operation)
		qty = roundFloat(qty, roundAmount)

		//// sell (bear entry) ////
		dash.SetPhase("SCANNING SELL")
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

			// MACD cross
			macdCross := "BULLISH"
			if macdLine[len(macdLine)-2] >= signalLine[len(signalLine)-2] && macdLine[len(macdLine)-1] < signalLine[len(signalLine)-1] {
				macdCross = "BEARISH"
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
				consensus, err := aiOrch.Analyze(ctx, snapshot, "BEAR")
				cancel()
				if err != nil {
					dash.LogError(fmt.Sprintf("AI: %v", err))
				} else {
					updateDashAI(dash, consensus)
					aiApproved = consensus.ShouldSell() || consensus.FinalSignal == ai.SignalHold
				}
			}

			// when to sell (bear entry)
			if rsi[len(rsi)-1] > float64(cfg.Indicators.Rsi.LowerLimit) &&
				macdLine[len(macdLine)-2] >= signalLine[len(signalLine)-2] &&
				macdLine[len(macdLine)-1] < signalLine[len(signalLine)-1] &&
				tendency == "down" &&
				distanceToUpper < distanceToLower &&
				adxStrong && volumeConfirmed && aiApproved {

				dash.SetPhase("SELLING")
				sell, err := TradeSell(symbol, qty, price, sellFactor, roundPrice)
				if err != nil {
					dash.LogError(fmt.Sprintf("SELL order failed: %v", err))
					return
				}
				sellOrder := reflect.ValueOf(sell).Elem()
				orderId := sellOrder.FieldByName("OrderId").Int()
				orderPrice := sellOrder.FieldByName("Price").String()
				sellPrice, _ = strconv.ParseFloat(orderPrice, 64)

				dash.LogOrder(fmt.Sprintf("[red::b]SELL[-] %f %s @ [white::b]%.*f[-] %s = %.*f %s",
					qty, scoin, roundPrice, sellPrice, dcoin, roundPrice, sellPrice*qty, dcoin))

				if getor, err := GetOrder(ticker, orderId); err == nil {
					dash.LogInfo(fmt.Sprintf("SELL order #%d - Status: %s", getor.OrderId, getor.Status))
				}

				waitOrderFilled(dash, ticker, orderId, "[red::b]SELL[-] order filled!", refreshInterval)
				break
			}
			time.Sleep(refreshInterval)
		}

		time.Sleep(30 * time.Second)

		//// buy back (bear exit) ////
		dash.SetPhase("MONITORING BUY-BACK")
		lowestPrice := sellPrice
		sellProceeds := sellPrice * qty

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

			// update indicators with P&L
			pnl := (sellPrice - price) / sellPrice * 100
			dash.UpdateIndicators(&tui.IndicatorData{
				RSI: rsi[len(rsi)-1], RSIUpperLimit: cfg.Indicators.Rsi.UpperLimit, RSILowerLimit: cfg.Indicators.Rsi.LowerLimit,
				Tendency: fmt.Sprintf("P&L: %+.2f%%", pnl),
			})

			if price < lowestPrice {
				lowestPrice = price
			}

			// trailing stop (inverse for bear)
			if cfg.TrailingStop.Enabled {
				activationPrice := sellPrice * (1 - cfg.TrailingStop.ActivationPct/100)
				if lowestPrice <= activationPrice {
					trailingStopPrice := lowestPrice * (1 + cfg.TrailingStop.TrailingPct/100)
					if price >= trailingStopPrice {
						dash.SetPhase("TRAILING STOP")
						buyBackQty := roundFloat(sellProceeds/price, roundAmount)
						buy, err := TradeBuy(symbol, buyBackQty, price, 1.0, roundPrice)
						if err != nil {
							dash.LogError(fmt.Sprintf("Trailing-Stop BUY failed: %v", err))
							return
						}
						buyOrder := reflect.ValueOf(buy).Elem()
						orderId := buyOrder.FieldByName("OrderId").Int()
						dash.LogOrder(fmt.Sprintf("[fuchsia::b]TRAILING-STOP BUY[-] %f %s @ [white::b]%.*f[-] %s",
							buyBackQty, scoin, roundPrice, price, dcoin))
						waitOrderFilled(dash, ticker, orderId, "[fuchsia::b]TRAILING-STOP BUY[-] filled!", refreshInterval)
						break
					}
				}
			}

			// stop loss: price goes UP
			stopLossPrice := sellPrice * (1 + stopLoss/100)
			if price >= stopLossPrice {
				dash.SetPhase("STOP LOSS")
				buyBackQty := roundFloat(sellProceeds/price, roundAmount)
				buy, err := TradeBuy(symbol, buyBackQty, price, 1.0, roundPrice)
				if err != nil {
					dash.LogError(fmt.Sprintf("Stop-Loss BUY failed: %v", err))
					return
				}
				buyOrder := reflect.ValueOf(buy).Elem()
				orderId := buyOrder.FieldByName("OrderId").Int()
				dash.LogOrder(fmt.Sprintf("[red::b]STOP-LOSS BUY[-] %f %s @ [white::b]%.*f[-] %s",
					buyBackQty, scoin, roundPrice, price, dcoin))
				waitOrderFilled(dash, ticker, orderId, "[red::b]STOP-LOSS BUY[-] filled!", refreshInterval)
				break
			}

			// take profit with AI exit confirmation
			profitPrice := sellPrice * (1 - takeProfit/100)
			var aiBuyApproved = true
			if price <= profitPrice && aiOrch != nil {
				snapshot := &ai.TechnicalSnapshot{
					Symbol: symbol, Price: price, PrevPrice: prevPrice,
					RSI: rsi[len(rsi)-1], Tendency: "buy-exit",
				}
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				consensus, err := aiOrch.Analyze(ctx, snapshot, "BEAR")
				cancel()
				if err != nil {
					dash.LogError(fmt.Sprintf("AI buy-back: %v", err))
				} else {
					updateDashAI(dash, consensus)
					aiBuyApproved = consensus.ShouldBuy() || consensus.FinalSignal == ai.SignalHold
				}
			}
			if price <= profitPrice && rsi[len(rsi)-1] > rsi[len(rsi)-2] && aiBuyApproved {
				dash.SetPhase("TAKE PROFIT")
				buyBackQty := roundFloat(sellProceeds/price, roundAmount)
				buy, err := TradeBuy(symbol, buyBackQty, price, buyFactor, roundPrice)
				if err != nil {
					dash.LogError(fmt.Sprintf("BUY order failed: %v", err))
					return
				}
				buyOrder := reflect.ValueOf(buy).Elem()
				orderId := buyOrder.FieldByName("OrderId").Int()
				dash.LogOrder(fmt.Sprintf("[green::b]BUY[-] %f %s @ [white::b]%.*f[-] %s = %.*f %s",
					buyBackQty, scoin, roundPrice, price, dcoin, roundPrice, price*buyBackQty, dcoin))
				waitOrderFilled(dash, ticker, orderId, "[green::b]BUY[-] order filled!", refreshInterval)
				break
			}
			time.Sleep(refreshInterval)
		}

		operation++
		dash.LogInfo(fmt.Sprintf("Operation #%d complete. Next in 1 min...", operation-1))
		time.Sleep(1 * time.Minute)
	}
}
