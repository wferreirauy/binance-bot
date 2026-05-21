# Contributing to binance-bot

Thanks for your interest in contributing! This document describes how to report issues, propose changes, and submit pull requests.

## Code of Conduct

By participating in this project you agree to keep interactions respectful and constructive. Be kind, assume good intent, and focus on the technical merits of contributions.

## Reporting Bugs

Open an issue on GitHub with:

- A clear, descriptive title.
- Steps to reproduce (commands, config snippets, env vars — **redact secrets**).
- Expected vs. actual behavior.
- Version (`binance-bot --version`), OS, and Go version.
- Relevant logs or stack traces.

Search existing issues first to avoid duplicates.

## Requesting Features

Open an issue describing:

- The problem or use case.
- The proposed solution and alternatives considered.
- Impact on existing CLI flags, config, or trade behavior (breaking vs. additive).

## Development Setup

Requires the Go version declared in [go.mod](go.mod).

```sh
git clone https://github.com/wferreirauy/binance-bot.git
cd binance-bot
go build ./...
go test -v ./...
```

Set the following environment variables as needed for runtime:

- `BINANCE_API_KEY`, `BINANCE_SECRET_KEY`
- `OPENAI_API_KEY`, `DEEPSEEK_API_KEY`, `ANTHROPIC_API_KEY`

Use [sample-binance-config.yml](sample-binance-config.yml) as a reference for the YAML config format.

## Branching & Pull Requests

- Branch from `main` using a descriptive name (e.g. `feat/rsi-divergence`, `fix/order-rounding`).
- Keep PRs focused — one logical change per PR.
- Rebase onto the latest `main` before opening the PR.
- Open the PR against `main` and fill in the description (see [PR-description-draft.md](PR-description-draft.md)).
- Ensure CI passes and at least one maintainer review is approved before merge.

## Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<optional scope>): <short summary>

<optional body>
<optional footer>
```

Common types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `perf`, `build`, `ci`.

Examples:

- `feat(strategy): add RSI divergence detector`
- `fix(tui): prevent panel race on shutdown`
- `docs(readme): document --dry-run flag`

## Coding Style

- Format with `gofmt` / `goimports` before committing.
- Run `go vet ./...` and fix any reported issues.
- Follow standard Go idioms: small interfaces, explicit errors, no panics in library code.
- TUI updates from goroutines must use `app.QueueUpdateDraw()` (see `tui/` package).
- Keep AI agent providers concurrency-safe; consensus is reached via `ai.Orchestrator`.

## Testing

- Add or update unit tests for any behavior change.
- Run the full suite locally: `go test -v ./...`.
- Aim to cover edge cases for trade logic, indicators, and config validation.
- Do not commit changes that leave the suite failing.

## Versioning

This project follows [Semantic Versioning](https://semver.org/). The version lives in the `Version` field of the `cli.App` struct in [main.go](main.go).

**Every PR must bump the version** according to its impact:

- **MAJOR** (`vX.0.0`) — breaking changes to CLI flags, config format, or trade behavior.
- **MINOR** (`v0.X.0`) — new features (indicators, TUI panels, AI providers, etc.).
- **PATCH** (`v0.0.X`) — bug fixes, display tweaks, dependency updates.

## Documentation

When a feature is added, removed, or changed, update [README.md](README.md) accordingly, including:

- Features list at the top.
- Usage examples and command documentation.
- Help output block (version, commands list).
- Command arguments table.
- Trading strategy logic section.
- Configuration sections (if config fields changed).

On every branch with new changes, rewrite [PR-description-draft.md](PR-description-draft.md) so it reflects the final PR description.

## Questions

For anything not covered here, open a GitHub issue.
