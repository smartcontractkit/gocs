package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindPackages(t *testing.T) {
	root := t.TempDir()

	// Create pnpm-workspace.yaml
	if err := os.WriteFile(filepath.Join(root, "pnpm-workspace.yaml"), []byte("packages:\n  - packages/*"), 0600); err != nil {
		t.Fatal(err)
	}

	pkgDir := filepath.Join(root, "packages", "mypackage")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name": "mypackage"}`), 0600); err != nil {
		t.Fatal(err)
	}

	packages, err := FindPackages(root)
	if err != nil {
		t.Fatal(err)
	}

	if len(packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(packages))
	}
	if packages[0].Name != "mypackage" {
		t.Errorf("expected name 'mypackage', got %q", packages[0].Name)
	}
	if packages[0].Path != "packages/mypackage" {
		t.Errorf("expected path 'packages/mypackage', got %q", packages[0].Path)
	}
}

func TestFindPackagesMultiple(t *testing.T) {
	root := t.TempDir()

	// Create pnpm-workspace.yaml
	if err := os.WriteFile(filepath.Join(root, "pnpm-workspace.yaml"), []byte("packages:\n  - packages/*"), 0600); err != nil {
		t.Fatal(err)
	}

	for _, pkg := range []string{"alpha", "beta", "gamma"} {
		pkgDir := filepath.Join(root, "packages", pkg)
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			t.Fatal(err)
		}
		content := `{"name": "` + pkg + `"}`
		if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}

	packages, err := FindPackages(root)
	if err != nil {
		t.Fatal(err)
	}

	if len(packages) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(packages))
	}

	if packages[0].Name != "alpha" || packages[1].Name != "beta" || packages[2].Name != "gamma" {
		t.Errorf("packages not sorted correctly: %v", packages)
	}
}

func TestFindPackagesFromPackageJSONWorkspaces(t *testing.T) {
	root := t.TempDir()

	// Create root package.json with workspaces field (npm/yarn style)
	rootPkgJSON := `{"name": "monorepo", "workspaces": ["packages/*"]}`
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(rootPkgJSON), 0600); err != nil {
		t.Fatal(err)
	}

	pkgDir := filepath.Join(root, "packages", "mylib")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name": "mylib"}`), 0600); err != nil {
		t.Fatal(err)
	}

	packages, err := FindPackages(root)
	if err != nil {
		t.Fatal(err)
	}

	// Should find both the workspace package and the root package
	if len(packages) != 2 {
		t.Fatalf("expected 2 packages, got %d: %v", len(packages), packages)
	}
}

func TestFindPackagesIgnoresNonWorkspacePackages(t *testing.T) {
	root := t.TempDir()

	// Create pnpm-workspace.yaml that only includes packages/*
	if err := os.WriteFile(filepath.Join(root, "pnpm-workspace.yaml"), []byte("packages:\n  - packages/*"), 0600); err != nil {
		t.Fatal(err)
	}

	// Create a package in the workspace
	workspacePkg := filepath.Join(root, "packages", "included")
	if err := os.MkdirAll(workspacePkg, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspacePkg, "package.json"), []byte(`{"name": "included"}`), 0600); err != nil {
		t.Fatal(err)
	}

	// Create a package outside the workspace (should be ignored)
	outsidePkg := filepath.Join(root, "scripts", "tool")
	if err := os.MkdirAll(outsidePkg, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outsidePkg, "package.json"), []byte(`{"name": "excluded"}`), 0600); err != nil {
		t.Fatal(err)
	}

	packages, err := FindPackages(root)
	if err != nil {
		t.Fatal(err)
	}

	if len(packages) != 1 {
		t.Fatalf("expected 1 package (non-workspace should be ignored), got %d: %v", len(packages), packages)
	}
	if packages[0].Name != "included" {
		t.Errorf("expected 'included', got %q", packages[0].Name)
	}
}

func TestFindPackagesSkipsNoName(t *testing.T) {
	root := t.TempDir()

	// Create workspace config
	if err := os.WriteFile(filepath.Join(root, "pnpm-workspace.yaml"), []byte("packages:\n  - packages/*"), 0600); err != nil {
		t.Fatal(err)
	}

	// Package with no name
	pkgDir := filepath.Join(root, "packages", "noname")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"version": "1.0.0"}`), 0600); err != nil {
		t.Fatal(err)
	}

	packages, err := FindPackages(root)
	if err != nil {
		t.Fatal(err)
	}

	if len(packages) != 0 {
		t.Fatalf("expected 0 packages (no name should be skipped), got %d", len(packages))
	}
}

func TestFindPackagesRootPackage(t *testing.T) {
	root := t.TempDir()

	// Root package with name but no workspaces - should still be found
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name": "root-pkg"}`), 0600); err != nil {
		t.Fatal(err)
	}

	packages, err := FindPackages(root)
	if err != nil {
		t.Fatal(err)
	}

	if len(packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(packages))
	}
	if packages[0].Path != "" {
		t.Errorf("expected empty path for root package, got %q", packages[0].Path)
	}
}

func TestFindPackagesPnpmWorkspaceTakesPrecedence(t *testing.T) {
	root := t.TempDir()

	// Create both pnpm-workspace.yaml and package.json with workspaces
	// pnpm-workspace.yaml should take precedence
	if err := os.WriteFile(filepath.Join(root, "pnpm-workspace.yaml"), []byte("packages:\n  - pnpm-packages/*"), 0600); err != nil {
		t.Fatal(err)
	}
	rootPkgJSON := `{"name": "monorepo", "workspaces": ["npm-packages/*"]}`
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(rootPkgJSON), 0600); err != nil {
		t.Fatal(err)
	}

	// Create package in pnpm workspace
	pnpmPkg := filepath.Join(root, "pnpm-packages", "pnpm-lib")
	if err := os.MkdirAll(pnpmPkg, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pnpmPkg, "package.json"), []byte(`{"name": "pnpm-lib"}`), 0600); err != nil {
		t.Fatal(err)
	}

	// Create package in npm workspace (should be ignored because pnpm takes precedence)
	npmPkg := filepath.Join(root, "npm-packages", "npm-lib")
	if err := os.MkdirAll(npmPkg, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(npmPkg, "package.json"), []byte(`{"name": "npm-lib"}`), 0600); err != nil {
		t.Fatal(err)
	}

	packages, err := FindPackages(root)
	if err != nil {
		t.Fatal(err)
	}

	// Should find pnpm-lib and the root monorepo package
	names := make(map[string]bool)
	for _, p := range packages {
		names[p.Name] = true
	}

	if !names["pnpm-lib"] {
		t.Error("expected pnpm-lib to be found")
	}
	if names["npm-lib"] {
		t.Error("npm-lib should not be found when pnpm-workspace.yaml exists")
	}
}
