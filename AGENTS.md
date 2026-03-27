# Project Guidelines

## Versioning

The application version is defined in `main.go` on the `Version` field of the `cli.App` struct.
**Every time a change is applied to the codebase, the version must be bumped accordingly.**

Follow [Semantic Versioning](https://semver.org/):
- **MAJOR** (`vX.0.0`): breaking changes to CLI flags, config format, or trade behavior
- **MINOR** (`v0.X.0`): new features (e.g., new indicators, TUI panels, AI providers)
- **PATCH** (`v0.0.X`): bug fixes, display tweaks, dependency updates

## Build and Test

- Build: `go build ./...`
- Test: `go test -v ./...`
- Module: `github.com/wferreirauy/binance-bot`

## Conventions

- Config file: YAML format (`sample-binance-config.yml` as reference)
- Environment variables for secrets: `BINANCE_API_KEY`, `BINANCE_SECRET_KEY`, `OPENAI_API_KEY`, `DEEPSEEK_API_KEY`, `ANTHROPIC_API_KEY`
- TUI dashboard (`tui/` package) uses `tview` — all updates from goroutines must use `app.QueueUpdateDraw()`
- AI agents (`ai/` package) run concurrently and return consensus via the `Orchestrator`
