package config

type ConfigFile struct {
	Filename     string `json:"-"` // Note: for internal use only
	PeriodTime   string `json:"period-time,omitempty"`
	PeriodLength string `json:"period-length,omitempty"`
	StopLoss     bool   `json:"stoploss,omitempty"`
}

// GetFilename returns the file name that this config file is based on.
func (configFile *ConfigFile) GetFilename() string {
	return configFile.Filename
}

// func (configFile *ConfigFile) LoadFromReader(configData io.Reader) (map[string]any, error) {
// 	if err := json.NewDecoder(configData).Decode(configFile); err != nil && !errors.Is(err, io.EOF) {
// 		return nil, err
// 	}
// 	var configs = make(map[string]any)
// 	configs["pt"] = append(configs, configFile.PeriodTime)
// 	return configs, nil
// }
