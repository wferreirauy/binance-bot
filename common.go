package main

import (
	"math"
)

// Returns a float rounded by the given precision
func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

// Implement a function to send alerts, for example, using a messaging service like Telegram or Slack.
/* func sendAlert(message string) {
	log.Printf("Alert: %s", message)
} */
