package config

import (
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	HistoricalPrices struct {
		Period   uint16 `yaml:"period"`
		Interval string `yaml:"interval"`
	} `yaml:"historical-prices"`
	Tendency struct {
		Interval string `yaml:"interval"`
	} `yaml:"tendency"`
	Indicators struct {
		Rsi struct {
			Interval string `yaml:"interval"`
			Length   uint16 `yaml:"length"`
		} `yaml:"rsi"`
		Dema struct {
			Length uint16 `yaml:"length"`
		} `yaml:"dema"`
	} `yaml:"indicators"`
}

func (c *Config) Read() (*Config, error) {
	f, err := os.Open("config.yml")
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
