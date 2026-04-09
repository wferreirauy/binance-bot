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

// DynamicTrade automatically detects market tendency and switches between
// bull (buy-low/sell-high) and bear (sell-high/buy-low) strategies per operation.
func DynamicTrade(
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
	strategy string,
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

	// validate strategy flag
	strategy = strings.ToLower(strategy)
	if strategy != "auto" && strategy != "bull" && strategy != "bear" {
		log.Fatal("error: --strategy must be 'auto', 'bull', or 'bear'")
	}

	// validate symbol in format 0-9A-Z/0-9A-Z
	if re := regexp.MustCompile(`(?m)^[0-9A-Z]{1,8}/[0-9A-Z]{2,8}$`); !re.Match([]byte(symbol)) {
		log.Fatal("error parsing ticker: must match ^[0-9A-Z]{1,8}/[0-9A-Z]{2,8}$")
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

	// initialize TUI dashboard with mode based on strategy
	initialMode := "AUTO"
	if strategy == "bull" {
		initialMode = "BULL (waiting)"
	} else if strategy == "bear" {
		initialMode = "BEAR (waiting)"
	}
	dash := tui.NewDashboard(initialMode, symbol)

	// initialize file logger
	fl, err := tui.NewFileLogger("binance-bot.log")
	if err != nil {
		log.Printf("Warning: could not open log file: %v", err)
	} else {
		defer fl.Close()
		dash.SetFileLogger(fl)
	}

	// run trade logic in a goroutine, TUI runs on main thread
	go func() {
		defer dash.Stop()
		dash.SetRefreshInterval(refreshInterval)
		dash.SetParams(&tui.TradeParams{
			Amount: qty, StopLoss: stopLoss, TakeProfit: takeProfit,
			BuyFactor: buyFactor, SellFactor: sellFactor,
			RoundPrice: roundPrice, RoundAmt: roundAmount, MaxOps: max_ops,
		})
		if aiOrch != nil {
			dash.LogInfo("AI Agents: [green]ENABLED[-]")
		}
		if strategy == "auto" {
			dash.LogInfo("[cyan::b]AUTO MODE[-] — tendency will be detected each operation")
		} else {
			dash.LogInfo(fmt.Sprintf("[cyan::b]%s STRATEGY[-] — will wait for matching tendency before entering", strings.ToUpper(strategy)))
		}
		dynamicTradeLoop(dash, client, cfg, aiOrch, symbol, ticker, scoin, dcoin, qty, stopLoss, takeProfit, buyFactor, sellFactor, roundPrice, roundAmount, max_ops, period, interval, refreshInterval, strategy)
	}()

	if err := dash.Run(); err != nil {
		log.Fatalf("TUI error: %v", err)
	}
}

func dynamicTradeLoop(
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
	strategy string,
) {
	var operation = 1
	var consecutiveSL int

	for range max_ops {
		dash.SetOperation(operation)
		qty = roundFloat(qty, roundAmount)

		// Detect current market tendency before each operation
		dash.SetPhase("DETECTING TENDENCY")
		var tendency string
		var isBull bool

		for {
			var err error
			tendency, err = getTendency(client, ticker, cfg.Tendency.Interval, period)
			if err != nil {
				dash.LogError(fmt.Sprintf("Tendency detection: %v", err))
				time.Sleep(refreshInterval)
				continue
			}

			// When a strategy is forced, wait for tendency to match
			if strategy == "bull" && tendency != "up" {
				dash.SetTradeMode("BULL (waiting)")
				dash.LogInfo(fmt.Sprintf("[yellow]Tendency is %s[-] — waiting for [green]UP[-] tendency to match BULL strategy", tendency))
				time.Sleep(refreshInterval)
				continue
			}
			if strategy == "bear" && tendency != "down" {
				dash.SetTradeMode("BEAR (waiting)")
				dash.LogInfo(fmt.Sprintf("[yellow]Tendency is %s[-] — waiting for [red]DOWN[-] tendency to match BEAR strategy", tendency))
				time.Sleep(refreshInterval)
				continue
			}
			break
		}

		isBull = tendency == "up"
		if isBull {
			dash.SetTradeMode("BULL")
			dash.LogInfo(fmt.Sprintf("[green::b]▲ BULL[-] tendency detected on %s — entering BUY mode", cfg.Tendency.Interval))
		} else {
			dash.SetTradeMode("BEAR")
			dash.LogInfo(fmt.Sprintf("[red::b]▼ BEAR[-] tendency detected on %s — entering SELL mode", cfg.Tendency.Interval))
		}

		//// ENTRY PHASE ////
		var entryPrice float64
		if isBull {
			dash.SetPhase("SCANNING BUY")
		} else {
			dash.SetPhase("SCANNING SELL")
		}

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

			// re-check tendency during scanning
			tendency, err = getTendency(client, ticker, cfg.Tendency.Interval, period)
			if err != nil {
				dash.LogError(fmt.Sprintf("Tendency: %v", err))
				time.Sleep(refreshInterval)
				continue
			}

			// if tendency flipped during scanning, handle based on strategy
			if (isBull && tendency != "up") || (!isBull && tendency != "down") {
				if strategy == "auto" {
					// auto mode: switch to the new tendency
					dash.LogInfo(fmt.Sprintf("[yellow]Tendency flipped to %s during scanning — re-detecting[-]", tendency))
					isBull = tendency == "up"
					if isBull {
						dash.SetTradeMode("BULL")
						dash.SetPhase("SCANNING BUY")
					} else {
						dash.SetTradeMode("BEAR")
						dash.SetPhase("SCANNING SELL")
					}
				} else {
					// forced strategy: tendency no longer matches, go back to waiting
					dash.LogInfo(fmt.Sprintf("[yellow]Tendency flipped to %s — no longer matches %s strategy, returning to wait[-]", tendency, strings.ToUpper(strategy)))
					break
				}
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
			var macdCross string
			if isBull {
				macdCross = "BEARISH"
				if macdLine[len(macdLine)-2] <= signalLine[len(signalLine)-2] && macdLine[len(macdLine)-1] > signalLine[len(signalLine)-1] {
					macdCross = "BULLISH"
				}
			} else {
				macdCross = "BULLISH"
				if macdLine[len(macdLine)-2] >= signalLine[len(signalLine)-2] && macdLine[len(macdLine)-1] < signalLine[len(signalLine)-1] {
					macdCross = "BEARISH"
				}
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
				aiMode := "BULL"
				if !isBull {
					aiMode = "BEAR"
				}
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				consensus, err := aiOrch.Analyze(ctx, snapshot, aiMode)
				cancel()
				if err != nil {
					dash.LogError(fmt.Sprintf("AI: %v", err))
				} else {
					updateDashAI(dash, consensus)
					if isBull {
						aiApproved = consensus.ShouldBuy() || consensus.FinalSignal == ai.SignalHold
					} else {
						aiApproved = consensus.ShouldSell() || consensus.FinalSignal == ai.SignalHold
					}
				}
			}

			// Higher-timeframe trend gate
			if cfg.Tendency.HTFEnabled && cfg.Tendency.HTFInterval != "" {
				htfTendency, htfErr := getTendency(client, ticker, cfg.Tendency.HTFInterval, period)
				if htfErr != nil {
					dash.LogError(fmt.Sprintf("HTF Tendency: %v", htfErr))
				} else {
					expectedHTF := "up"
					if !isBull {
						expectedHTF = "down"
					}
					if htfTendency != expectedHTF {
						dash.LogInfo(fmt.Sprintf("[red]HTF GATE[-] %s trend is [red]%s[-] on %s — skipping %s entry",
							symbol, htfTendency, cfg.Tendency.HTFInterval, strings.ToUpper(tendency)))
						time.Sleep(refreshInterval)
						continue
					}
				}
			}

			// Entry conditions
			var shouldEnter bool
			if cfg.ScalpMode.Enabled {
				score := 0
				if isBull {
					if rsi[len(rsi)-1] < float64(cfg.Indicators.Rsi.UpperLimit) {
						score++
					}
					if macdLine[len(macdLine)-1] > signalLine[len(signalLine)-1] {
						score++
					}
					if tendency == "up" {
						score++
					}
					if distanceToLower < distanceToUpper {
						score++
					}
				} else {
					if rsi[len(rsi)-1] > float64(cfg.Indicators.Rsi.LowerLimit) {
						score++
					}
					if macdLine[len(macdLine)-1] < signalLine[len(signalLine)-1] {
						score++
					}
					if tendency == "down" {
						score++
					}
					if distanceToUpper < distanceToLower {
						score++
					}
				}
				if adxStrong {
					score++
				}
				if volumeConfirmed {
					score++
				}
				minScore := cfg.ScalpMode.MinScore
				if minScore <= 0 {
					minScore = 3
				}
				shouldEnter = score >= minScore && aiApproved
				if shouldEnter {
					dash.LogInfo(fmt.Sprintf("[yellow]Scalp entry: score %d/%d (min %d)[-]", score, 6, minScore))
				}
			} else {
				if isBull {
					shouldEnter = rsi[len(rsi)-1] < float64(cfg.Indicators.Rsi.UpperLimit) &&
						macdLine[len(macdLine)-2] <= signalLine[len(signalLine)-2] &&
						macdLine[len(macdLine)-1] > signalLine[len(signalLine)-1] &&
						tendency == "up" &&
						distanceToLower < distanceToUpper &&
						adxStrong && volumeConfirmed && aiApproved
				} else {
					shouldEnter = rsi[len(rsi)-1] > float64(cfg.Indicators.Rsi.LowerLimit) &&
						macdLine[len(macdLine)-2] >= signalLine[len(signalLine)-2] &&
						macdLine[len(macdLine)-1] < signalLine[len(signalLine)-1] &&
						tendency == "down" &&
						distanceToUpper < distanceToLower &&
						adxStrong && volumeConfirmed && aiApproved
				}
			}

			if shouldEnter {
				if isBull {
					dash.SetPhase("BUYING")
					buy, err := TradeBuy(symbol, qty, price, buyFactor, roundPrice)
					if err != nil {
						dash.LogError(fmt.Sprintf("BUY order failed: %v", err))
						return
					}
					buyOrder := reflect.ValueOf(buy).Elem()
					orderId := buyOrder.FieldByName("OrderId").Int()
					orderPrice := buyOrder.FieldByName("Price").String()
					entryPrice, _ = strconv.ParseFloat(orderPrice, 64)

					dash.LogOrder(fmt.Sprintf("[green::b]BUY[-] %f %s @ [white::b]%.*f[-] %s = %.*f %s",
						qty, scoin, roundPrice, entryPrice, dcoin, roundPrice, entryPrice*qty, dcoin))

					if getor, err := GetOrder(ticker, orderId); err == nil {
						dash.LogInfo(fmt.Sprintf("BUY order #%d - Status: %s", getor.OrderId, getor.Status))
					}
					waitOrderFilled(dash, ticker, orderId, "[green::b]BUY order filled![-]", refreshInterval)
				} else {
					dash.SetPhase("SELLING")
					sell, err := TradeSell(symbol, qty, price, sellFactor, roundPrice)
					if err != nil {
						dash.LogError(fmt.Sprintf("SELL order failed: %v", err))
						return
					}
					sellOrder := reflect.ValueOf(sell).Elem()
					orderId := sellOrder.FieldByName("OrderId").Int()
					orderPrice := sellOrder.FieldByName("Price").String()
					entryPrice, _ = strconv.ParseFloat(orderPrice, 64)

					dash.LogOrder(fmt.Sprintf("[red::b]SELL[-] %f %s @ [white::b]%.*f[-] %s = %.*f %s",
						qty, scoin, roundPrice, entryPrice, dcoin, roundPrice, entryPrice*qty, dcoin))

					if getor, err := GetOrder(ticker, orderId); err == nil {
						dash.LogInfo(fmt.Sprintf("SELL order #%d - Status: %s", getor.OrderId, getor.Status))
					}
					waitOrderFilled(dash, ticker, orderId, "[red::b]SELL order filled![-]", refreshInterval)
				}
				break
			}
			time.Sleep(refreshInterval)
		}

		postDelay := 30
		if cfg.ScalpMode.Enabled && cfg.ScalpMode.PostBuyDelay > 0 {
			postDelay = cfg.ScalpMode.PostBuyDelay
		}
		time.Sleep(time.Duration(postDelay) * time.Second)

		//// EXIT PHASE ////
		exitType := ""
		if isBull {
			dash.SetPhase("MONITORING SELL")
			highestPrice := entryPrice

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

				pnl := (price - entryPrice) / entryPrice * 100
				dash.UpdateIndicators(&tui.IndicatorData{
					RSI: rsi[len(rsi)-1], RSIUpperLimit: cfg.Indicators.Rsi.UpperLimit, RSILowerLimit: cfg.Indicators.Rsi.LowerLimit,
					Tendency: fmt.Sprintf("P&L: %+.2f%%", pnl),
				})

				if price > highestPrice {
					highestPrice = price
				}

				// trailing stop-loss
				if cfg.TrailingStop.Enabled {
					activationPrice := entryPrice * (1 + cfg.TrailingStop.ActivationPct/100)
					if highestPrice >= activationPrice {
						trailingStopPrice := highestPrice * (1 - cfg.TrailingStop.TrailingPct/100)
						if price <= trailingStopPrice {
							dash.SetPhase("TRAILING STOP")
							sell, err := TradeMarketSell(symbol, roundFloat(qty*0.998, roundAmount), price, roundPrice)
							if err != nil {
								dash.LogError(fmt.Sprintf("Trailing-Stop MARKET SELL failed: %v", err))
								return
							}
							sellOrder := reflect.ValueOf(sell).Elem()
							orderId := sellOrder.FieldByName("OrderId").Int()
							dash.LogOrder(fmt.Sprintf("[fuchsia::b]TRAILING-STOP MARKET SELL[-] %f %s @ ~[white::b]%.*f[-] %s",
								qty, scoin, roundPrice, price, dcoin))
							waitOrderFilled(dash, ticker, orderId, "[fuchsia::b]TRAILING-STOP MARKET SELL[-] filled!", refreshInterval)
							exitType = "ts"
							break
						}
					}
				}

				// ATR-based dynamic stop-loss
				effectiveSL := stopLoss
				if cfg.ScalpMode.ATRStopLoss && cfg.Indicators.Atr.Period > 0 {
					atr := calculateATR(ohlcv.Highs, ohlcv.Lows, ohlcv.Closes, cfg.Indicators.Atr.Period)
					if len(atr) > 0 {
						atrMultiplier := cfg.ScalpMode.ATRMultiplier
						if atrMultiplier <= 0 {
							atrMultiplier = 1.5
						}
						atrPct := (atr[len(atr)-1] / price) * atrMultiplier * 100
						if atrPct > effectiveSL {
							dash.LogInfo(fmt.Sprintf("[yellow]ATR-SL[-] widened SL from %.2f%% to %.2f%% (ATR=%.8f, price=%.8f)",
								stopLoss, atrPct, atr[len(atr)-1], price))
							effectiveSL = atrPct
						}
					}
				}

				// fixed stop loss
				stopLossPrice := entryPrice * (1 - effectiveSL/100)
				if price <= stopLossPrice {
					dash.SetPhase("STOP LOSS")
					sell, err := TradeMarketSell(symbol, roundFloat(qty*0.998, roundAmount), price, roundPrice)
					if err != nil {
						dash.LogError(fmt.Sprintf("Stop-Loss MARKET SELL failed: %v", err))
						return
					}
					sellOrder := reflect.ValueOf(sell).Elem()
					orderId := sellOrder.FieldByName("OrderId").Int()
					dash.LogOrder(fmt.Sprintf("[red::b]STOP-LOSS MARKET SELL[-] %f %s @ [white::b]%.*f[-] %s (SL=%.2f%%)",
						qty, scoin, roundPrice, price, dcoin, effectiveSL))
					waitOrderFilled(dash, ticker, orderId, "[red::b]STOP-LOSS MARKET SELL[-] filled!", refreshInterval)
					exitType = "sl"
					break
				}

				// take profit
				profitPrice := entryPrice * (1 + takeProfit/100)
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
				rsiDeclining := rsi[len(rsi)-1] < rsi[len(rsi)-2]
				rsiExitOk := rsiDeclining || (cfg.ScalpMode.Enabled && !cfg.ScalpMode.RequireRSIExit)
				if price >= profitPrice && rsiExitOk && aiSellApproved {
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
					exitType = "tp"
					break
				}
				time.Sleep(refreshInterval)
			}

		} else {
			// BEAR exit phase
			dash.SetPhase("MONITORING BUY-BACK")
			lowestPrice := entryPrice
			sellProceeds := entryPrice * qty

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

				pnl := (entryPrice - price) / entryPrice * 100
				dash.UpdateIndicators(&tui.IndicatorData{
					RSI: rsi[len(rsi)-1], RSIUpperLimit: cfg.Indicators.Rsi.UpperLimit, RSILowerLimit: cfg.Indicators.Rsi.LowerLimit,
					Tendency: fmt.Sprintf("P&L: %+.2f%%", pnl),
				})

				if price < lowestPrice {
					lowestPrice = price
				}

				// trailing stop (inverse for bear)
				if cfg.TrailingStop.Enabled {
					activationPrice := entryPrice * (1 - cfg.TrailingStop.ActivationPct/100)
					if lowestPrice <= activationPrice {
						trailingStopPrice := lowestPrice * (1 + cfg.TrailingStop.TrailingPct/100)
						if price >= trailingStopPrice {
							dash.SetPhase("TRAILING STOP")
							buyBackQty := roundFloat(sellProceeds/price, roundAmount)
							buy, err := TradeMarketBuy(symbol, buyBackQty, price, roundPrice)
							if err != nil {
								dash.LogError(fmt.Sprintf("Trailing-Stop MARKET BUY failed: %v", err))
								return
							}
							buyOrder := reflect.ValueOf(buy).Elem()
							orderId := buyOrder.FieldByName("OrderId").Int()
							dash.LogOrder(fmt.Sprintf("[fuchsia::b]TRAILING-STOP MARKET BUY[-] %f %s @ ~[white::b]%.*f[-] %s",
								buyBackQty, scoin, roundPrice, price, dcoin))
							waitOrderFilled(dash, ticker, orderId, "[fuchsia::b]TRAILING-STOP MARKET BUY[-] filled!", refreshInterval)
							exitType = "ts"
							break
						}
					}
				}

				// ATR-based dynamic stop-loss
				effectiveSL := stopLoss
				if cfg.ScalpMode.ATRStopLoss && cfg.Indicators.Atr.Period > 0 {
					atr := calculateATR(ohlcv.Highs, ohlcv.Lows, ohlcv.Closes, cfg.Indicators.Atr.Period)
					if len(atr) > 0 {
						atrMultiplier := cfg.ScalpMode.ATRMultiplier
						if atrMultiplier <= 0 {
							atrMultiplier = 1.5
						}
						atrPct := (atr[len(atr)-1] / price) * atrMultiplier * 100
						if atrPct > effectiveSL {
							dash.LogInfo(fmt.Sprintf("[yellow]ATR-SL[-] widened SL from %.2f%% to %.2f%% (ATR=%.8f, price=%.8f)",
								stopLoss, atrPct, atr[len(atr)-1], price))
							effectiveSL = atrPct
						}
					}
				}

				// stop loss: price goes UP
				stopLossPrice := entryPrice * (1 + effectiveSL/100)
				if price >= stopLossPrice {
					dash.SetPhase("STOP LOSS")
					buyBackQty := roundFloat(sellProceeds/price, roundAmount)
					buy, err := TradeMarketBuy(symbol, buyBackQty, price, roundPrice)
					if err != nil {
						dash.LogError(fmt.Sprintf("Stop-Loss MARKET BUY failed: %v", err))
						return
					}
					buyOrder := reflect.ValueOf(buy).Elem()
					orderId := buyOrder.FieldByName("OrderId").Int()
					dash.LogOrder(fmt.Sprintf("[red::b]STOP-LOSS MARKET BUY[-] %f %s @ [white::b]%.*f[-] %s (SL=%.2f%%)",
						buyBackQty, scoin, roundPrice, price, dcoin, effectiveSL))
					waitOrderFilled(dash, ticker, orderId, "[red::b]STOP-LOSS MARKET BUY[-] filled!", refreshInterval)
					exitType = "sl"
					break
				}

				// take profit
				profitPrice := entryPrice * (1 - takeProfit/100)
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
				rsiRising := rsi[len(rsi)-1] > rsi[len(rsi)-2]
				rsiExitOk := rsiRising || (cfg.ScalpMode.Enabled && !cfg.ScalpMode.RequireRSIExit)
				if price <= profitPrice && rsiExitOk && aiBuyApproved {
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
					exitType = "tp"
					break
				}
				time.Sleep(refreshInterval)
			}
		}

		// Update consecutive SL counter and apply cooldown
		if exitType == "sl" {
			consecutiveSL++
			maxConsec := cfg.ScalpMode.MaxConsecutiveSL
			if maxConsec <= 0 {
				maxConsec = 2
			}
			if cfg.ScalpMode.SLCooldown && consecutiveSL >= maxConsec {
				baseSecs := cfg.ScalpMode.CooldownBaseSecs
				if baseSecs <= 0 {
					baseSecs = 60
				}
				exponent := consecutiveSL - maxConsec
				cooldown := baseSecs * (1 << exponent)
				if cooldown > 600 {
					cooldown = 600
				}
				dash.LogInfo(fmt.Sprintf("[red]SL COOLDOWN[-] %d consecutive SLs — waiting %ds before next entry", consecutiveSL, cooldown))
				time.Sleep(time.Duration(cooldown) * time.Second)
			}
		} else {
			consecutiveSL = 0
		}

		operation++
		interOpDelay := 60
		if cfg.ScalpMode.Enabled && cfg.ScalpMode.InterOpDelay > 0 {
			interOpDelay = cfg.ScalpMode.InterOpDelay
		}
		dash.LogInfo(fmt.Sprintf("Operation #%d complete. Next in %ds...", operation-1, interOpDelay))
		time.Sleep(time.Duration(interOpDelay) * time.Second)
	}
}
