package main

import (
	"fmt"
	"testing"
)

func TestCalculateBollingerBands(t *testing.T) {
	prices := []float64{100, 102, 104, 103, 105, 108, 110, 112, 113, 115}
	period := 5
	multiplier := 2.0

	bands, err := CalculateBollingerBands(prices, period, multiplier)
	if err != nil {
		t.Fatalf("Error: %v", err)
	}

	if len(bands.UpperBand) == 0 || len(bands.MiddleBand) == 0 || len(bands.LowerBand) == 0 {
		t.Fatalf("Las bandas no se calcularon correctamente")
	}

	// Validar resultados esperados
	fmt.Println("Bollinger Bands:", bands)
}
