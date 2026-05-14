# Summary

Adds the feature set inspired by `ccxt/binance-trade-bot`:

- multi-asset rotation scout mode through a configurable bridge asset
- managed order timeouts with partial-fill handling
- fee-aware take-profit adjustment
- persistent JSONL trade/scout/value history
- local HTTP history API
- backtest command backed by a small strategy registry

# Documentation

- Bumped CLI version to `v0.10.0`
- Updated `README.md` with new features, commands, help output, and config sections
- Extended both sample config files with persistence, fee, order-management, rotation, backtest, and API settings

# Validation

- `go test -v ./...`
- `go build ./...`
