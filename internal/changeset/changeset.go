// Package changeset handles writing changeset markdown files.
package changeset

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/smartcontractkit/gocs/internal/names"
)

// VersionType represents the type (SemVer impact) of the version bump.
type VersionType string

const (
	Major VersionType = "major"
	Minor VersionType = "minor"
	Patch VersionType = "patch"
)

// Entry represents a single package's changeset entry.
type Entry struct {
	Package     string
	VersionType VersionType
}

// Changeset represents a complete changeset with entries and a summary.
type Changeset struct {
	Entries []Entry
	Summary string
}

// Write creates a new changeset markdown file in the .changeset directory.
// It returns the path to the created file.
// Returns an error if the .changeset directory does not exist.
func Write(root string, cs Changeset) (string, error) {
	// Check that .changeset directory exists
	changesetDir := filepath.Join(root, ".changeset")
	info, err := os.Stat(changesetDir)
	if os.IsNotExist(err) {
		return "", errors.New(".changeset directory not found - is this repo set up for changesets?")
	}
	if err != nil {
		return "", fmt.Errorf("failed to access .changeset directory: %w", err)
	}
	if !info.IsDir() {
		return "", errors.New(".changeset exists but is not a directory")
	}

	// Generate a unique filename, retrying on collision
	var filename string
	for i := range 10 {
		name := names.Generate()
		filename = filepath.Join(changesetDir, name+".md")
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			break
		}
		if i == 9 {
			return "", errors.New("failed to generate unique changeset name after 10 attempts")
		}
	}

	// Build the changeset content
	content := buildContent(cs)

	// Write the file
	if err := os.WriteFile(filename, []byte(content), 0600); err != nil {
		return "", fmt.Errorf("failed to write changeset file: %w", err)
	}

	return filename, nil
}

// buildContent creates the markdown content for a changeset.
func buildContent(cs Changeset) string {
	var sb strings.Builder

	sb.WriteString("---\n")
	for _, entry := range cs.Entries {
		fmt.Fprintf(&sb, "\"%s\": %s\n", entry.Package, entry.VersionType)
	}
	sb.WriteString("---\n\n")
	sb.WriteString(cs.Summary)
	sb.WriteString("\n")

	return sb.String()
}
