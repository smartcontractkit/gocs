# gocs

A Go tool for generating [changesets](https://github.com/changesets/changesets) markdown files without requiring the node package from NPM.

> **Note:** This tool only generates `.changeset/*.md` files. It does not replace the full `@changesets/cli` functionality—you still need changesets in CI for `changeset version` and `changeset publish` commands.

## Installation

```bash
go install github.com/smartcontractkit/gocs/cmd/gocs@latest
```

Or add as a tool dependency (Go 1.24+):

```bash
go get -tool github.com/smartcontractkit/gocs/cmd/gocs@latest
go tool gocs
```

Or run directly without installing:

```bash
go run github.com/smartcontractkit/gocs/cmd/gocs@latest
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

## How It Works

### Package Discovery

`gocs` scans your repository for `package.json` files and extracts the `name` field from each. These become the selectable packages in interactive mode or valid values for the `-pkg` flag. Only packages with a `name` field are discovered.

### Changeset Format

Each generated changeset markdown file contains:

- **Package name(s)**: The npm package name from `package.json`
- **Impact type**: `major`, `minor`, or `patch` — determines how the version number changes
- **Changelog message**: Description of the change for the changelog

### CI Integration with Changesets

The generated `.changeset/*.md` files are consumed by the [Changesets CLI](https://github.com/changesets/changesets) in CI:

1. **`changeset version`**: Reads all pending changesets and:
   - Updates the `version` field in each package's `package.json` based on the impact type:
     - `major`: `1.2.3` → `2.0.0`
     - `minor`: `1.2.3` → `1.3.0`
     - `patch`: `1.2.3` → `1.2.4`
   - Appends the changelog message to the package's `CHANGELOG.md`
   - Deletes the consumed changeset files

2. **`changeset publish`**: Publishes the updated packages to npm

This separation allows `gocs` to handle changeset creation without Node.js, while still leveraging the full Changesets workflow in CI.

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
