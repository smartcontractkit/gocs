package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/smartcontractkit/gocs/internal/discovery"
)

func TestGetBaseBranch(t *testing.T) {
	tests := []struct {
		name       string
		configJSON string
		want       string
	}{
		{
			name:       "default when no config",
			configJSON: "",
			want:       "main",
		},
		{
			name:       "reads baseBranch from config",
			configJSON: `{"baseBranch": "develop"}`,
			want:       "develop",
		},
		{
			name:       "default when baseBranch empty",
			configJSON: `{"baseBranch": ""}`,
			want:       "main",
		},
		{
			name:       "default when baseBranch missing",
			configJSON: `{"other": "value"}`,
			want:       "main",
		},
		{
			name:       "default when invalid JSON",
			configJSON: `{invalid`,
			want:       "main",
		},
		{
			name:       "master branch",
			configJSON: `{"baseBranch": "master"}`,
			want:       "master",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()

			if tt.configJSON != "" {
				changesetDir := filepath.Join(root, ".changeset")
				if err := os.MkdirAll(changesetDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(changesetDir, "config.json"), []byte(tt.configJSON), 0600); err != nil {
					t.Fatal(err)
				}
			}

			got := getBaseBranch(root)
			if got != tt.want {
				t.Errorf("getBaseBranch() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsFileInPackage(t *testing.T) {
	packages := []discovery.Package{
		{Name: "root-pkg", Path: ""},
		{Name: "pkg-a", Path: "packages/a"},
		{Name: "pkg-b", Path: "packages/b"},
		{Name: "lib-x", Path: "libs/x"},
	}

	tests := []struct {
		name   string
		file   string
		pkgDir string
		want   bool
	}{
		// Root package tests
		{
			name:   "root level file belongs to root package",
			file:   "README.md",
			pkgDir: "",
			want:   true,
		},
		{
			name:   "root config file belongs to root package",
			file:   "package.json",
			pkgDir: "",
			want:   true,
		},
		{
			name:   "file in packages/a does not belong to root",
			file:   "packages/a/index.ts",
			pkgDir: "",
			want:   false,
		},
		{
			name:   "file in libs/x does not belong to root",
			file:   "libs/x/src/main.ts",
			pkgDir: "",
			want:   false,
		},
		{
			name:   "file in non-package subdir belongs to root",
			file:   "scripts/build.sh",
			pkgDir: "",
			want:   true,
		},
		// packages/a tests
		{
			name:   "file in packages/a belongs to pkg-a",
			file:   "packages/a/index.ts",
			pkgDir: "packages/a",
			want:   true,
		},
		{
			name:   "nested file in packages/a belongs to pkg-a",
			file:   "packages/a/src/utils/helper.ts",
			pkgDir: "packages/a",
			want:   true,
		},
		{
			name:   "file in packages/b does not belong to pkg-a",
			file:   "packages/b/index.ts",
			pkgDir: "packages/a",
			want:   false,
		},
		{
			name:   "root file does not belong to pkg-a",
			file:   "README.md",
			pkgDir: "packages/a",
			want:   false,
		},
		// Edge cases
		{
			name:   "file with similar prefix does not match",
			file:   "packages/ab/index.ts",
			pkgDir: "packages/a",
			want:   false,
		},
		{
			name:   "exact directory match",
			file:   "packages/a",
			pkgDir: "packages/a",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isFileInPackage(tt.file, tt.pkgDir, packages)
			if got != tt.want {
				t.Errorf("isFileInPackage(%q, %q) = %v, want %v", tt.file, tt.pkgDir, got, tt.want)
			}
		})
	}
}
