package tui

import (
	"strings"
	"testing"

	"github.com/smartcontractkit/gocs/internal/changeset"
	"github.com/smartcontractkit/gocs/internal/discovery"
)

func TestNewModel(t *testing.T) {
	packages := []discovery.Package{
		{Name: "pkg-a", Path: "packages/a"},
		{Name: "pkg-b", Path: "packages/b"},
	}

	m := NewModel(packages)

	if m.state != StateSelectPackages {
		t.Errorf("expected state StateSelectPackages, got %d", m.state)
	}
	if len(m.packages) != 2 {
		t.Errorf("expected 2 packages, got %d", len(m.packages))
	}
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m.cursor)
	}
	if len(m.selected) != 0 {
		t.Errorf("expected no selections, got %d", len(m.selected))
	}
}

func TestNewModelWithChanged(t *testing.T) {
	packages := []discovery.Package{
		{Name: "pkg-a", Path: "packages/a"},
		{Name: "pkg-b", Path: "packages/b"},
		{Name: "pkg-c", Path: "packages/c"},
		{Name: "pkg-d", Path: "packages/d"},
	}

	changedPkgs := map[string]bool{
		"pkg-b": true,
		"pkg-d": true,
	}

	m := NewModelWithChanged(packages, changedPkgs)

	// Verify counts
	if m.changedCount != 2 {
		t.Errorf("expected changedCount=2, got %d", m.changedCount)
	}
	if m.unchangedCount != 2 {
		t.Errorf("expected unchangedCount=2, got %d", m.unchangedCount)
	}

	// Verify total display items
	if len(m.displayItems) != 4 {
		t.Fatalf("expected 4 displayItems, got %d", len(m.displayItems))
	}

	// Verify changed packages come first
	if !m.displayItems[0].changed || m.displayItems[0].pkg.Name != "pkg-b" {
		t.Errorf("expected first item to be changed pkg-b, got %+v", m.displayItems[0])
	}
	if !m.displayItems[1].changed || m.displayItems[1].pkg.Name != "pkg-d" {
		t.Errorf("expected second item to be changed pkg-d, got %+v", m.displayItems[1])
	}

	// Verify unchanged packages come after
	if m.displayItems[2].changed || m.displayItems[2].pkg.Name != "pkg-a" {
		t.Errorf("expected third item to be unchanged pkg-a, got %+v", m.displayItems[2])
	}
	if m.displayItems[3].changed || m.displayItems[3].pkg.Name != "pkg-c" {
		t.Errorf("expected fourth item to be unchanged pkg-c, got %+v", m.displayItems[3])
	}

	// Verify origIndex is correct for selection mapping
	if m.displayItems[0].origIndex != 1 { // pkg-b is at index 1 in original
		t.Errorf("expected origIndex=1 for pkg-b, got %d", m.displayItems[0].origIndex)
	}
	if m.displayItems[2].origIndex != 0 { // pkg-a is at index 0 in original
		t.Errorf("expected origIndex=0 for pkg-a, got %d", m.displayItems[2].origIndex)
	}
}

func TestNewModelWithChangedByPath(t *testing.T) {
	packages := []discovery.Package{
		{Name: "pkg-a", Path: "packages/a"},
		{Name: "pkg-b", Path: "packages/b"},
	}

	// Match by path instead of name
	changedPkgs := map[string]bool{
		"packages/a": true,
	}

	m := NewModelWithChanged(packages, changedPkgs)

	if m.changedCount != 1 {
		t.Errorf("expected changedCount=1, got %d", m.changedCount)
	}
	if m.displayItems[0].pkg.Name != "pkg-a" {
		t.Errorf("expected pkg-a to be changed, got %s", m.displayItems[0].pkg.Name)
	}
}

func TestNewModelWithChangedAllChanged(t *testing.T) {
	packages := []discovery.Package{
		{Name: "pkg-a", Path: "packages/a"},
		{Name: "pkg-b", Path: "packages/b"},
	}

	changedPkgs := map[string]bool{
		"pkg-a": true,
		"pkg-b": true,
	}

	m := NewModelWithChanged(packages, changedPkgs)

	if m.changedCount != 2 {
		t.Errorf("expected changedCount=2, got %d", m.changedCount)
	}
	if m.unchangedCount != 0 {
		t.Errorf("expected unchangedCount=0, got %d", m.unchangedCount)
	}
}

func TestNewModelWithChangedNoneChanged(t *testing.T) {
	packages := []discovery.Package{
		{Name: "pkg-a", Path: "packages/a"},
		{Name: "pkg-b", Path: "packages/b"},
	}

	changedPkgs := map[string]bool{}

	m := NewModelWithChanged(packages, changedPkgs)

	if m.changedCount != 0 {
		t.Errorf("expected changedCount=0, got %d", m.changedCount)
	}
	if m.unchangedCount != 2 {
		t.Errorf("expected unchangedCount=2, got %d", m.unchangedCount)
	}
}

func TestCursorToLineNumber(t *testing.T) {
	packages := []discovery.Package{
		{Name: "pkg-a", Path: "packages/a"},
		{Name: "pkg-b", Path: "packages/b"},
		{Name: "pkg-c", Path: "packages/c"},
		{Name: "pkg-d", Path: "packages/d"},
	}

	changedPkgs := map[string]bool{
		"pkg-a": true,
		"pkg-b": true,
	}

	m := NewModelWithChanged(packages, changedPkgs)

	// Layout is:
	// Line 0: "changed packages" header
	// Line 1: pkg-a (cursor 0)
	// Line 2: pkg-b (cursor 1)
	// Line 3: "unchanged packages" header
	// Line 4: pkg-c (cursor 2)
	// Line 5: pkg-d (cursor 3)

	tests := []struct {
		cursor   int
		expected int
	}{
		{cursor: 0, expected: 1}, // pkg-a: cursor 0 + 1 header
		{cursor: 1, expected: 2}, // pkg-b: cursor 1 + 1 header
		{cursor: 2, expected: 4}, // pkg-c: cursor 2 + 2 headers
		{cursor: 3, expected: 5}, // pkg-d: cursor 3 + 2 headers
	}

	for _, tt := range tests {
		m.cursor = tt.cursor
		got := m.cursorToLineNumber()
		if got != tt.expected {
			t.Errorf("cursorToLineNumber() with cursor=%d: got %d, want %d", tt.cursor, got, tt.expected)
		}
	}
}

func TestCursorToLineNumberOnlyChanged(t *testing.T) {
	packages := []discovery.Package{
		{Name: "pkg-a", Path: "packages/a"},
		{Name: "pkg-b", Path: "packages/b"},
	}

	changedPkgs := map[string]bool{
		"pkg-a": true,
		"pkg-b": true,
	}

	m := NewModelWithChanged(packages, changedPkgs)

	// Layout is:
	// Line 0: "changed packages" header
	// Line 1: pkg-a (cursor 0)
	// Line 2: pkg-b (cursor 1)

	tests := []struct {
		cursor   int
		expected int
	}{
		{cursor: 0, expected: 1}, // pkg-a: cursor 0 + 1 header
		{cursor: 1, expected: 2}, // pkg-b: cursor 1 + 1 header
	}

	for _, tt := range tests {
		m.cursor = tt.cursor
		got := m.cursorToLineNumber()
		if got != tt.expected {
			t.Errorf("cursorToLineNumber() with cursor=%d: got %d, want %d", tt.cursor, got, tt.expected)
		}
	}
}

func TestCursorToLineNumberOnlyUnchanged(t *testing.T) {
	packages := []discovery.Package{
		{Name: "pkg-a", Path: "packages/a"},
		{Name: "pkg-b", Path: "packages/b"},
	}

	changedPkgs := map[string]bool{}

	m := NewModelWithChanged(packages, changedPkgs)

	// Layout is:
	// Line 0: "unchanged packages" header
	// Line 1: pkg-a (cursor 0)
	// Line 2: pkg-b (cursor 1)

	tests := []struct {
		cursor   int
		expected int
	}{
		{cursor: 0, expected: 1}, // pkg-a: cursor 0 + 1 header (unchanged header)
		{cursor: 1, expected: 2}, // pkg-b: cursor 1 + 1 header
	}

	for _, tt := range tests {
		m.cursor = tt.cursor
		got := m.cursorToLineNumber()
		if got != tt.expected {
			t.Errorf("cursorToLineNumber() with cursor=%d: got %d, want %d", tt.cursor, got, tt.expected)
		}
	}
}

func TestCursorToLineNumberNoDisplayItems(t *testing.T) {
	packages := []discovery.Package{
		{Name: "pkg-a", Path: "packages/a"},
		{Name: "pkg-b", Path: "packages/b"},
	}

	m := NewModel(packages)

	// With no displayItems, cursorToLineNumber returns cursor directly
	tests := []struct {
		cursor   int
		expected int
	}{
		{cursor: 0, expected: 0},
		{cursor: 1, expected: 1},
	}

	for _, tt := range tests {
		m.cursor = tt.cursor
		got := m.cursorToLineNumber()
		if got != tt.expected {
			t.Errorf("cursorToLineNumber() with cursor=%d: got %d, want %d", tt.cursor, got, tt.expected)
		}
	}
}

func TestVersionSelectionShowsCurrentVersion(t *testing.T) {
	packages := []discovery.Package{
		{Name: "chainlink", Path: "packages/chainlink", Version: "2.1.0"},
		{Name: "contracts", Path: "packages/contracts", Version: "0.5.3"},
		{Name: "noversion", Path: "packages/noversion"},
	}

	m := NewModel(packages)
	m.selectedPackages = packages
	m.versionTypes = make([]changeset.VersionType, len(packages))
	m.state = StateSelectVersion
	m.currentPkgIndex = 0
	m.versionCursor = 2

	// First package has a version - should show it
	view := m.View()
	if !strings.Contains(view, "chainlink") {
		t.Error("expected view to contain package name 'chainlink'")
	}
	if !strings.Contains(view, "current: 2.1.0") {
		t.Error("expected view to contain 'current: 2.1.0'")
	}

	// Second package also has a version
	m.currentPkgIndex = 1
	view = m.View()
	if !strings.Contains(view, "contracts") {
		t.Error("expected view to contain package name 'contracts'")
	}
	if !strings.Contains(view, "current: 0.5.3") {
		t.Error("expected view to contain 'current: 0.5.3'")
	}

	// Third package has no version - should not show version info
	m.currentPkgIndex = 2
	view = m.View()
	if !strings.Contains(view, "noversion") {
		t.Error("expected view to contain package name 'noversion'")
	}
	if strings.Contains(view, "current:") {
		t.Error("expected view to NOT contain 'current:' for package without version")
	}
}

func TestSelectionWithGroupedDisplay(t *testing.T) {
	packages := []discovery.Package{
		{Name: "pkg-a", Path: "packages/a"},
		{Name: "pkg-b", Path: "packages/b"},
		{Name: "pkg-c", Path: "packages/c"},
	}

	changedPkgs := map[string]bool{
		"pkg-b": true, // pkg-b is at original index 1, but will be at display index 0
	}

	m := NewModelWithChanged(packages, changedPkgs)

	// Verify display order
	if m.displayItems[0].pkg.Name != "pkg-b" {
		t.Fatalf("expected first display item to be pkg-b, got %s", m.displayItems[0].pkg.Name)
	}
	if m.displayItems[0].origIndex != 1 {
		t.Fatalf("expected origIndex=1 for pkg-b, got %d", m.displayItems[0].origIndex)
	}

	// Simulate selecting the first item (pkg-b at display index 0)
	m.cursor = 0
	origIdx := m.displayItems[m.cursor].origIndex
	m.selected[origIdx] = true

	// Verify selection is stored by original index
	if !m.selected[1] {
		t.Error("expected pkg-b (origIndex 1) to be selected")
	}
	if m.selected[0] {
		t.Error("pkg-a (origIndex 0) should not be selected")
	}

	// Verify the correct package would be collected
	var selectedPackages []discovery.Package
	for i, pkg := range m.packages {
		if m.selected[i] {
			selectedPackages = append(selectedPackages, pkg)
		}
	}

	if len(selectedPackages) != 1 {
		t.Fatalf("expected 1 selected package, got %d", len(selectedPackages))
	}
	if selectedPackages[0].Name != "pkg-b" {
		t.Errorf("expected pkg-b to be selected, got %s", selectedPackages[0].Name)
	}
}
