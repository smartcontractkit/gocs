# gocs

A Go tool for generating [changesets](https://github.com/changesets/changesets) markdown files without requiring the node package from NPM.

> **Note:** This tool only generates `.changeset/*.md` files. It does not replace the full `@changesets/cli` functionality—you still need changesets in CI for `changeset version` and `changeset publish` commands.

## Installation

```bash
go install github.com/smartcontractkit/gocs/cmd/gocs@latest
```

Or use directly with `go tool` (Go 1.24+):

```bash
go tool github.com/smartcontractkit/gocs/cmd/gocs@latest
```

## Usage

### Interactive Mode (TUI)

```bash
gocs
```

This launches an interactive terminal UI where you can:

1. Select one or more packages from discovered `package.json` files
2. Choose version bump type (major/minor/patch) for each package
3. Enter a changelog message

### Non-Interactive Mode

```bash
# Single package
gocs -pkg chainlink -m "Fix memory leak #internal"

# Multiple packages (same version type)
gocs -pkg chainlink,contracts -type minor -m "Add new feature"

# Specify version type
gocs -pkg mypackage -type major -m "Breaking API change"
```

### Flags

| Flag       | Description                             | Default |
| ---------- | --------------------------------------- | ------- |
| `-pkg`     | Package name(s), comma-separated        | -       |
| `-m`       | Changelog message                       | -       |
| `-type`    | Version bump: `major`, `minor`, `patch` | `patch` |
| `-version` | Print version                           | -       |
| `-help`    | Show help                               | -       |

## Output

Creates a markdown file in `.changeset/` with a random name:

```markdown
---
"package-name": patch
---

Your changelog message here
```

## Adding to a Go Project

Add to your `go.mod` as a tool dependency:

```bash
go get -tool github.com/smartcontractkit/gocs/cmd/gocs@latest
```

Then use with:

```bash
go tool gocs
```

## License

MIT
