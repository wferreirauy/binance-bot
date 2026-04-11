package exchange

import "os"

var (
	APIKey    string = os.Getenv("BINANCE_API_KEY")
	SecretKey string = os.Getenv("BINANCE_SECRET_KEY")
	BaseURL   string = "https://api1.binance.com"
)
