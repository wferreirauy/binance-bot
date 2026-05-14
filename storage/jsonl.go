package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	TradesFile       = "trades.jsonl"
	ScoutsFile       = "scouts.jsonl"
	ValuesFile       = "values.jsonl"
	CurrentAssetFile = "current_asset.json"
)

type Store struct {
	dir string
}

type TradeRecord struct {
	Time           time.Time `json:"time"`
	Symbol         string    `json:"symbol"`
	Side           string    `json:"side"`
	OrderID        int64     `json:"order_id,omitempty"`
	Status         string    `json:"status"`
	Quantity       float64   `json:"quantity"`
	Price          float64   `json:"price"`
	QuoteQuantity  float64   `json:"quote_quantity,omitempty"`
	Reason         string    `json:"reason,omitempty"`
	Mode           string    `json:"mode,omitempty"`
	Operation      int       `json:"operation,omitempty"`
	ExecutedQty    float64   `json:"executed_qty,omitempty"`
	PartialHandled bool      `json:"partial_handled,omitempty"`
}

type ScoutRecord struct {
	Time          time.Time `json:"time"`
	FromAsset     string    `json:"from_asset"`
	ToAsset       string    `json:"to_asset"`
	BridgeAsset   string    `json:"bridge_asset"`
	FromPrice     float64   `json:"from_price"`
	ToPrice       float64   `json:"to_price"`
	BaselineRatio float64   `json:"baseline_ratio"`
	CurrentRatio  float64   `json:"current_ratio"`
	Opportunity   float64   `json:"opportunity"`
	TradeFeePct   float64   `json:"trade_fee_pct"`
	Selected      bool      `json:"selected"`
}

type ValueRecord struct {
	Time        time.Time `json:"time"`
	Asset       string    `json:"asset"`
	Balance     float64   `json:"balance"`
	BridgeAsset string    `json:"bridge_asset"`
	BridgeValue float64   `json:"bridge_value"`
}

func New(dir string) (*Store, error) {
	if dir == "" {
		dir = ".binance-bot"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("storage: create data dir: %w", err)
	}
	return &Store{dir: dir}, nil
}

func (s *Store) Dir() string {
	return s.dir
}

func (s *Store) AppendTrade(record TradeRecord) error {
	if record.Time.IsZero() {
		record.Time = time.Now()
	}
	return s.appendJSONL(TradesFile, record)
}

func (s *Store) AppendScout(record ScoutRecord) error {
	if record.Time.IsZero() {
		record.Time = time.Now()
	}
	return s.appendJSONL(ScoutsFile, record)
}

func (s *Store) AppendValue(record ValueRecord) error {
	if record.Time.IsZero() {
		record.Time = time.Now()
	}
	return s.appendJSONL(ValuesFile, record)
}

func (s *Store) SetCurrentAsset(asset string) error {
	payload := struct {
		Asset string    `json:"asset"`
		Time  time.Time `json:"time"`
	}{Asset: asset, Time: time.Now()}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("storage: encode current asset: %w", err)
	}
	return os.WriteFile(filepath.Join(s.dir, CurrentAssetFile), append(data, '\n'), 0o644)
}

func (s *Store) CurrentAsset() (string, bool, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, CurrentAssetFile))
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("storage: read current asset: %w", err)
	}
	payload := struct {
		Asset string `json:"asset"`
	}{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", false, fmt.Errorf("storage: decode current asset: %w", err)
	}
	return payload.Asset, payload.Asset != "", nil
}

func ReadJSONL[T any](dir, name string, limit int) ([]T, error) {
	path := filepath.Join(dir, name)
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return []T{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: open %s: %w", name, err)
	}
	defer file.Close()

	var records []T
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record T
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return nil, fmt.Errorf("storage: decode %s: %w", name, err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("storage: scan %s: %w", name, err)
	}
	if limit > 0 && len(records) > limit {
		return records[len(records)-limit:], nil
	}
	return records, nil
}

func (s *Store) appendJSONL(name string, record any) error {
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("storage: encode %s: %w", name, err)
	}
	file, err := os.OpenFile(filepath.Join(s.dir, name), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("storage: open %s: %w", name, err)
	}
	defer file.Close()
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("storage: append %s: %w", name, err)
	}
	return nil
}
