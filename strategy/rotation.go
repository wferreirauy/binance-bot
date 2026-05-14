package strategy

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	binance_connector "github.com/binance/binance-connector-go"
	"github.com/wferreirauy/binance-bot/config"
	"github.com/wferreirauy/binance-bot/exchange"
	"github.com/wferreirauy/binance-bot/storage"
)

type rotationPair struct {
	FromAsset     string
	ToAsset       string
	BaselineRatio float64
}

func RotationTrade(configFile string) {
	var c config.Config
	cfg, err := c.Read(configFile)
	if err != nil {
		log.Fatal(err)
	}
	if cfg.BaseURL != "" {
		exchange.BaseURL = cfg.BaseURL
	}
	if err := runRotation(cfg); err != nil {
		log.Fatal(err)
	}
}

func runRotation(cfg *config.Config) error {
	store, err := storage.New(cfg.DataDir)
	if err != nil {
		return err
	}
	bridge := strings.ToUpper(cfg.Rotation.BridgeAsset)
	assets := normalizedAssets(cfg.Rotation.SupportedAssets, bridge)
	if len(assets) == 0 {
		return fmt.Errorf("rotation: supported-assets is empty")
	}
	currentAsset := strings.ToUpper(cfg.Rotation.CurrentAsset)
	if persisted, ok, err := store.CurrentAsset(); err != nil {
		return err
	} else if ok {
		currentAsset = strings.ToUpper(persisted)
	}
	if currentAsset == "" {
		currentAsset = assets[0]
	}
	if err := store.SetCurrentAsset(currentAsset); err != nil {
		return err
	}
	client := binance_connector.NewClient(exchange.APIKey, exchange.SecretKey, exchange.BaseURL)
	baselines := map[string]rotationPair{}
	jumps := 0
	sleep := time.Duration(cfg.Rotation.ScoutSleepSeconds) * time.Second
	if sleep <= 0 {
		sleep = time.Second
	}

	fmt.Printf("Rotation scout started: current=%s bridge=%s assets=%v dry-run=%v\n", currentAsset, bridge, assets, cfg.Rotation.DryRun)
	for {
		prices, err := exchange.GetAllTickerPrices(client)
		if err != nil {
			return err
		}
		if len(baselines) == 0 {
			initializeRotationBaselines(baselines, assets, bridge, prices)
		}
		best, ok := bestRotationCandidate(cfg, store, currentAsset, assets, bridge, baselines, prices)
		if ok {
			fmt.Printf("Selected rotation %s -> %s through %s (opportunity %.8f)\n", currentAsset, best.ToAsset, bridge, best.Opportunity)
			if err := executeRotation(cfg, store, currentAsset, best.ToAsset, bridge, prices); err != nil {
				return err
			}
			currentAsset = best.ToAsset
			jumps++
			initializeRotationBaselines(baselines, assets, bridge, prices)
			if cfg.Rotation.MaxJumps > 0 && jumps >= cfg.Rotation.MaxJumps {
				return nil
			}
		}
		time.Sleep(sleep)
	}
}

type rotationCandidate struct {
	ToAsset     string
	Opportunity float64
}

func bestRotationCandidate(cfg *config.Config, store *storage.Store, currentAsset string, assets []string, bridge string, baselines map[string]rotationPair, prices map[string]float64) (rotationCandidate, bool) {
	fromPrice := prices[currentAsset+bridge]
	if fromPrice <= 0 {
		return rotationCandidate{}, false
	}
	defaultFee := cfg.Fees.DefaultTakerPct
	if defaultFee <= 0 {
		defaultFee = 0.1
	}
	tradeFeePct := defaultFee * 2
	var best rotationCandidate
	for _, asset := range assets {
		if asset == currentAsset {
			continue
		}
		toPrice := prices[asset+bridge]
		if toPrice <= 0 {
			continue
		}
		key := currentAsset + ">" + asset
		pair, ok := baselines[key]
		if !ok || pair.BaselineRatio <= 0 {
			continue
		}
		currentRatio := fromPrice / toPrice
		var opportunity float64
		if cfg.Rotation.UseMargin {
			opportunity = (1-tradeFeePct/100)*currentRatio/pair.BaselineRatio - 1 - cfg.Rotation.ScoutMarginPct/100
		} else {
			multiplier := cfg.Rotation.ScoutMultiplier
			if multiplier <= 0 {
				multiplier = 5
			}
			opportunity = currentRatio - (tradeFeePct/100)*multiplier*currentRatio - pair.BaselineRatio
		}
		selected := opportunity > 0 && opportunity > best.Opportunity
		_ = store.AppendScout(storage.ScoutRecord{
			FromAsset:     currentAsset,
			ToAsset:       asset,
			BridgeAsset:   bridge,
			FromPrice:     fromPrice,
			ToPrice:       toPrice,
			BaselineRatio: pair.BaselineRatio,
			CurrentRatio:  currentRatio,
			Opportunity:   opportunity,
			TradeFeePct:   tradeFeePct,
			Selected:      selected,
		})
		if selected {
			best = rotationCandidate{ToAsset: asset, Opportunity: opportunity}
		}
	}
	return best, best.ToAsset != ""
}

func executeRotation(cfg *config.Config, store *storage.Store, fromAsset, toAsset, bridge string, prices map[string]float64) error {
	if cfg.Rotation.DryRun {
		_ = store.SetCurrentAsset(toAsset)
		return store.AppendTrade(storage.TradeRecord{
			Symbol: fromAsset + bridge + ">" + toAsset + bridge,
			Side:   "ROTATE",
			Status: "DRY_RUN",
			Reason: "rotation-scout",
			Mode:   "rotation",
		})
	}
	if fromAsset != bridge {
		sellSymbol := fromAsset + bridge
		balance, err := exchange.GetBalance(fromAsset)
		if err != nil {
			return err
		}
		if balance <= 0 {
			return fmt.Errorf("rotation: no %s balance to sell", fromAsset)
		}
		if _, err := TradeMarketSell(sellSymbol, balance, prices[sellSymbol], 8); err != nil {
			return err
		}
		_ = store.AppendTrade(storage.TradeRecord{Symbol: sellSymbol, Side: "SELL", Status: "SUBMITTED", Quantity: balance, Price: prices[sellSymbol], Reason: "rotation-scout", Mode: "rotation"})
	}
	bridgeBalance, err := exchange.GetBalance(bridge)
	if err != nil {
		return err
	}
	buySymbol := toAsset + bridge
	price := prices[buySymbol]
	if price <= 0 {
		return fmt.Errorf("rotation: missing price for %s", buySymbol)
	}
	buffer := cfg.Rotation.MinNotionalBuffer
	if buffer <= 0 {
		buffer = 1.01
	}
	qty := math.Floor((bridgeBalance/price)/buffer*1e8) / 1e8
	if qty <= 0 {
		return fmt.Errorf("rotation: bridge balance %.8f is too small to buy %s", bridgeBalance, toAsset)
	}
	if _, err := TradeMarketBuy(buySymbol, qty, price, 8); err != nil {
		return err
	}
	_ = store.SetCurrentAsset(toAsset)
	return store.AppendTrade(storage.TradeRecord{Symbol: buySymbol, Side: "BUY", Status: "SUBMITTED", Quantity: qty, Price: price, QuoteQuantity: bridgeBalance, Reason: "rotation-scout", Mode: "rotation"})
}

func initializeRotationBaselines(out map[string]rotationPair, assets []string, bridge string, prices map[string]float64) {
	for key := range out {
		delete(out, key)
	}
	for _, from := range assets {
		fromPrice := prices[from+bridge]
		if fromPrice <= 0 {
			continue
		}
		for _, to := range assets {
			if from == to {
				continue
			}
			toPrice := prices[to+bridge]
			if toPrice <= 0 {
				continue
			}
			out[from+">"+to] = rotationPair{
				FromAsset:     from,
				ToAsset:       to,
				BaselineRatio: fromPrice / toPrice,
			}
		}
	}
}

func normalizedAssets(input []string, bridge string) []string {
	seen := map[string]bool{}
	var out []string
	for _, asset := range input {
		asset = strings.ToUpper(strings.TrimSpace(asset))
		if asset == "" || asset == bridge || seen[asset] {
			continue
		}
		seen[asset] = true
		out = append(out, asset)
	}
	return out
}
