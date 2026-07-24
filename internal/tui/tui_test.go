package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

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

	// Verify rows include section headers
	// Layout: [changed header, pkg-b, pkg-d, unchanged header, pkg-a, pkg-c]
	if len(m.rows) != 6 {
		t.Fatalf("expected 6 rows, got %d", len(m.rows))
	}
	if !m.rows[0].isHeader || m.rows[0].headerIdx != 0 {
		t.Error("expected row 0 to be changed header")
	}
	if m.rows[1].isHeader || m.rows[1].item.pkg.Name != "pkg-b" {
		t.Error("expected row 1 to be pkg-b")
	}
	if !m.rows[3].isHeader || m.rows[3].headerIdx != 1 {
		t.Error("expected row 3 to be unchanged header")
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
	// Only changed header + 2 packages (no unchanged header)
	if len(m.rows) != 3 {
		t.Errorf("expected 3 rows, got %d", len(m.rows))
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
	// Only unchanged header + 2 packages
	if len(m.rows) != 3 {
		t.Errorf("expected 3 rows, got %d", len(m.rows))
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

	// With rows-based approach, cursor maps directly to line number:
	// Row 0: "changed packages" header (cursor 0)
	// Row 1: pkg-a (cursor 1)
	// Row 2: pkg-b (cursor 2)
	// Row 3: "unchanged packages" header (cursor 3)
	// Row 4: pkg-c (cursor 4)
	// Row 5: pkg-d (cursor 5)

	tests := []struct {
		cursor   int
		expected int
	}{
		{cursor: 0, expected: 0},
		{cursor: 1, expected: 1},
		{cursor: 2, expected: 2},
		{cursor: 3, expected: 3},
		{cursor: 4, expected: 4},
		{cursor: 5, expected: 5},
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

func TestSectionToggle(t *testing.T) {
	packages := []discovery.Package{
		{Name: "pkg-a", Path: "packages/a"},
		{Name: "pkg-b", Path: "packages/b"},
		{Name: "pkg-c", Path: "packages/c"},
	}

	changedPkgs := map[string]bool{
		"pkg-a": true,
		"pkg-b": true,
	}

	m := NewModelWithChanged(packages, changedPkgs)

	// Toggle changed section (headerIdx 0) - should select all changed
	m.toggleSection(0)
	if !m.selected[m.displayItems[0].origIndex] {
		t.Error("expected pkg-a to be selected after toggle")
	}
	if !m.selected[m.displayItems[1].origIndex] {
		t.Error("expected pkg-b to be selected after toggle")
	}

	// Verify sectionAllSelected returns true
	if !m.sectionAllSelected(0) {
		t.Error("expected sectionAllSelected(0) to be true")
	}

	// Toggle again - should deselect all changed
	m.toggleSection(0)
	if m.selected[m.displayItems[0].origIndex] {
		t.Error("expected pkg-a to be deselected after second toggle")
	}
	if m.selected[m.displayItems[1].origIndex] {
		t.Error("expected pkg-b to be deselected after second toggle")
	}

	// Toggle unchanged section (headerIdx 1) - should select pkg-c
	m.toggleSection(1)
	if !m.selected[m.displayItems[2].origIndex] {
		t.Error("expected pkg-c to be selected after unchanged toggle")
	}
	// Changed should still be deselected
	if m.selected[m.displayItems[0].origIndex] {
		t.Error("changed pkg-a should still be deselected")
	}
}

func TestSectionTogglePartialSelection(t *testing.T) {
	packages := []discovery.Package{
		{Name: "pkg-a", Path: "packages/a"},
		{Name: "pkg-b", Path: "packages/b"},
		{Name: "pkg-c", Path: "packages/c"},
	}

	changedPkgs := map[string]bool{
		"pkg-a": true,
		"pkg-b": true,
	}

	m := NewModelWithChanged(packages, changedPkgs)

	// Select only pkg-a (partial selection of changed section)
	m.selected[m.displayItems[0].origIndex] = true

	if m.sectionAllSelected(0) {
		t.Error("sectionAllSelected(0) should be false with partial selection")
	}

	// Toggle changed section - since not all selected, should select all
	m.toggleSection(0)
	if !m.selected[m.displayItems[0].origIndex] || !m.selected[m.displayItems[1].origIndex] {
		t.Error("expected all changed packages to be selected")
	}
}

func TestVersionSelectionPnpmStyle(t *testing.T) {
	packages := []discovery.Package{
		{Name: "chainlink", Path: "packages/chainlink", Version: "2.1.0"},
		{Name: "contracts", Path: "packages/contracts", Version: "0.5.3"},
		{Name: "noversion", Path: "packages/noversion"},
	}

	m := NewModel(packages)
	m.selectedPackages = packages
	m.versionTypes = make(map[int]changeset.VersionType)
	m.versionStep = changeset.Major
	m.versionSelected = make(map[int]bool)
	m.versionRows = m.buildVersionRows()
	m.state = StateSelectVersion

	// View should show "major bump" title
	view := m.View()
	if !strings.Contains(view, "major") {
		t.Error("expected view to contain 'major'")
	}

	// Should show "all packages" header
	if !strings.Contains(view, "all packages") {
		t.Error("expected view to contain 'all packages'")
	}

	// Should show package names with versions
	if !strings.Contains(view, "chainlink@2.1.0") {
		t.Error("expected view to contain 'chainlink@2.1.0'")
	}
	if !strings.Contains(view, "contracts@0.5.3") {
		t.Error("expected view to contain 'contracts@0.5.3'")
	}
	if !strings.Contains(view, "noversion") {
		t.Error("expected view to contain 'noversion'")
	}
}

func TestVersionAllToggle(t *testing.T) {
	packages := []discovery.Package{
		{Name: "pkg-a", Path: "packages/a"},
		{Name: "pkg-b", Path: "packages/b"},
		{Name: "pkg-c", Path: "packages/c"},
	}

	m := NewModel(packages)
	m.selectedPackages = packages
	m.versionTypes = make(map[int]changeset.VersionType)
	m.versionStep = changeset.Major
	m.versionSelected = make(map[int]bool)
	m.versionRows = m.buildVersionRows()

	// Toggle all
	m.toggleAllVersionPackages()
	if !m.versionAllSelected() {
		t.Error("expected all version packages to be selected")
	}

	for i := range packages {
		if !m.versionSelected[i] {
			t.Errorf("expected package %d to be selected", i)
		}
	}

	// Toggle again - should deselect all
	m.toggleAllVersionPackages()
	if m.versionAllSelected() {
		t.Error("expected no version packages to be selected")
	}
}

func TestAssignRemainingAsPatch(t *testing.T) {
	packages := []discovery.Package{
		{Name: "pkg-a", Path: "packages/a"},
		{Name: "pkg-b", Path: "packages/b"},
		{Name: "pkg-c", Path: "packages/c"},
	}

	m := NewModel(packages)
	m.selectedPackages = packages
	m.versionTypes = make(map[int]changeset.VersionType)

	// Assign pkg-a as major manually
	m.versionTypes[0] = changeset.Major

	// Assign remaining as patch
	m.assignRemainingAsPatch()

	if m.versionTypes[0] != changeset.Major {
		t.Error("expected pkg-a to remain major")
	}
	if m.versionTypes[1] != changeset.Patch {
		t.Error("expected pkg-b to be patch")
	}
	if m.versionTypes[2] != changeset.Patch {
		t.Error("expected pkg-c to be patch")
	}
}

func TestBuildVersionRowsSkipsAssigned(t *testing.T) {
	packages := []discovery.Package{
		{Name: "pkg-a", Path: "packages/a"},
		{Name: "pkg-b", Path: "packages/b"},
		{Name: "pkg-c", Path: "packages/c"},
	}

	m := NewModel(packages)
	m.selectedPackages = packages
	m.versionTypes = make(map[int]changeset.VersionType)

	// Assign pkg-a as major
	m.versionTypes[0] = changeset.Major

	rows := m.buildVersionRows()

	// Should have: header + pkg-b + pkg-c (pkg-a already assigned)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if !rows[0].isHeader {
		t.Error("expected first row to be header")
	}
	if rows[1].pkgIndex != 1 {
		t.Errorf("expected second row to be pkg-b (index 1), got %d", rows[1].pkgIndex)
	}
	if rows[2].pkgIndex != 2 {
		t.Errorf("expected third row to be pkg-c (index 2), got %d", rows[2].pkgIndex)
	}
}

func TestSelectionWithGroupedDisplay(t *testing.T) {
	packages := []discovery.Package{
		{Name: "pkg-a", Path: "packages/a"},
		{Name: "pkg-b", Path: "packages/b"},
		{Name: "pkg-c", Path: "packages/c"},
	}

	changedPkgs := map[string]bool{
		"pkg-b": true, // pkg-b is at original index 1
	}

	m := NewModelWithChanged(packages, changedPkgs)

	// Verify display order
	if m.displayItems[0].pkg.Name != "pkg-b" {
		t.Fatalf("expected first display item to be pkg-b, got %s", m.displayItems[0].pkg.Name)
	}
	if m.displayItems[0].origIndex != 1 {
		t.Fatalf("expected origIndex=1 for pkg-b, got %d", m.displayItems[0].origIndex)
	}

	// Rows layout: [changed header, pkg-b, unchanged header, pkg-a, pkg-c]
	// Cursor 1 = pkg-b row
	m.cursor = 1
	row := m.rows[m.cursor]
	origIdx := row.item.origIndex
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

func TestFixedVersionTypeSkipsVersionSelection(t *testing.T) {
	packages := []discovery.Package{
		{Name: "pkg-a", Path: "packages/a"},
		{Name: "pkg-b", Path: "packages/b"},
		{Name: "pkg-c", Path: "packages/c"},
	}

	m := NewModelWithChanged(packages, map[string]bool{"pkg-a": true, "pkg-b": true})
	vt := changeset.Minor
	m.fixedVersionType = &vt

	// Select all packages via the changed section header (cursor starts at row 0)
	// Row 0 = changed header, toggling selects pkg-a and pkg-b
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	m = updated.(Model)

	// Move to unchanged header and toggle it too
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	m = updated.(Model)

	// Verify all 3 packages are selected
	for i := range packages {
		if !m.selected[i] {
			t.Errorf("expected package %d to be selected", i)
		}
	}

	// Press enter to confirm package selection
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	// Should skip StateSelectVersion and go directly to StateEnterSummary
	if m.state != StateEnterSummary {
		t.Fatalf("expected state StateEnterSummary, got %d", m.state)
	}

	// All 3 selected packages should have minor version type
	if len(m.selectedPackages) != 3 {
		t.Fatalf("expected 3 selectedPackages, got %d", len(m.selectedPackages))
	}
	for i, pkg := range m.selectedPackages {
		got, ok := m.versionTypes[i]
		if !ok {
			t.Errorf("expected versionType for package %s to be set", pkg.Name)
		}
		if got != changeset.Minor {
			t.Errorf("expected versionType for %s to be minor, got %s", pkg.Name, got)
		}
	}
}

func TestSummaryInputMultiLineConfirm(t *testing.T) {
	packages := []discovery.Package{{Name: "pkg-a", Path: "packages/a"}}

	m := NewModelWithChanged(packages, map[string]bool{"pkg-a": true})
	vt := changeset.Patch
	m.fixedVersionType = &vt

	// Select the changed package and confirm to reach the summary step.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.state != StateEnterSummary {
		t.Fatalf("expected state StateEnterSummary, got %d", m.state)
	}

	// Type a first line, then Enter to insert a newline (multi-line input),
	// then a second line.
	for _, r := range "line one" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	for _, r := range "line two" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}

	// Enter should NOT confirm; still in the summary step.
	if m.state != StateEnterSummary {
		t.Fatalf("enter should insert a newline, not confirm; state=%d", m.state)
	}

	// Ctrl+D confirms.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = updated.(Model)

	if m.state != StateDone {
		t.Fatalf("expected state StateDone after ctrl+d, got %d", m.state)
	}
	if m.result == nil {
		t.Fatal("expected a changeset result")
	}
	if want := "line one\nline two"; m.result.Summary != want {
		t.Errorf("expected multi-line summary %q, got %q", want, m.result.Summary)
	}
}
