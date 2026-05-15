# Summary

Adds the feature set inspired by `ccxt/binance-trade-bot` and fixes managed-order phase handling:

- multi-asset rotation scout mode through a configurable bridge asset
- managed order timeouts with partial-fill handling
- explicit filled/not-filled order wait results so unfilled entries return to scanning instead of entering exit monitoring
- exit monitoring remains active when stop-loss, trailing-stop, or take-profit orders do not fill
- fee-aware take-profit adjustment
- persistent JSONL trade/scout/value history
- local HTTP history API
- backtest command backed by a small strategy registry

# Documentation

- Bumped CLI version to `v0.10.1`
- Updated `README.md` with new features, commands, help output, config sections, and managed-order behavior
- Extended both sample config files with persistence, fee, order-management, rotation, backtest, and API settings

# Validation

- `go test -v ./...`
- `go build ./...`
