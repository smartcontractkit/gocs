// Package git provides utilities for detecting changed files in git repositories.
package git

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/smartcontractkit/gocs/internal/discovery"
)

// changesetConfig represents the .changeset/config.json structure
type changesetConfig struct {
	BaseBranch string `json:"baseBranch"`
}

// GetModifiedPackages returns a map of package names that have changes since the base branch.
// This emulates @changesets/cli behavior of detecting which packages have been modified
// by comparing against the configured baseBranch (defaults to "main").
func GetModifiedPackages(root string, packages []discovery.Package) (map[string]bool, error) {
	modified := make(map[string]bool)

	// Get the base branch from .changeset/config.json (default: "main")
	baseBranch := getBaseBranch(root)

	// Get list of changed files since base branch + uncommitted changes
	changedFiles, err := getChangedFiles(root, baseBranch)
	if err != nil {
		// If git fails (not a repo, etc.), return empty map
		return modified, nil
	}

	// For each package, check if any changed file is within its directory
	for _, pkg := range packages {
		pkgDir := pkg.Path
		if pkgDir == "" {
			// Root package - check if any file at root level changed
			for _, file := range changedFiles {
				// File is at root if it doesn't contain a path separator
				// or if it's directly in the root and not in a subdirectory that's another package
				if isFileInPackage(file, "", packages) {
					modified[pkg.Name] = true
					break
				}
			}
		} else {
			for _, file := range changedFiles {
				if isFileInPackage(file, pkgDir, packages) {
					modified[pkg.Name] = true
					break
				}
			}
		}
	}

	return modified, nil
}

// getBaseBranch reads the baseBranch from .changeset/config.json, defaulting to "main".
func getBaseBranch(root string) string {
	configPath := filepath.Join(root, ".changeset", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "main"
	}

	var config changesetConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return "main"
	}

	if config.BaseBranch == "" {
		return "main"
	}
	return config.BaseBranch
}

// getChangedFiles returns a list of files that have been modified since the base branch,
// including committed changes on the current branch, staged, and unstaged changes.
// This matches @changesets/cli behavior which uses `git diff --name-only <mergeBase>`.
// Using a single commit (not a range) compares to the working tree, automatically including
// staged and unstaged modifications to tracked files.
// Note: Untracked files are NOT included, matching @changesets/cli behavior.
func getChangedFiles(root, baseBranch string) ([]string, error) {
	seen := make(map[string]bool)
	var files []string

	addFiles := func(output []byte) {
		if len(output) > 0 {
			for line := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
				if line != "" && !seen[line] {
					seen[line] = true
					files = append(files, line)
				}
			}
		}
	}

	// Try to find the merge-base with the base branch
	// First try origin/<baseBranch>, then <baseBranch>
	var mergeBase string
	for _, ref := range []string{"origin/" + baseBranch, baseBranch} {
		cmd := exec.CommandContext(context.Background(), "git", "merge-base", ref, "HEAD")
		cmd.Dir = root
		output, err := cmd.Output()
		if err == nil {
			mergeBase = strings.TrimSpace(string(output))
			break
		}
	}

	// Get all changes since merge-base (committed + staged + unstaged to tracked files)
	// Using single commit compares to working tree, not HEAD
	if mergeBase != "" {
		cmd := exec.CommandContext(context.Background(), "git", "diff", "--name-only", mergeBase)
		cmd.Dir = root
		output, _ := cmd.Output()
		addFiles(output)
	} else {
		// Fallback: no merge-base found, compare against HEAD for uncommitted changes
		cmd := exec.CommandContext(context.Background(), "git", "diff", "--name-only", "HEAD")
		cmd.Dir = root
		output, err := cmd.Output()
		if err != nil {
			// Try without HEAD for new repos with no commits
			cmd = exec.CommandContext(context.Background(), "git", "diff", "--name-only")
			cmd.Dir = root
			output, _ = cmd.Output()
		}
		addFiles(output)

		// Also get staged changes in fallback mode
		cmd = exec.CommandContext(context.Background(), "git", "diff", "--name-only", "--cached")
		cmd.Dir = root
		output, _ = cmd.Output()
		addFiles(output)
	}

	return files, nil
}

// isFileInPackage checks if a file belongs to a specific package directory.
func isFileInPackage(file, pkgDir string, packages []discovery.Package) bool {
	if pkgDir == "" {
		// For root package, check if file is not in any other package's directory
		for _, pkg := range packages {
			if pkg.Path != "" && strings.HasPrefix(file, pkg.Path+"/") {
				return false
			}
		}
		// File is at root level or in a non-package subdirectory
		return true
	}

	// Normalize paths for comparison
	file = filepath.Clean(file)
	pkgDir = filepath.Clean(pkgDir)

	// Check if file is within the package directory
	return strings.HasPrefix(file, pkgDir+"/") || file == pkgDir
}
