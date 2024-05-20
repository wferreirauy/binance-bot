package main

import (
	"fmt"
	"os"
	"time"
)

var apikey string = os.Getenv("BINANCE_API_KEY")
var secretkey string = os.Getenv("BINANCE_SECRET_KEY")
var baseurl string = "https://api1.binance.com"

func main() {
	for range 50 {
		go func() {
			if sol_price, err := GetPrice("SOLUSDT"); err == nil {
				fmt.Println(sol_price)
			}
		}()
		time.Sleep(5 * time.Second)
	}
	select {}

}
