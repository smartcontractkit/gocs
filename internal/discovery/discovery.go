// Package discovery finds packages in a monorepo workspace.
// It emulates @changesets/cli behavior by reading workspace configuration
// from pnpm-workspace.yaml or the workspaces field in package.json.
package discovery

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Package represents a discovered package.
type Package struct {
	Name    string
	Path    string // relative path to the package directory
	Version string // version from package.json (e.g. "1.2.3")
	Private bool   // whether the package is marked as private
}

// pnpmWorkspace represents the structure of pnpm-workspace.yaml
type pnpmWorkspace struct {
	Packages []string `yaml:"packages"`
}

// rootPackageJSON represents relevant fields from the root package.json
type rootPackageJSON struct {
	Name       string   `json:"name"`
	Workspaces []string `json:"workspaces"`
	Private    bool     `json:"private"`
}

// FindPackages searches for packages in a monorepo workspace.
// It reads workspace configuration from pnpm-workspace.yaml or the root package.json
// workspaces field, emulating @changesets/cli behavior.
func FindPackages(root string) ([]Package, error) {
	var packages []Package

	// Try to get workspace globs from configuration
	workspaceGlobs, err := getWorkspaceGlobs(root)
	if err != nil {
		return nil, err
	}

	// If we have workspace globs, use them to find packages
	if len(workspaceGlobs) > 0 {
		packages, err = findPackagesFromGlobs(root, workspaceGlobs)
		if err != nil {
			return nil, err
		}
	}

	// Always check for a root package (if it has a name)
	rootPkg, err := parsePackageJSON(filepath.Join(root, "package.json"))
	if err == nil && rootPkg.Name != "" {
		rootPkg.Path = ""
		// Add root package if not already in list
		hasRoot := false
		for _, p := range packages {
			if p.Path == "" {
				hasRoot = true
				break
			}
		}
		if !hasRoot {
			packages = append(packages, rootPkg)
		}
	}

	// Sort packages by name for consistent ordering
	sort.Slice(packages, func(i, j int) bool {
		return packages[i].Name < packages[j].Name
	})

	return packages, nil
}

// getWorkspaceGlobs reads workspace configuration from pnpm-workspace.yaml
// or the workspaces field in the root package.json.
func getWorkspaceGlobs(root string) ([]string, error) {
	// First, try pnpm-workspace.yaml (takes precedence)
	pnpmPath := filepath.Join(root, "pnpm-workspace.yaml")
	if data, err := os.ReadFile(pnpmPath); err == nil {
		var workspace pnpmWorkspace
		if err := yaml.Unmarshal(data, &workspace); err == nil && len(workspace.Packages) > 0 {
			return workspace.Packages, nil
		}
	}

	// Fall back to workspaces field in root package.json
	pkgPath := filepath.Join(root, "package.json")
	if data, err := os.ReadFile(pkgPath); err == nil {
		var pkg rootPackageJSON
		if err := json.Unmarshal(data, &pkg); err == nil && len(pkg.Workspaces) > 0 {
			return pkg.Workspaces, nil
		}
	}

	return nil, nil
}

// findPackagesFromGlobs finds packages matching the given workspace globs.
func findPackagesFromGlobs(root string, globs []string) ([]Package, error) {
	var packages []Package
	seen := make(map[string]bool)

	for _, glob := range globs {
		// Skip negation patterns (we handle them later)
		if strings.HasPrefix(glob, "!") {
			continue
		}

		// Normalize the glob pattern
		pattern := filepath.Join(root, glob)

		// Handle both "packages/*" and "packages/**" patterns
		// For directory globs, we need to find package.json inside
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				continue
			}

			var pkgJSONPath string
			if info.IsDir() {
				pkgJSONPath = filepath.Join(match, "package.json")
			} else if filepath.Base(match) == "package.json" {
				pkgJSONPath = match
			} else {
				continue
			}

			// Skip if we've already seen this package
			if seen[pkgJSONPath] {
				continue
			}
			seen[pkgJSONPath] = true

			pkg, err := parsePackageJSON(pkgJSONPath)
			if err != nil {
				continue
			}

			// Calculate relative path
			relPath, err := filepath.Rel(root, filepath.Dir(pkgJSONPath))
			if err != nil {
				relPath = filepath.Dir(pkgJSONPath)
			}
			if relPath == "." {
				relPath = ""
			}

			pkg.Path = relPath
			packages = append(packages, pkg)
		}
	}

	return packages, nil
}

// parsePackageJSON reads and parses a package.json file.
func parsePackageJSON(path string) (Package, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Package{}, err
	}

	var pkgJSON struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Private bool   `json:"private"`
	}

	if err := json.Unmarshal(data, &pkgJSON); err != nil {
		return Package{}, err
	}

	// Skip packages without a name
	if pkgJSON.Name == "" {
		return Package{}, os.ErrNotExist
	}

	return Package{Name: pkgJSON.Name, Version: pkgJSON.Version, Private: pkgJSON.Private}, nil
}
