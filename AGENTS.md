# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`gocs` is a Go CLI that generates [changesets](https://github.com/changesets/changesets) markdown files (`.changeset/*.md`) without requiring the node `@changesets/cli` package at authoring time. It emulates changesets' package-discovery and change-detection behavior. It only *writes* changeset files — `changeset version` / `changeset publish` still run in CI via the node toolchain.

## Commands

Toolchain is managed via [mise](https://mise.jdx.dev/) (see `.tool-versions`: Go 1.26, golangci-lint 2.10, nodejs 22).

```bash
make build         # build ./gocs from ./cmd/gocs (injects git-describe version via -ldflags)
make install       # install to GOBIN/GOPATH/bin; override with INSTALL_DIR=/usr/local/bin
make test          # go test -race -v ./...
make coverage      # tests + HTML coverage report at coverage.html
make lint          # golangci-lint run ./...
make fmt           # go fmt + gofumpt -w
make vet           # go vet
make tidy          # go mod tidy && go mod verify
make check         # fmt vet lint test (full local gate)
```

Run a single test: `go test -race -v ./internal/discovery/ -run TestFindPackages`

CI (`.github/workflows/ci.yml`) runs `make tidy` (with `git diff --exit-code` on go.mod/go.sum), `make vet`, `go test -race -coverprofile`, `make build`, and golangci-lint against `.golangci.yml`. Keep go.mod/go.sum tidy so CI's diff check passes.

## Releases

Releases are driven by changesets, not git tags you cut manually. The `Changesets` workflow (`.github/workflows/changesets.yml`) runs on push to `main`, consumes `.changeset/*.md`, versions `package.json`, and publishes. Tags are formatted `<pkg>/v<version>` (`changesets-tag-separator: "/v"`). To cut a release, add a changeset file (the tool's own output) and merge to main.

## Architecture

Entry point: `cmd/gocs/main.go`. It branches on flags into non-interactive or interactive (TUI) mode, then calls `changeset.Write` to produce the file.

Four internal packages, each with a `_test.go`:

- **`internal/discovery`** — `FindPackages(root)` emulates `@changesets/cli` workspace resolution. Reads `pnpm-workspace.yaml` (YAML, takes precedence), falling back to the `workspaces` field in root `package.json`. Globs are resolved with `filepath.Glob` (single-segment only; `packages/**` deep globs are not expanded). Always also considers the root `package.json` itself if it has a `name`. Packages without a `name` field are skipped. Result sorted by name.
- **`internal/git`** — `GetModifiedPackages` flags packages touched since `.changeset/config.json`'s `baseBranch` (default `main`). Uses `git merge-base origin/<base> HEAD` then `git diff --name-only <mergeBase>` to get committed + staged + unstaged changes to *tracked* files (untracked files excluded, matching changesets). Falls back to `git diff HEAD` / `--cached` when no merge-base is found. Git errors are swallowed (returns empty map) so non-git dirs don't break the TUI.
- **`internal/tui`** — Bubble Tea (`bubbletea` + `bubbles/textarea` + `lipgloss`) state machine with four states: `StateSelectPackages` → `StateSelectVersion` → `StateEnterSummary` → `StateDone`. Packages are grouped into "changed" / "unchanged" sections (changed first, not pre-selected); section headers are selectable rows that toggle the whole section. Version selection is pnpm-style two-step: first prompt for `major` bumps, then `minor`; anything unselected after both steps becomes `patch`. A `-type` flag (`fixedVersionType`) skips version selection entirely. `ctrl+d` confirms the summary (not enter — enter is a newline in the textarea). Scrolling is manual via `scrollOffset` against `windowHeight`.
- **`internal/changeset`** — `Write(root, cs)` creates `.changeset/<random>.md`. The `.changeset` directory must already exist (error otherwise — this is how it detects a changesets-configured repo). Filenames come from `internal/names` (`adjective-noun-verb`, `math/rand/v2`, retried up to 10× on collision). File mode `0600`. Content is a YAML frontmatter block mapping each package to its bump type, then the summary.

## Conventions

- golangci-lint config (`.golangci.yml`) is shared Chainlink infra config — it references `chainlink-common` logger printf rules and depguard denials that aren't used here but are kept for consistency. `gosec` G404 (weak random) is the reason `names.Generate` has `//nolint:gosec` comments — don't remove them.
- `goimports` is configured with `local-prefixes: github.com/smartcontractkit/` — group imports std / third-party / `smartcontractkit` separately.
- Node/pnpm here exists solely to run `@changesets/cli` for versioning/publishing in CI (`package.json` scripts `ci:changeset:version` / `ci:changeset:publish`). The Go code does not depend on node.
