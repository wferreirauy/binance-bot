package config

import (
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	HistoricalPrices struct {
		Period   int    `yaml:"period"`
		Interval string `yaml:"interval"`
	} `yaml:"historical-prices"`
	Tendency struct {
		Interval  string `yaml:"interval"`
		Direction string `yaml:"direction"`
	} `yaml:"tendency"`
	Indicators struct {
		Rsi struct {
			Interval    string `yaml:"interval"`
			Length      int    `yaml:"length"`
			UpperLimit  int    `yaml:"upper-limit"`
			MiddleLimit int    `yaml:"middle-limit"`
			LowerLimit  int    `yaml:"lower-limit"`
		} `yaml:"rsi"`
		Dema struct {
			Length int `yaml:"length"`
		} `yaml:"dema"`
		Macd struct {
			FastLength   int `yaml:"fast-length"`
			SlowLength   int `yaml:"slow-length"`
			SignalLength int `yaml:"signal-length"`
		} `yaml:"macd"`
		BollingerBands struct {
			Length     int     `yaml:"length"`
			Multiplier float64 `yaml:"multiplier"`
		} `yaml:"bollinger-bands"`
	} `yaml:"indicators"`
}

func (c *Config) Read(filePath string) (*Config, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("Config: could not open config file: %w", err)
	}
	defer f.Close()
	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("Config: could not decode the config file: %w", err)
	}
	return &cfg, nil
}
