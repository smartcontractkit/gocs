package names

import (
	"regexp"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	name := Generate()

	parts := strings.Split(name, "-")
	if len(parts) != 3 {
		t.Errorf("expected 3 parts separated by hyphens, got %d: %q", len(parts), name)
	}

	matched, err := regexp.MatchString(`^[a-z]+-[a-z]+-[a-z]+$`, name)
	if err != nil {
		t.Fatalf("regex error: %v", err)
	}
	if !matched {
		t.Errorf("name %q does not match expected pattern", name)
	}
}

func TestGenerateUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	iterations := 100

	for range iterations {
		name := Generate()
		seen[name] = true
	}

	if len(seen) < iterations/2 {
		t.Errorf("expected more unique names, got %d unique out of %d", len(seen), iterations)
	}
}

func TestWordListsNotEmpty(t *testing.T) {
	if len(adjectives) == 0 {
		t.Error("adjectives list is empty")
	}
	if len(nouns) == 0 {
		t.Error("nouns list is empty")
	}
	if len(verbs) == 0 {
		t.Error("verbs list is empty")
	}
}
