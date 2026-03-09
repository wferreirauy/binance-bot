package main

import (
	"math"
	"strconv"
)

// Returns a float rounded by the given precision
func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func parseFloat(value string) float64 {
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return parsed
}

// Implement a function to send alerts, for example, using a messaging service like Telegram or Slack.
/* func sendAlert(message string) {
	log.Printf("Alert: %s", message)
} */
