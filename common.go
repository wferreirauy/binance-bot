package main

import (
	"math"
)

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}

/* func sendAlert(message string) {
	// Implementa una función para enviar alertas, por ejemplo, usando un servicio de mensajería como Telegram o Slack
	log.Printf("Alerta: %s", message)
} */
