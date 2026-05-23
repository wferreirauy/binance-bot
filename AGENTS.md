# Project Guidelines

Before applying any change, make sure to update the local files with the remote git repo, `git pull`.
Amd for every new task, follow the git workflow indicated bellow, ensuring to create a new branch of updated main.

## Git Workflow

For every change set, follow this workflow:

1. **Create a new branch** off `main` with a descriptive name reflecting the change
   (e.g. `feat/<short-desc>`, `fix/<short-desc>`, `docs/<short-desc>`,
   `chore/<short-desc>`).
2. **Stage** the relevant files (`git add <files>`).
3. **Commit** using [Conventional Commits](https://www.conventionalcommits.org/)
   and include the bumped semantic version in the message
   (e.g. `feat(strategy): add RSI divergence detector (v0.13.0)`,
   `fix(tui): prevent panel race on shutdown (v0.12.2)`,
   `docs: add CONTRIBUTING.md (v0.12.1)`).
4. **Push** the branch to the remote (`git push -u origin <branch>`).

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

## Documentation

**Every time a feature is added, removed, or changed, update `README.md` accordingly.**
This includes:
- Features list at the top
- Usage examples and command documentation
- Help output block (version, commands list)
- Command arguments table
- Trading strategy logic section
- Configuration sections (if config fields were added/changed)

## Release
On every new changes on new branch, rewrite completely the file PR-description-draft.md, that will be used to create the PR with the new changes created. Create the file if does not exist.
