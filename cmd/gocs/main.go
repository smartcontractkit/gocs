// gocs is a tool for generating changeset markdown files.
//
// Usage:
//
//	gocs                                          # Interactive TUI mode
//	gocs -pkg <name> -m "changelog message"       # Non-interactive mode
//	gocs -pkg <name> -type minor -m "message"     # Specify version type
//
// Install as a Go tool:
//
//	go install github.com/smartcontractkit/gocs/cmd/gocs@latest
//
// Or add as a tool dependency (Go 1.24+):
//
//	go get -tool github.com/smartcontractkit/gocs/cmd/gocs@latest
//	go tool gocs
//
// Or run directly without installing:
//
//	go run github.com/smartcontractkit/gocs/cmd/gocs@latest
//
// The tool discovers package.json files in the current directory tree
// and allows you to create changeset files in the .changeset directory.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/smartcontractkit/gocs/internal/changeset"
	"github.com/smartcontractkit/gocs/internal/discovery"
	"github.com/smartcontractkit/gocs/internal/git"
	"github.com/smartcontractkit/gocs/internal/tui"
)

var version = "dev"

func init() {
	if version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	fs := flag.NewFlagSet("gocs", flag.ContinueOnError)

	pkgFlag := fs.String("pkg", "", "Package name(s) to include (comma-separated for multiple)")
	msgFlag := fs.String("m", "", "Changelog message")
	typeFlag := fs.String("type", "patch", "Version bump type: major, minor, or patch")
	versionFlag := fs.Bool("version", false, "Print version and exit")
	helpFlag := fs.Bool("help", false, "Show help")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "gocs - Generate changeset markdown files\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  gocs                                      # Interactive TUI mode\n")
		fmt.Fprintf(os.Stderr, "  gocs -pkg <name> -m \"message\"           # Non-interactive mode\n")
		fmt.Fprintf(os.Stderr, "  gocs -pkg <name> -type minor -m \"msg\"   # With version type\n")
		fmt.Fprintf(os.Stderr, "\nFlags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  gocs -pkg chainlink -m 'Fix memory leak #internal'\n")
		fmt.Fprintf(os.Stderr, "  gocs -pkg chainlink,contracts -type minor -m 'Add new feature'\n")
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	if *helpFlag {
		fs.Usage()
		return nil
	}

	if *versionFlag {
		fmt.Printf("gocs version %s\n", version)
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Validate flag combinations
	hasPkg := *pkgFlag != ""
	hasMsg := *msgFlag != ""
	hasType := *typeFlag != "patch" // non-default type was specified

	if hasPkg && hasMsg {
		return runNonInteractive(cwd, *pkgFlag, *msgFlag, *typeFlag)
	}

	if hasPkg || hasMsg || hasType {
		if !hasPkg {
			return fmt.Errorf("missing required flag: -pkg is required for non-interactive mode")
		}
		if !hasMsg {
			return fmt.Errorf("missing required flag: -m is required for non-interactive mode")
		}
	}

	return runInteractive(cwd)
}

func runNonInteractive(cwd, pkgNames, message, versionType string) error {
	var vt changeset.VersionType
	switch strings.ToLower(versionType) {
	case "major":
		vt = changeset.Major
	case "minor":
		vt = changeset.Minor
	case "patch":
		vt = changeset.Patch
	default:
		return fmt.Errorf("invalid version type %q: must be major, minor, or patch", versionType)
	}

	parts := strings.Split(pkgNames, ",")
	var packages []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			packages = append(packages, p)
		}
	}

	if len(packages) == 0 {
		return fmt.Errorf("no valid package names provided")
	}

	entries := make([]changeset.Entry, len(packages))
	for i, pkg := range packages {
		entries[i] = changeset.Entry{
			Package:     pkg,
			VersionType: vt,
		}
	}

	cs := changeset.Changeset{
		Entries: entries,
		Summary: message,
	}

	path, err := changeset.Write(cwd, cs)
	if err != nil {
		return err
	}

	fmt.Printf("🦋 Created changeset: %s\n", path)
	return nil
}

func runInteractive(cwd string) error {
	packages, err := discovery.FindPackages(cwd)
	if err != nil {
		return fmt.Errorf("failed to discover packages: %w", err)
	}

	if len(packages) == 0 {
		return fmt.Errorf("no package.json files found in %s", cwd)
	}

	// Detect modified packages to show them first (grouped)
	modifiedPkgs, _ := git.GetModifiedPackages(cwd, packages)

	cs, err := tui.RunWithChanged(packages, modifiedPkgs)
	if err != nil {
		return err
	}

	if cs == nil {
		fmt.Println("Cancelled")
		return nil
	}

	path, err := changeset.Write(cwd, *cs)
	if err != nil {
		return err
	}

	fmt.Printf("\n🦋 Created changeset: %s\n", path)
	return nil
}
