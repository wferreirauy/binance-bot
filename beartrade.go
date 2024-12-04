package main

import (
	"log"
	"regexp"
	"strings"
	"time"

	binance_connector "github.com/binance/binance-connector-go"
)

func BearTrade(symbol string, qty, stopLoss, takeProfit, sellFactor, buyFactor float64, roundPrice, roundAmount, max_ops uint) {
	client := binance_connector.NewClient(apikey, secretkey, baseurl)

	// Validar símbolo
	if re := regexp.MustCompile(`(?m)^[0-9A-Z]{2,8}/[0-9A-Z]{2,8}$`); !re.Match([]byte(symbol)) {
		log.Fatal("error parsing ticker: must match ^[0-9A-Z]{2,8}/[0-9A-Z]{2,8}$")
	}
	ticker := strings.Replace(symbol, "/", "", -1)

	var sellPrice, buyPrice float64
	operation := 0

	for range max_ops {
		hp, err := getHistoricalPrices(client, ticker, interval, period)
		if err != nil {
			log.Printf("Error getting historical prices: %v\n", err)
			time.Sleep(10 * time.Second)
			continue
		}

		// Calcular Bollinger Bands
		bands, err := CalculateBollingerBands(hp, period, 2.0)
		if err != nil {
			log.Printf("Error calculating Bollinger Bands: %v\n", err)
			time.Sleep(10 * time.Second)
			continue
		}

		price := hp[len(hp)-1]
		upperBand := bands.UpperBand[len(bands.UpperBand)-1]
		lowerBand := bands.LowerBand[len(bands.LowerBand)-1]

		// Primera operación: Venta
		if operation == 0 && price >= upperBand {
			log.Printf("Selling at price: %f (near upper band: %f)", price, upperBand)
			sellPrice = price
			// Aquí realizarías la orden de venta real
			operation++
		}

		// Segunda operación: Recompra
		if operation == 1 && price <= lowerBand {
			log.Printf("Buying back at price: %f (below lower band: %f)", price, lowerBand)
			buyPrice = price
			profit := sellPrice - buyPrice
			log.Printf("Profit: %f", profit)
			operation++
		}

		// Terminar si se excede el número máximo de operaciones
		if operation >= int(max_ops) {
			break
		}

		time.Sleep(1 * time.Second)
	}
}
