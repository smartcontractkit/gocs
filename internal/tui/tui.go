// Package tui provides an interactive terminal UI for creating changesets.
package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/smartcontractkit/gocs/internal/changeset"
	"github.com/smartcontractkit/gocs/internal/discovery"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212"))

	checkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99"))
)

// State represents the current step in the TUI flow.
type State int

const (
	StateSelectPackages State = iota
	StateSelectVersion
	StateEnterSummary
	StateDone
)

// displayRow represents a row in the package list: either a section header or a package item.
type displayRow struct {
	isHeader  bool
	headerIdx int // 0 = changed header, 1 = unchanged header
	item      packageItem
}

// packageItem represents a package in the display list with its metadata
type packageItem struct {
	pkg       discovery.Package
	origIndex int // original index in the full package list
	changed   bool
}

// Model represents the TUI state.
type Model struct {
	state State

	// All packages (original order for selection tracking)
	packages []discovery.Package

	// Display items (grouped: changed first, then unchanged)
	displayItems   []packageItem
	changedCount   int
	unchangedCount int

	// Flattened rows for cursor navigation (includes section headers)
	rows []displayRow

	// Package selection
	cursor   int
	selected map[int]bool // keyed by origIndex

	// Viewport for scrolling
	windowHeight int
	scrollOffset int

	// Version selection (pnpm-style: major step, then minor step, rest = patch)
	selectedPackages []discovery.Package
	versionCursor    int
	versionSelected  map[int]bool                  // keyed by index in selectedPackages for current version step
	versionStep      changeset.VersionType         // current step: Major then Minor
	versionTypes     map[int]changeset.VersionType // keyed by index in selectedPackages
	// versionRows for the version selection list (all packages header + individual)
	versionRows []versionDisplayRow

	// Fixed version type: when set, skip version selection entirely
	fixedVersionType *changeset.VersionType

	// Summary input
	textInput textinput.Model
	summary   string

	// Result
	result *changeset.Changeset
}

// versionDisplayRow represents a row in the version selection list
type versionDisplayRow struct {
	isHeader bool // "all packages" header
	pkgIndex int  // index into selectedPackages (-1 for header)
}

// NewModel creates a new TUI model with the given packages.
func NewModel(packages []discovery.Package) Model {
	ti := textinput.New()
	ti.Placeholder = "Enter changelog message..."
	ti.CharLimit = 500
	ti.Width = 60

	return Model{
		state:        StateSelectPackages,
		packages:     packages,
		displayItems: make([]packageItem, 0, len(packages)),
		selected:     make(map[int]bool),
		textInput:    ti,
		windowHeight: 24, // default, will be updated
	}
}

// NewModelWithChanged creates a new TUI model with packages grouped by changed status.
func NewModelWithChanged(packages []discovery.Package, changedPkgs map[string]bool) Model {
	m := NewModel(packages)

	// Build display items: changed packages first, then unchanged
	var changed, unchanged []packageItem

	for i, pkg := range packages {
		isChanged := changedPkgs[pkg.Name] || changedPkgs[pkg.Path]
		item := packageItem{
			pkg:       pkg,
			origIndex: i,
			changed:   isChanged,
		}
		if isChanged {
			changed = append(changed, item)
		} else {
			unchanged = append(unchanged, item)
		}
	}

	m.changedCount = len(changed)
	m.unchangedCount = len(unchanged)
	m.displayItems = append(changed, unchanged...)

	// Build flattened rows with section headers as selectable items
	m.rows = buildDisplayRows(m.displayItems, m.changedCount, m.unchangedCount)

	return m
}

// buildDisplayRows creates the flattened row list including section headers.
func buildDisplayRows(items []packageItem, changedCount, unchangedCount int) []displayRow {
	var rows []displayRow

	if changedCount > 0 {
		rows = append(rows, displayRow{isHeader: true, headerIdx: 0})
		for i := 0; i < changedCount; i++ {
			rows = append(rows, displayRow{item: items[i]})
		}
	}

	if unchangedCount > 0 {
		rows = append(rows, displayRow{isHeader: true, headerIdx: 1})
		for i := changedCount; i < len(items); i++ {
			rows = append(rows, displayRow{item: items[i]})
		}
	}

	return rows
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle window resize for all states
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.windowHeight = msg.Height
	}

	switch m.state {
	case StateSelectPackages:
		return m.updatePackageSelection(msg)
	case StateSelectVersion:
		return m.updateVersionSelection(msg)
	case StateEnterSummary:
		return m.updateSummaryInput(msg)
	case StateDone:
		return m, nil
	}
	return m, nil
}

func (m Model) updatePackageSelection(msg tea.Msg) (tea.Model, tea.Cmd) {
	listLen := len(m.rows)
	if listLen == 0 {
		// Fallback for simple mode (no grouped display)
		listLen = len(m.packages)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.ensureCursorVisible()
			}

		case "down", "j":
			if m.cursor < listLen-1 {
				m.cursor++
				m.ensureCursorVisible()
			}

		case " ":
			if len(m.rows) > 0 {
				row := m.rows[m.cursor]
				if row.isHeader {
					// Toggle all packages in this section
					m.toggleSection(row.headerIdx)
				} else {
					origIdx := row.item.origIndex
					m.selected[origIdx] = !m.selected[origIdx]
				}
			} else {
				m.selected[m.cursor] = !m.selected[m.cursor]
			}

		case "enter":
			// Collect selected packages
			for i, pkg := range m.packages {
				if m.selected[i] {
					m.selectedPackages = append(m.selectedPackages, pkg)
				}
			}

			if len(m.selectedPackages) == 0 {
				return m, nil
			}

			// If a fixed version type was provided, skip version selection
			if m.fixedVersionType != nil {
				m.versionTypes = make(map[int]changeset.VersionType, len(m.selectedPackages))
				for i := range m.selectedPackages {
					m.versionTypes[i] = *m.fixedVersionType
				}
				m.state = StateEnterSummary
				m.textInput.Focus()
				return m, textinput.Blink
			}

			// Start pnpm-style version selection: major first
			m.versionTypes = make(map[int]changeset.VersionType, len(m.selectedPackages))
			m.versionStep = changeset.Major
			m.versionSelected = make(map[int]bool)
			m.versionRows = m.buildVersionRows()
			m.versionCursor = 0
			m.scrollOffset = 0
			m.state = StateSelectVersion
		}
	}
	return m, nil
}

// toggleSection toggles all packages in a section (changed=0, unchanged=1).
func (m *Model) toggleSection(headerIdx int) {
	// Determine which display items belong to this section
	var start, end int
	if headerIdx == 0 {
		// Changed section
		start, end = 0, m.changedCount
	} else {
		// Unchanged section
		start, end = m.changedCount, m.changedCount+m.unchangedCount
	}

	// Check if all in this section are currently selected
	allSelected := true
	for i := start; i < end; i++ {
		if !m.selected[m.displayItems[i].origIndex] {
			allSelected = false
			break
		}
	}

	// Toggle: if all selected, deselect all; otherwise select all
	for i := start; i < end; i++ {
		m.selected[m.displayItems[i].origIndex] = !allSelected
	}
}

// sectionAllSelected returns true if all packages in a section are selected.
func (m Model) sectionAllSelected(headerIdx int) bool {
	var start, end int
	if headerIdx == 0 {
		start, end = 0, m.changedCount
	} else {
		start, end = m.changedCount, m.changedCount+m.unchangedCount
	}
	if start == end {
		return false
	}
	for i := start; i < end; i++ {
		if !m.selected[m.displayItems[i].origIndex] {
			return false
		}
	}
	return true
}

// ensureCursorVisible adjusts scroll offset to keep cursor in view
func (m *Model) ensureCursorVisible() {
	visibleLines := max(m.windowHeight-6, 5)

	lineNum := m.cursor // rows already include headers, so cursor == line number

	if lineNum < m.scrollOffset {
		m.scrollOffset = lineNum
	} else if lineNum >= m.scrollOffset+visibleLines {
		m.scrollOffset = lineNum - visibleLines + 1
	}
}

// cursorToLineNumber converts cursor position to line number in rendered output.
// With the new rows-based approach, cursor position maps directly to line number.
func (m Model) cursorToLineNumber() int {
	return m.cursor
}

// buildVersionRows builds the display rows for the version selection step.
func (m Model) buildVersionRows() []versionDisplayRow {
	var rows []versionDisplayRow
	// "all packages" / "all remaining packages" header
	rows = append(rows, versionDisplayRow{isHeader: true, pkgIndex: -1})
	for i := range m.selectedPackages {
		// Only show packages not yet assigned a version type
		if _, assigned := m.versionTypes[i]; !assigned {
			rows = append(rows, versionDisplayRow{pkgIndex: i})
		}
	}
	return rows
}

func (m Model) updateVersionSelection(msg tea.Msg) (tea.Model, tea.Cmd) {
	listLen := len(m.versionRows)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.versionCursor > 0 {
				m.versionCursor--
				m.ensureVersionCursorVisible()
			}

		case "down", "j":
			if m.versionCursor < listLen-1 {
				m.versionCursor++
				m.ensureVersionCursorVisible()
			}

		case " ":
			row := m.versionRows[m.versionCursor]
			if row.isHeader {
				// Toggle all displayed packages
				m.toggleAllVersionPackages()
			} else {
				m.versionSelected[row.pkgIndex] = !m.versionSelected[row.pkgIndex]
			}

		case "enter":
			// Assign current version step to all selected packages
			for idx, sel := range m.versionSelected {
				if sel {
					m.versionTypes[idx] = m.versionStep
				}
			}

			if m.versionStep == changeset.Major {
				// Move to minor step
				m.versionStep = changeset.Minor
				m.versionSelected = make(map[int]bool)
				m.versionRows = m.buildVersionRows()
				m.versionCursor = 0
				m.scrollOffset = 0

				// If all packages already assigned, skip to summary
				if len(m.versionRows) <= 1 {
					// Only header row left, assign remaining as patch
					m.assignRemainingAsPatch()
					m.state = StateEnterSummary
					m.textInput.Focus()
					return m, textinput.Blink
				}
			} else {
				// After minor step, assign remaining packages as patch
				m.assignRemainingAsPatch()
				m.state = StateEnterSummary
				m.textInput.Focus()
				return m, textinput.Blink
			}
		}
	}
	return m, nil
}

// ensureVersionCursorVisible adjusts scroll offset for the version selection list
func (m *Model) ensureVersionCursorVisible() {
	visibleLines := max(m.windowHeight-6, 5)
	if m.versionCursor < m.scrollOffset {
		m.scrollOffset = m.versionCursor
	} else if m.versionCursor >= m.scrollOffset+visibleLines {
		m.scrollOffset = m.versionCursor - visibleLines + 1
	}
}

// toggleAllVersionPackages toggles all packages in the version selection list.
func (m *Model) toggleAllVersionPackages() {
	// Check if all are selected
	allSelected := true
	for _, row := range m.versionRows {
		if row.isHeader {
			continue
		}
		if !m.versionSelected[row.pkgIndex] {
			allSelected = false
			break
		}
	}

	for _, row := range m.versionRows {
		if row.isHeader {
			continue
		}
		m.versionSelected[row.pkgIndex] = !allSelected
	}
}

// versionAllSelected returns true if all packages in the version list are selected.
func (m Model) versionAllSelected() bool {
	count := 0
	for _, row := range m.versionRows {
		if row.isHeader {
			continue
		}
		count++
		if !m.versionSelected[row.pkgIndex] {
			return false
		}
	}
	return count > 0
}

// assignRemainingAsPatch assigns patch version to any selected packages not yet assigned.
func (m *Model) assignRemainingAsPatch() {
	for i := range m.selectedPackages {
		if _, assigned := m.versionTypes[i]; !assigned {
			m.versionTypes[i] = changeset.Patch
		}
	}
}

func (m Model) updateSummaryInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "enter":
			m.summary = m.textInput.Value()
			if m.summary == "" {
				return m, nil
			}

			// Build the changeset
			entries := make([]changeset.Entry, len(m.selectedPackages))
			for i, pkg := range m.selectedPackages {
				entries[i] = changeset.Entry{
					Package:     pkg.Name,
					VersionType: m.versionTypes[i],
				}
			}

			m.result = &changeset.Changeset{
				Entries: entries,
				Summary: m.summary,
			}
			m.state = StateDone
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// View renders the TUI.
func (m Model) View() string {
	switch m.state {
	case StateSelectPackages:
		return m.viewPackageSelection()
	case StateSelectVersion:
		return m.viewVersionSelection()
	case StateEnterSummary:
		return m.viewSummaryInput()
	case StateDone:
		return ""
	}
	return ""
}

func (m Model) viewPackageSelection() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Which packages would you like to include?"))
	sb.WriteString("\n\n")

	// Calculate visible area
	visibleLines := max(m.windowHeight-6, 5)

	// Build all lines
	var lines []string

	if len(m.rows) > 0 {
		for i, row := range m.rows {
			if row.isHeader {
				lines = append(lines, m.renderSectionHeader(i, row.headerIdx))
			} else {
				lines = append(lines, m.renderPackageRow(i, row))
			}
		}
	} else {
		// Simple list mode (no changed info)
		for i := range m.packages {
			lines = append(lines, m.renderPackageItemSimple(i))
		}
	}

	// Apply scrolling
	startIdx := max(m.scrollOffset, 0)
	endIdx := min(startIdx+visibleLines, len(lines))
	if startIdx > len(lines) {
		startIdx = len(lines)
	}

	// Show scroll indicator at top if needed
	if startIdx > 0 {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("^ %d more above", startIdx)))
		sb.WriteString("\n")
	}

	// Render visible lines
	for i := startIdx; i < endIdx; i++ {
		sb.WriteString(lines[i])
		sb.WriteString("\n")
	}

	// Show scroll indicator at bottom if needed
	remaining := len(lines) - endIdx
	if remaining > 0 {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("v %d more below", remaining)))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("up/down navigate | space select | enter confirm | q quit"))

	return sb.String()
}

// renderSectionHeader renders a section header as a selectable row.
func (m Model) renderSectionHeader(cursorIdx, headerIdx int) string {
	isCursor := m.cursor == cursorIdx
	allSel := m.sectionAllSelected(headerIdx)

	var cursor string
	if isCursor {
		cursor = "> "
	} else {
		cursor = "  "
	}

	var checked string
	if allSel {
		checked = checkStyle.Render("[x]")
	} else {
		checked = "[ ]"
	}

	var label string
	if headerIdx == 0 {
		label = "changed packages"
	} else {
		label = "unchanged packages"
	}

	name := sectionStyle.Render(label)
	if isCursor {
		name = selectedStyle.Render(label)
	}

	return fmt.Sprintf("%s%s %s", cursor, checked, name)
}

// renderPackageRow renders a package item in the rows-based display.
func (m Model) renderPackageRow(cursorIdx int, row displayRow) string {
	item := row.item
	isCursor := m.cursor == cursorIdx
	isSelected := m.selected[item.origIndex]

	var cursor string
	if isCursor {
		cursor = "> "
	} else {
		cursor = "  "
	}

	var checked string
	if isSelected {
		checked = checkStyle.Render("[x]")
	} else {
		checked = "[ ]"
	}

	name := item.pkg.Name
	if isCursor {
		name = selectedStyle.Render(name)
	}

	path := ""
	if item.pkg.Path != "" {
		path = dimStyle.Render(fmt.Sprintf(" (%s)", item.pkg.Path))
	}

	return fmt.Sprintf("%s%s %s%s", cursor, checked, name, path)
}

// renderPackageItemSimple renders a package item for simple (non-grouped) display
func (m Model) renderPackageItemSimple(idx int) string {
	pkg := m.packages[idx]
	isCursor := m.cursor == idx
	isSelected := m.selected[idx]

	var cursor string
	if isCursor {
		cursor = "> "
	} else {
		cursor = "  "
	}

	var checked string
	if isSelected {
		checked = checkStyle.Render("[x]")
	} else {
		checked = "[ ]"
	}

	name := pkg.Name
	if isCursor {
		name = selectedStyle.Render(name)
	}

	path := ""
	if pkg.Path != "" {
		path = dimStyle.Render(fmt.Sprintf(" (%s)", pkg.Path))
	}

	return fmt.Sprintf("%s%s %s%s", cursor, checked, name, path)
}

func (m Model) viewVersionSelection() string {
	var sb strings.Builder

	title := fmt.Sprintf("Which packages should have a %s bump?", string(m.versionStep))
	sb.WriteString(titleStyle.Render(title))
	sb.WriteString("\n\n")

	// Calculate visible area
	visibleLines := max(m.windowHeight-6, 5)

	var lines []string
	for i, row := range m.versionRows {
		if row.isHeader {
			lines = append(lines, m.renderVersionHeader(i))
		} else {
			lines = append(lines, m.renderVersionPackage(i, row))
		}
	}

	// Apply scrolling
	startIdx := max(m.scrollOffset, 0)
	endIdx := min(startIdx+visibleLines, len(lines))
	if startIdx > len(lines) {
		startIdx = len(lines)
	}

	if startIdx > 0 {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("^ %d more above", startIdx)))
		sb.WriteString("\n")
	}

	for i := startIdx; i < endIdx; i++ {
		sb.WriteString(lines[i])
		sb.WriteString("\n")
	}

	remaining := len(lines) - endIdx
	if remaining > 0 {
		sb.WriteString(dimStyle.Render(fmt.Sprintf("v %d more below", remaining)))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	if m.versionStep == changeset.Major {
		sb.WriteString(helpStyle.Render("up/down navigate | space select | enter confirm (unselected will be asked for minor) | q quit"))
	} else {
		sb.WriteString(helpStyle.Render("up/down navigate | space select | enter confirm (unselected will be patch) | q quit"))
	}

	return sb.String()
}

// renderVersionHeader renders the "all packages" toggle header for version selection.
func (m Model) renderVersionHeader(cursorIdx int) string {
	isCursor := m.versionCursor == cursorIdx
	allSel := m.versionAllSelected()

	var cursor string
	if isCursor {
		cursor = "> "
	} else {
		cursor = "  "
	}

	var checked string
	if allSel {
		checked = checkStyle.Render("[x]")
	} else {
		checked = "[ ]"
	}

	label := "all packages"
	// On minor step some packages may already be assigned major,
	// so the toggle only applies to remaining packages.
	if m.versionStep != changeset.Major {
		label = "all remaining packages"
	}
	if isCursor {
		label = selectedStyle.Render(label)
	} else {
		label = sectionStyle.Render(label)
	}

	return fmt.Sprintf("%s%s %s", cursor, checked, label)
}

// renderVersionPackage renders a package row in the version selection list.
func (m Model) renderVersionPackage(cursorIdx int, row versionDisplayRow) string {
	pkg := m.selectedPackages[row.pkgIndex]
	isCursor := m.versionCursor == cursorIdx
	isSelected := m.versionSelected[row.pkgIndex]

	var cursor string
	if isCursor {
		cursor = "> "
	} else {
		cursor = "  "
	}

	var checked string
	if isSelected {
		checked = checkStyle.Render("[x]")
	} else {
		checked = "[ ]"
	}

	name := pkg.Name
	if pkg.Version != "" {
		name += "@" + pkg.Version
	}
	if isCursor {
		name = selectedStyle.Render(name)
	}

	return fmt.Sprintf("%s%s %s", cursor, checked, name)
}

func (m Model) viewSummaryInput() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Enter changelog message"))
	sb.WriteString("\n\n")

	// Show a compact summary of selected packages grouped by version type
	sb.WriteString(dimStyle.Render(m.summarizeSelections()))
	sb.WriteString("\n")

	sb.WriteString(m.textInput.View())
	sb.WriteString("\n\n")
	sb.WriteString(helpStyle.Render("enter confirm | ctrl+c quit"))

	return sb.String()
}

// summarizeSelections returns a compact summary of selected packages.
// If all packages share the same version type, shows "N packages: type".
// Otherwise groups by version type and lists individual packages for small groups.
func (m Model) summarizeSelections() string {
	// Group packages by version type
	groups := make(map[changeset.VersionType][]string)
	order := []changeset.VersionType{changeset.Major, changeset.Minor, changeset.Patch}
	for i, pkg := range m.selectedPackages {
		vt := m.versionTypes[i]
		groups[vt] = append(groups[vt], pkg.Name)
	}

	// If all packages have the same version type, show a one-liner
	if len(groups) == 1 {
		for vt, pkgs := range groups {
			return fmt.Sprintf("%d packages: %s\n", len(pkgs), vt)
		}
	}

	// Multiple version types: show count per type, list names only for small groups
	const maxListSize = 5
	var sb strings.Builder
	for _, vt := range order {
		pkgs := groups[vt]
		if len(pkgs) == 0 {
			continue
		}
		if len(pkgs) <= maxListSize {
			fmt.Fprintf(&sb, "%s:\n", vt)
			for _, name := range pkgs {
				fmt.Fprintf(&sb, "  - %s\n", name)
			}
		} else {
			fmt.Fprintf(&sb, "%s: %d packages\n", vt, len(pkgs))
		}
	}
	return sb.String()
}

// Result returns the changeset result after the TUI completes.
func (m Model) Result() *changeset.Changeset {
	return m.result
}

// Run starts the TUI and returns the resulting changeset.
func Run(packages []discovery.Package) (*changeset.Changeset, error) {
	return RunWithChanged(packages, nil, nil)
}

// RunWithChanged starts the TUI with packages grouped by changed status.
// Changed packages are shown first but not pre-selected.
// If fixedVersionType is non-nil, the version selection step is skipped
// and all selected packages are assigned that version type.
func RunWithChanged(packages []discovery.Package, changedPkgs map[string]bool, fixedVersionType *changeset.VersionType) (*changeset.Changeset, error) {
	if len(packages) == 0 {
		return nil, errors.New("no packages found")
	}

	if changedPkgs == nil {
		changedPkgs = make(map[string]bool)
	}
	model := NewModelWithChanged(packages, changedPkgs)
	model.fixedVersionType = fixedVersionType

	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("TUI error: %w", err)
	}

	m, ok := finalModel.(Model)
	if !ok {
		return nil, errors.New("unexpected model type")
	}

	return m.Result(), nil
}

// RunWithPreselected is deprecated, use RunWithChanged instead.
// This is kept for backwards compatibility.
func RunWithPreselected(packages []discovery.Package, preselected map[string]bool) (*changeset.Changeset, error) {
	return RunWithChanged(packages, preselected, nil)
}
