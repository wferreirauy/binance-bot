package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	binance_connector "github.com/binance/binance-connector-go"
	color "github.com/fatih/color"
	"github.com/gosuri/uilive"
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
	maxOps uint,
) error {
	var c config.Config
	cfg, err := c.Read(configFile)
	if err != nil {
		return err
	}
	period := cfg.HistoricalPrices.Period
	interval := cfg.HistoricalPrices.Interval

	if err := validateBullTradeInputs(symbol, qty, stopLoss, takeProfit, buyFactor, sellFactor, maxOps); err != nil {
		return err
	}

	client := binance_connector.NewClient(apikey, secretkey, baseurl)
	ticker := strings.Replace(symbol, "/", "", -1)
	rules, err := fetchSymbolRules(context.Background(), ticker)
	if err != nil {
		return fmt.Errorf("validating symbol rules for %s: %w", symbol, err)
	}
	if roundPrice < decimalPlacesFromStep(rules.TickSize, rules.QuotePrecision) {
		log.Printf("round-price=%d is lower than Binance tick precision; using exchange tick size %.12f for validation", roundPrice, rules.TickSize)
	}
	if roundAmount < decimalPlacesFromStep(rules.StepSize, rules.BaseAssetPrecision) {
		log.Printf("round-amount=%d is lower than Binance step precision; using exchange step size %.12f for validation", roundAmount, rules.StepSize)
	}

	cyan := color.New(color.FgHiCyan, color.Bold).SprintFunc()
	red := color.New(color.FgHiRed, color.Bold).SprintFunc()
	green := color.New(color.FgHiGreen, color.Bold).SprintFunc()
	white := color.New(color.FgHiWhite, color.Bold).SprintFunc()

	scoin, dcoin, _ := strings.Cut(symbol, "/")
	buyPrice := 0.0
	operation := 1

	for range maxOps {
		cpw := uilive.New()
		cpw.Start()

		fmt.Println(white("Operation"), cyan("#"+strconv.Itoa(operation)))
		qty = roundFloat(qty, roundAmount)

		for {
			hp, err := getHistoricalPrices(client, ticker, interval, period)
			if err != nil {
				log.Printf("Error getting historical prices with %s interval: %v\n", interval, err)
				time.Sleep(defaultPollInterval)
				continue
			}
			if len(hp) < 2 {
				log.Printf("Not enough historical prices for %s: got %d\n", ticker, len(hp))
				time.Sleep(defaultPollInterval)
				continue
			}

			price := hp[len(hp)-1]
			prevPrice := hp[len(hp)-2]
			printPrice(cpw, symbol, price, prevPrice, roundPrice)

			tendency, err := getTendency(client, ticker, cfg.Tendency.Interval, period)
			if err != nil {
				log.Printf("Error getting tendency: %v\n", err)
				time.Sleep(defaultPollInterval)
				continue
			}

			dema := calculateDEMA(hp, cfg.Indicators.Dema.Length)
			rsi := calculateRSI(hp, cfg.Indicators.Rsi.Length)
			macdLine, signalLine := calculateMACD(hp, cfg.Indicators.Macd.FastLength, cfg.Indicators.Macd.SlowLength, cfg.Indicators.Macd.SignalLength)
			bb, err := CalculateBollingerBands(hp, cfg.Indicators.BollingerBands.Length, cfg.Indicators.BollingerBands.Multiplier)
			if err != nil {
				log.Printf("Error getting BollingerBands: %v\n", err)
				time.Sleep(defaultPollInterval)
				continue
			}
			if len(dema) == 0 || len(rsi) < 1 || len(macdLine) < 2 || len(signalLine) < 2 || len(bb.LowerBand) == 0 || len(bb.UpperBand) == 0 {
				log.Printf("Indicators not ready yet for %s\n", ticker)
				time.Sleep(defaultPollInterval)
				continue
			}

			currentDema := dema[len(dema)-1]
			lowerBand := bb.LowerBand[len(bb.LowerBand)-1]
			upperBand := bb.UpperBand[len(bb.UpperBand)-1]
			distanceToUpper := math.Abs(currentDema - upperBand)
			distanceToLower := math.Abs(currentDema - lowerBand)

			if rsi[len(rsi)-1] < float64(cfg.Indicators.Rsi.UpperLimit) &&
				macdLine[len(macdLine)-2] <= signalLine[len(signalLine)-2] &&
				macdLine[len(macdLine)-1] > signalLine[len(signalLine)-1] &&
				tendency == cfg.Tendency.Direction &&
				distanceToLower < distanceToUpper {
				buy, err := TradeBuy(context.Background(), symbol, qty, price, buyFactor, roundPrice)
				if err != nil {
					return fmt.Errorf("creating BUY order: %w", err)
				}
				buyOrder := reflect.ValueOf(buy).Elem()
				orderID := buyOrder.FieldByName("OrderId").Int()
				orderPrice := buyOrder.FieldByName("Price").String()
				buyPrice, err = strconv.ParseFloat(orderPrice, 64)
				if err != nil {
					return fmt.Errorf("could not convert buy order price to float: %w", err)
				}

				fmt.Printf("%s %s %f %s - PRICE: %s - Total %s: %f\n",
					time.Now().Format("02/01/2006 15:04:05"), green("BUY"), qty, scoin, white(buyPrice), dcoin, buyPrice*qty)

				if getor, err := GetOrder(context.Background(), ticker, orderID); err == nil {
					fmt.Printf("%s BUY order created. Id: %d - Status: %s\n",
						time.Now().Format("02/01/2006 15:04:05"), getor.OrderId, getor.Status)
				}

				if _, err := WaitForOrderFillOrCancel(context.Background(), ticker, orderID, defaultOrderFillWait); err != nil {
					return fmt.Errorf("buy order %d was not filled safely: %w", orderID, err)
				}
				fmt.Printf("%s BUY order filled!\n\n", time.Now().Format("02/01/2006 15:04:05"))
				break
			}

			time.Sleep(defaultPollInterval)
		}
		cpw.Stop()
		time.Sleep(30 * time.Second)

		cpw.Start()
		for {
			hp, err := getHistoricalPrices(client, ticker, interval, period)
			if err != nil {
				log.Printf("Error getting historical prices with %s interval: %v\n", interval, err)
				time.Sleep(defaultPollInterval)
				continue
			}
			rsiprices, err := getHistoricalPrices(client, ticker, cfg.Indicators.Rsi.Interval, period)
			if err != nil {
				log.Printf("Error getting historical prices with %s interval: %v\n", interval, err)
				time.Sleep(defaultPollInterval)
				continue
			}
			if len(hp) < 2 || len(rsiprices) == 0 {
				log.Printf("Not enough sell-side data for %s\n", ticker)
				time.Sleep(defaultPollInterval)
				continue
			}

			price := hp[len(hp)-1]
			prevPrice := hp[len(hp)-2]
			rsi := calculateRSI(rsiprices, cfg.Indicators.Rsi.Length)
			if len(rsi) < 2 {
				log.Printf("RSI not ready yet for %s\n", ticker)
				time.Sleep(defaultPollInterval)
				continue
			}

			printPrice(cpw, symbol, price, prevPrice, roundPrice)

			stopLossPrice := buyPrice * (1 - stopLoss/100)
			if price <= stopLossPrice {
				sell, err := TradeStopLossSell(context.Background(), symbol, roundFloat(qty*0.998, roundAmount))
				if err != nil {
					return fmt.Errorf("creating stop-loss market sell for amount %f: %w", roundFloat(qty*0.998, roundAmount), err)
				}
				sellOrder := reflect.ValueOf(sell).Elem()
				orderID := sellOrder.FieldByName("OrderId").Int()

				fmt.Printf("%s %s %f %s - PRICE: %s - Total %s: %f\n",
					time.Now().Format("02/01/2006 15:04:05"), red("SELL"), qty, scoin, white(price), dcoin, price*qty)

				if getor, err := GetOrder(context.Background(), ticker, orderID); err == nil {
					fmt.Printf("%s %s order created. Id: %d - Status: %s\n",
						time.Now().Format("02/01/2006 15:04:05"), red("STOP-LOSS MARKET SELL"), getor.OrderId, getor.Status)
				}
				if _, err := WaitForOrderFill(context.Background(), ticker, orderID, 30*time.Second); err != nil {
					return fmt.Errorf("stop-loss market sell %d did not finish cleanly: %w", orderID, err)
				}
				fmt.Printf("%s %s order filled!\n\n", time.Now().Format("02/01/2006 15:04:05"), red("STOP-LOSS MARKET SELL"))
				break
			}

			profitPrice := buyPrice * (1 + takeProfit/100)
			if price >= profitPrice && rsi[len(rsi)-1] < rsi[len(rsi)-2] {
				sell, err := TradeSell(context.Background(), symbol, roundFloat(qty*0.998, roundAmount), price, sellFactor, roundPrice)
				if err != nil {
					return fmt.Errorf("creating SELL order with amount %f: %w", roundFloat(qty*0.998, roundAmount), err)
				}
				sellOrder := reflect.ValueOf(sell).Elem()
				orderID := sellOrder.FieldByName("OrderId").Int()

				fmt.Printf("%s %s %f %s - PRICE: %s - Total %s: %f\n",
					time.Now().Format("02/01/2006 15:04:05"), red("SELL"), qty, scoin, white(price), dcoin, price*qty)

				if getor, err := GetOrder(context.Background(), ticker, orderID); err == nil {
					fmt.Printf("%s SELL order created. Id: %d - Status: %s\n",
						time.Now().Format("02/01/2006 15:04:05"), getor.OrderId, getor.Status)
				}

				if _, err := WaitForOrderFillOrCancel(context.Background(), ticker, orderID, defaultOrderFillWait); err != nil {
					return fmt.Errorf("sell order %d was not filled safely: %w", orderID, err)
				}
				fmt.Printf("%s SELL order filled!\n\n", time.Now().Format("02/01/2006 15:04:05"))
				break
			}
			time.Sleep(defaultPollInterval)
		}
		cpw.Stop()
		operation++
		time.Sleep(1 * time.Minute)
	}

	return nil
}

func validateBullTradeInputs(symbol string, qty, stopLoss, takeProfit, buyFactor, sellFactor float64, maxOps uint) error {
	if re := regexp.MustCompile(`(?m)^[0-9A-Z]{2,8}/[0-9A-Z]{2,8}$`); !re.Match([]byte(symbol)) {
		return fmt.Errorf("error parsing ticker: must match ^[0-9A-Z]{2,8}/[0-9A-Z]{2,8}$")
	}
	if qty <= 0 {
		return fmt.Errorf("amount must be greater than zero")
	}
	if stopLoss <= 0 || stopLoss >= 100 {
		return fmt.Errorf("stop-loss must be between 0 and 100")
	}
	if takeProfit <= 0 {
		return fmt.Errorf("take-profit must be greater than zero")
	}
	if buyFactor <= 0 {
		return fmt.Errorf("buy-factor must be greater than zero")
	}
	if sellFactor <= 0 {
		return fmt.Errorf("sell-factor must be greater than zero")
	}
	if maxOps == 0 {
		return fmt.Errorf("operations must be greater than zero")
	}
	return nil
}
