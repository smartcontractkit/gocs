package changeset

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWrite(t *testing.T) {
	root := t.TempDir()

	changesetDir := filepath.Join(root, ".changeset")
	if err := os.MkdirAll(changesetDir, 0755); err != nil {
		t.Fatal(err)
	}

	cs := Changeset{
		Entries: []Entry{
			{Package: "mypackage", VersionType: Patch},
		},
		Summary: "Fixed a bug",
	}

	path, err := Write(root, cs)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(path, changesetDir) {
		t.Errorf("expected path to be in %s, got %s", changesetDir, path)
	}

	if !strings.HasSuffix(path, ".md") {
		t.Errorf("expected .md extension, got %s", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	expected := "---\n\"mypackage\": patch\n---\n\nFixed a bug\n"
	if string(content) != expected {
		t.Errorf("unexpected content:\ngot:\n%s\nwant:\n%s", string(content), expected)
	}
}

func TestWriteMultiplePackages(t *testing.T) {
	root := t.TempDir()
	changesetDir := filepath.Join(root, ".changeset")
	if err := os.MkdirAll(changesetDir, 0755); err != nil {
		t.Fatal(err)
	}

	cs := Changeset{
		Entries: []Entry{
			{Package: "pkg-a", VersionType: Major},
			{Package: "pkg-b", VersionType: Minor},
			{Package: "pkg-c", VersionType: Patch},
		},
		Summary: "Multi-package change",
	}

	path, err := Write(root, cs)
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	expected := "---\n\"pkg-a\": major\n\"pkg-b\": minor\n\"pkg-c\": patch\n---\n\nMulti-package change\n"
	if string(content) != expected {
		t.Errorf("unexpected content:\ngot:\n%s\nwant:\n%s", string(content), expected)
	}
}

func TestWriteFailsWithoutChangesetDir(t *testing.T) {
	root := t.TempDir()

	cs := Changeset{
		Entries: []Entry{
			{Package: "mypackage", VersionType: Patch},
		},
		Summary: "Test",
	}

	_, err := Write(root, cs)
	if err == nil {
		t.Fatal("expected error when .changeset directory doesn't exist")
	}

	if !strings.Contains(err.Error(), ".changeset directory not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestWriteFailsWhenChangesetIsFile(t *testing.T) {
	root := t.TempDir()

	changesetPath := filepath.Join(root, ".changeset")
	if err := os.WriteFile(changesetPath, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}

	cs := Changeset{
		Entries: []Entry{
			{Package: "mypackage", VersionType: Patch},
		},
		Summary: "Test",
	}

	_, err := Write(root, cs)
	if err == nil {
		t.Fatal("expected error when .changeset is a file")
	}

	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestVersionTypes(t *testing.T) {
	tests := []struct {
		vt   VersionType
		want string
	}{
		{Major, "major"},
		{Minor, "minor"},
		{Patch, "patch"},
	}

	for _, tt := range tests {
		if string(tt.vt) != tt.want {
			t.Errorf("VersionType %v = %q, want %q", tt.vt, string(tt.vt), tt.want)
		}
	}
}
