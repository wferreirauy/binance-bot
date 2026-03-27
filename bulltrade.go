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
	color "github.com/fatih/color"
	"github.com/gosuri/uilive"
	"github.com/wferreirauy/binance-bot/ai"
	"github.com/wferreirauy/binance-bot/config"
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

	// define text colors
	cyan := color.New(color.FgHiCyan, color.Bold).SprintFunc()
	red := color.New(color.FgHiRed, color.Bold).SprintFunc()
	green := color.New(color.FgHiGreen, color.Bold).SprintFunc()
	white := color.New(color.FgHiWhite, color.Bold).SprintFunc()
	magenta := color.New(color.FgHiMagenta, color.Bold).SprintFunc()

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
		if aiOrch.IsEnabled() {
			fmt.Println(white("AI Agents:"), green("ENABLED"))
		} else {
			fmt.Println(white("AI Agents:"), red("No API keys found - running without AI"))
			aiOrch = nil
		}
	}

	var buyPrice float64
	var operation = 1

	for range max_ops {
		// set tui writers
		cpw := uilive.New() // current price line writer
		cpw.Start()

		fmt.Println(white("Operation"), cyan("#"+strconv.Itoa(operation)))
		qty = roundFloat(qty, roundAmount)

		//// buy ////
		for {
			// get historical OHLCV data
			ohlcv, err := getHistoricalOHLCV(client, ticker, interval, period)
			if err != nil {
				log.Printf("Error getting historical OHLCV data with %s interval: %v\n", interval, err)
				time.Sleep(10 * time.Second)
				continue
			}

			price := ohlcv.Closes[len(ohlcv.Closes)-1]
			prevPrice := ohlcv.Closes[len(ohlcv.Closes)-2]

			// print current price
			printPrice(cpw, symbol, price, prevPrice, roundPrice)

			// indicators
			// tendency "up" or "down"
			tendency, err := getTendency(client, ticker, cfg.Tendency.Interval, period)
			if err != nil {
				log.Printf("Error getting tendency: %v\n", err)
				time.Sleep(10 * time.Second)
				continue
			}
			// dema
			dema := calculateDEMA(ohlcv.Closes, cfg.Indicators.Dema.Length)
			currentDema := dema[len(dema)-1]
			// rsi
			rsi := calculateRSI(ohlcv.Closes, cfg.Indicators.Rsi.Length)
			// macd
			macdLine, signalLine := calculateMACD(
				ohlcv.Closes,
				cfg.Indicators.Macd.FastLength,
				cfg.Indicators.Macd.SlowLength,
				cfg.Indicators.Macd.SignalLength,
			)
			// bollingerbands
			bb, err := CalculateBollingerBands(
				ohlcv.Closes,
				cfg.Indicators.BollingerBands.Length,
				cfg.Indicators.BollingerBands.Multiplier,
			)
			if err != nil {
				log.Printf("Error getting BollingerBands: %v\n", err)
			}
			lowerBand := bb.LowerBand[len(bb.LowerBand)-1]
			upperBand := bb.UpperBand[len(bb.UpperBand)-1]
			distanceToUpper := math.Abs(currentDema - upperBand)
			distanceToLower := math.Abs(currentDema - lowerBand)

			// adx - trend strength (passes through if not configured)
			var adxStrong bool
			if cfg.Indicators.Adx.Period > 0 {
				adx := calculateADX(ohlcv.Highs, ohlcv.Lows, ohlcv.Closes, cfg.Indicators.Adx.Period)
				adxStrong = len(adx) == 0 || adx[len(adx)-1] > float64(cfg.Indicators.Adx.Threshold)
			} else {
				adxStrong = true
			}

			// volume confirmation (passes through if not configured)
			var volumeConfirmed bool
			if cfg.Indicators.Volume.MaPeriod > 0 {
				volumeMA := calculateSMA(ohlcv.Volumes, cfg.Indicators.Volume.MaPeriod)
				currentVolume := ohlcv.Volumes[len(ohlcv.Volumes)-1]
				volumeConfirmed = len(volumeMA) == 0 || currentVolume > volumeMA[len(volumeMA)-1]
			} else {
				volumeConfirmed = true
			}

			// AI analysis (if enabled)
			var aiApproved = true
			if aiOrch != nil {
				snapshot := &ai.TechnicalSnapshot{
					Symbol:         symbol,
					Price:          price,
					PrevPrice:      prevPrice,
					RSI:            rsi[len(rsi)-1],
					MACDLine:       macdLine[len(macdLine)-1],
					SignalLine:     signalLine[len(signalLine)-1],
					PrevMACDLine:   macdLine[len(macdLine)-2],
					PrevSignalLine: signalLine[len(signalLine)-2],
					UpperBand:      upperBand,
					LowerBand:      lowerBand,
					DEMA:           currentDema,
					Tendency:       tendency,
				}
				if cfg.Indicators.Adx.Period > 0 {
					adxVals := calculateADX(ohlcv.Highs, ohlcv.Lows, ohlcv.Closes, cfg.Indicators.Adx.Period)
					if len(adxVals) > 0 {
						snapshot.ADX = adxVals[len(adxVals)-1]
					}
				}
				if cfg.Indicators.Volume.MaPeriod > 0 {
					volumeMA := calculateSMA(ohlcv.Volumes, cfg.Indicators.Volume.MaPeriod)
					snapshot.Volume = ohlcv.Volumes[len(ohlcv.Volumes)-1]
					if len(volumeMA) > 0 {
						snapshot.AvgVolume = volumeMA[len(volumeMA)-1]
					}
				}

				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				consensus, err := aiOrch.Analyze(ctx, snapshot, "BULL")
				cancel()
				if err != nil {
					log.Printf("AI analysis error: %v\n", err)
				} else {
					fmt.Print(consensus.String())
					minConf := cfg.AI.MinConfidence
					if minConf <= 0 {
						minConf = 0.5
					}
					aiApproved = consensus.ShouldBuy() || consensus.FinalSignal == ai.SignalHold
				}
			}

			// when to buy
			if rsi[len(rsi)-1] < float64(cfg.Indicators.Rsi.UpperLimit) && // RSI below upper limit
				macdLine[len(macdLine)-2] <= signalLine[len(signalLine)-2] &&
				macdLine[len(macdLine)-1] > signalLine[len(signalLine)-1] && // MACD crosses signal
				tendency == cfg.Tendency.Direction && // tendency
				distanceToLower < distanceToUpper && // dema closer than lower band
				adxStrong && // trend has strength
				volumeConfirmed && // volume above average
				aiApproved { // AI consensus supports entry
				buy, err := TradeBuy(symbol, qty, price, buyFactor, roundPrice)
				if err != nil {
					log.Fatalf("error creating BUY order: %s\n", err)
				}
				buyOrder := reflect.ValueOf(buy).Elem()
				orderId := buyOrder.FieldByName("OrderId").Int()
				orderPrice := buyOrder.FieldByName("Price").String()
				buyPrice, err = strconv.ParseFloat(orderPrice, 64)
				if err != nil {
					log.Printf("could not convert price on buy order to float: %s\n", err)
				}

				fmt.Printf("%s %s %f %s - PRICE: %s - Total %s: %f\n",
					time.Now().Format("02/01/2006 15:04:05"), green("BUY"), qty, scoin, white(buyPrice), dcoin, buyPrice*qty)

				if getor, err := GetOrder(ticker, orderId); err == nil {
					fmt.Printf("%s BUY order created. Id: %d - Status: %s\n",
						time.Now().Format("02/01/2006 15:04:05"), getor.OrderId, getor.Status)
				}

				for { // looking at buy order until is filled
					if getor, err := GetOrder(ticker, orderId); err == nil {
						if getor.Status == "FILLED" {
							fmt.Printf("%s BUY order filled!\n\n", time.Now().Format("02/01/2006 15:04:05"))
							break // buy filled
						}
					}
					time.Sleep(refreshInterval)
				}
				break // indicators conditions met
			}

			time.Sleep(refreshInterval)
		}
		cpw.Stop()
		time.Sleep(30 * time.Second) // sleep before start selling process

		//// sell ////
		cpw.Start()
		highestPrice := buyPrice // track highest price for trailing stop

		for {
			ohlcv, err := getHistoricalOHLCV(client, ticker, interval, period)
			if err != nil {
				log.Printf("Error getting historical OHLCV data with %s interval: %v\n", interval, err)
				time.Sleep(10 * time.Second)
				continue
			}
			rsiprices, err := getHistoricalPrices(client, ticker, cfg.Indicators.Rsi.Interval, period)
			if err != nil {
				log.Printf("Error getting historical prices with %s interval: %v\n", cfg.Indicators.Rsi.Interval, err)
				time.Sleep(10 * time.Second)
				continue
			}

			price := ohlcv.Closes[len(ohlcv.Closes)-1]
			prevPrice := ohlcv.Closes[len(ohlcv.Closes)-2]
			rsi := calculateRSI(rsiprices, cfg.Indicators.Rsi.Length)

			// print current price
			printPrice(cpw, symbol, price, prevPrice, roundPrice)

			// update highest price for trailing stop
			if price > highestPrice {
				highestPrice = price
			}

			// trailing stop-loss: locks in profit once price moves favorably
			if cfg.TrailingStop.Enabled {
				activationPrice := buyPrice * (1 + cfg.TrailingStop.ActivationPct/100)
				if highestPrice >= activationPrice {
					trailingStopPrice := highestPrice * (1 - cfg.TrailingStop.TrailingPct/100)
					if price <= trailingStopPrice {
						sell, err := TradeSell(symbol, roundFloat(qty*0.998, roundAmount), price, 1.0, roundPrice)
						if err != nil {
							log.Fatalf("error creating Trailing-Stop SELL order with amount %f: %s\n",
								roundFloat(qty*0.998, roundAmount), err)
						}
						sellOrder := reflect.ValueOf(sell).Elem()
						orderId := sellOrder.FieldByName("OrderId").Int()

						fmt.Printf("%s %s %f %s - PRICE: %s - Total %s: %f\n",
							time.Now().Format("02/01/2006 15:04:05"), magenta("TRAILING-STOP SELL"), qty, scoin, white(price), dcoin, price*qty)

						if getor, err := GetOrder(ticker, orderId); err == nil {
							fmt.Printf("%s %s order created. Id: %d - Status: %s\n",
								time.Now().Format("02/01/2006 15:04:05"), magenta("TRAILING-STOP SELL"), getor.OrderId, getor.Status)
						}

						for {
							if getor, err := GetOrder(ticker, orderId); err == nil {
								if getor.Status == "FILLED" {
									fmt.Printf("%s %s order filled!\n\n", time.Now().Format("02/01/2006 15:04:05"), magenta("TRAILING-STOP SELL"))
									break
								}
							}
							time.Sleep(refreshInterval)
						}
						break // sold (trailing stop)
					}
				}
			}

			// fixed stop loss
			stopLossPrice := buyPrice * (1 - stopLoss/100)
			if price <= stopLossPrice {
				sell, err := TradeSell(symbol, roundFloat(qty*0.998, roundAmount), price, 1.0, roundPrice)
				if err != nil {
					log.Fatalf("error creating Stop-Loss SELL order with amount %f: %s\n",
						roundFloat(qty*0.998, roundAmount), err)
				}
				sellOrder := reflect.ValueOf(sell).Elem()
				orderId := sellOrder.FieldByName("OrderId").Int()

				fmt.Printf("%s %s %f %s - PRICE: %s - Total %s: %f\n",
					time.Now().Format("02/01/2006 15:04:05"), red("SELL"), qty, scoin, white(price), dcoin, price*qty)

				if getor, err := GetOrder(ticker, orderId); err == nil {
					fmt.Printf("%s %s order created. Id: %d - Status: %s\n",
						time.Now().Format("02/01/2006 15:04:05"), red("STOP-LOSS SELL"), getor.OrderId, getor.Status)
				}
				for { // looking at sell order until is filled
					if getor, err := GetOrder(ticker, orderId); err == nil {
						if getor.Status == "FILLED" {
							fmt.Printf("%s %s order filled!\n\n", time.Now().Format("02/01/2006 15:04:05"), red("STOP-LOSS SELL"))
							break // sell filled
						}
					}
					time.Sleep(refreshInterval)
				}
				break // sold (stop loss)
			}

			// take profit with AI exit confirmation
			profitPrice := buyPrice * (1 + takeProfit/100)
			var aiSellApproved = true
			if price >= profitPrice && aiOrch != nil {
				snapshot := &ai.TechnicalSnapshot{
					Symbol:    symbol,
					Price:     price,
					PrevPrice: prevPrice,
					RSI:       rsi[len(rsi)-1],
					Tendency:  "sell-exit",
				}
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				consensus, err := aiOrch.Analyze(ctx, snapshot, "BULL")
				cancel()
				if err != nil {
					log.Printf("AI sell analysis error: %v\n", err)
				} else {
					fmt.Print(consensus.String())
					aiSellApproved = consensus.ShouldSell() || consensus.FinalSignal == ai.SignalHold
				}
			}
			if price >= profitPrice && // price reach take profit percentage
				rsi[len(rsi)-1] < rsi[len(rsi)-2] && // and rsi turns down
				aiSellApproved { // AI supports exit
				sell, err := TradeSell(symbol, roundFloat(qty*0.998, roundAmount), price, sellFactor, roundPrice)
				if err != nil {
					log.Fatalf("error creating SELL order with amount %f: %s\n",
						roundFloat(qty*0.998, roundAmount), err)
				}
				sellOrder := reflect.ValueOf(sell).Elem()
				orderId := sellOrder.FieldByName("OrderId").Int()

				fmt.Printf("%s %s %f %s - PRICE: %s - Total %s: %f\n",
					time.Now().Format("02/01/2006 15:04:05"), red("SELL"), qty, scoin, white(price), dcoin, price*qty)

				if getor, err := GetOrder(ticker, orderId); err == nil {
					fmt.Printf("%s SELL order created. Id: %d - Status: %s\n",
						time.Now().Format("02/01/2006 15:04:05"), getor.OrderId, getor.Status)
				}

				for { // looking at sell order until is filled
					if getor, err := GetOrder(ticker, orderId); err == nil {
						if getor.Status == "FILLED" {
							fmt.Printf("%s SELL order filled!\n\n", time.Now().Format("02/01/2006 15:04:05"))
							break // sell filled
						}
					}
					time.Sleep(refreshInterval)
				}
				break // sold
			}
			time.Sleep(refreshInterval)
		}
		cpw.Stop()
		operation++
		time.Sleep(1 * time.Minute) // 1 minute to start next operation
	}
}
