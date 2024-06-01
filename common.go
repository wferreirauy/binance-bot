package main

import (
	"math"
)

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

/* func sendAlert(message string) {
	// Implementa una función para enviar alertas, por ejemplo, usando un servicio de mensajería como Telegram o Slack
	log.Printf("Alerta: %s", message)
} */
