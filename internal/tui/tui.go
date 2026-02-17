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

	// Package selection
	cursor   int
	selected map[int]bool // keyed by origIndex

	// Viewport for scrolling
	windowHeight int
	scrollOffset int

	// Version selection for each selected package
	selectedPackages []discovery.Package
	versionCursor    int
	currentPkgIndex  int
	versionTypes     []changeset.VersionType

	// Summary input
	textInput textinput.Model
	summary   string

	// Result
	result *changeset.Changeset
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

	return m
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
	// Determine the list length based on whether we have grouped display
	listLen := len(m.packages)
	if len(m.displayItems) > 0 {
		listLen = len(m.displayItems)
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
			// Toggle selection
			if len(m.displayItems) > 0 {
				// Using grouped display
				origIdx := m.displayItems[m.cursor].origIndex
				m.selected[origIdx] = !m.selected[origIdx]
			} else {
				m.selected[m.cursor] = !m.selected[m.cursor]
			}

		case "enter":
			// Collect selected packages and move to version selection
			for i, pkg := range m.packages {
				if m.selected[i] {
					m.selectedPackages = append(m.selectedPackages, pkg)
				}
			}

			if len(m.selectedPackages) == 0 {
				// No packages selected, don't proceed
				return m, nil
			}

			m.versionTypes = make([]changeset.VersionType, len(m.selectedPackages))
			m.state = StateSelectVersion
			m.currentPkgIndex = 0
			m.versionCursor = 2 // Default to patch
		}
	}
	return m, nil
}

// ensureCursorVisible adjusts scroll offset to keep cursor in view
func (m *Model) ensureCursorVisible() {
	// Calculate visible area (leave room for header and footer)
	visibleLines := max(
		// title, blank, help, blank lines
		m.windowHeight-6, 5)

	// Calculate the actual line number in the rendered output
	// that corresponds to the current cursor position
	lineNum := m.cursorToLineNumber()

	// When at the first item of a section, scroll up to show the header too
	targetScrollOffset := lineNum
	if m.cursor == 0 {
		// First item - show the section header
		targetScrollOffset = 0
	} else if m.cursor == m.changedCount && m.changedCount > 0 && m.unchangedCount > 0 {
		// First item of unchanged section - show that section header
		// lineNum includes both headers, so subtract 1 to show "unchanged packages" header
		targetScrollOffset = lineNum - 1
	}

	if targetScrollOffset < m.scrollOffset {
		m.scrollOffset = targetScrollOffset
	} else if lineNum >= m.scrollOffset+visibleLines {
		m.scrollOffset = lineNum - visibleLines + 1
	}
}

// cursorToLineNumber converts cursor position to line number in rendered output
func (m Model) cursorToLineNumber() int {
	if len(m.displayItems) == 0 {
		return m.cursor
	}

	// Account for section headers
	lineNum := m.cursor

	// If we have changed packages, add 1 for the "changed packages" header
	if m.changedCount > 0 {
		lineNum++
	}

	// If cursor is in unchanged section and we have unchanged packages,
	// add 1 for the "unchanged packages" header
	if m.unchangedCount > 0 && m.cursor >= m.changedCount {
		lineNum++
	}

	return lineNum
}

func (m Model) updateVersionSelection(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.versionCursor > 0 {
				m.versionCursor--
			}

		case "down", "j":
			if m.versionCursor < 2 {
				m.versionCursor++
			}

		case "enter":
			// Set version type for current package
			switch m.versionCursor {
			case 0:
				m.versionTypes[m.currentPkgIndex] = changeset.Major
			case 1:
				m.versionTypes[m.currentPkgIndex] = changeset.Minor
			case 2:
				m.versionTypes[m.currentPkgIndex] = changeset.Patch
			}

			// Move to next package or summary input
			m.currentPkgIndex++
			if m.currentPkgIndex >= len(m.selectedPackages) {
				m.state = StateEnterSummary
				m.textInput.Focus()
				return m, textinput.Blink
			}
			m.versionCursor = 2 // Reset to patch default
		}
	}
	return m, nil
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

	// Build all lines first
	var lines []string

	if len(m.displayItems) > 0 {
		// Grouped display mode
		if m.changedCount > 0 {
			lines = append(lines, sectionStyle.Render("changed packages"))
			for i := 0; i < m.changedCount; i++ {
				lines = append(lines, m.renderPackageItem(i))
			}
		}
		if m.unchangedCount > 0 {
			lines = append(lines, sectionStyle.Render("unchanged packages"))
			for i := m.changedCount; i < len(m.displayItems); i++ {
				lines = append(lines, m.renderPackageItem(i))
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

// renderPackageItem renders a package item for grouped display
func (m Model) renderPackageItem(displayIdx int) string {
	item := m.displayItems[displayIdx]
	isCursor := m.cursor == displayIdx
	isSelected := m.selected[item.origIndex]

	// Use consistent ASCII-only prefix
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

	// Use consistent ASCII-only prefix
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

	pkg := m.selectedPackages[m.currentPkgIndex]
	sb.WriteString(titleStyle.Render("Select version bump for " + pkg.Name))
	sb.WriteString("\n\n")

	versionOptions := []struct {
		name string
		desc string
	}{
		{"major", "Breaking changes"},
		{"minor", "New features (backwards compatible)"},
		{"patch", "Bug fixes"},
	}

	for i, opt := range versionOptions {
		var cursor string
		if m.versionCursor == i {
			cursor = "> "
		} else {
			cursor = "  "
		}

		name := opt.name
		if m.versionCursor == i {
			name = selectedStyle.Render(name)
		}

		desc := dimStyle.Render(" - " + opt.desc)
		fmt.Fprintf(&sb, "%s%s%s\n", cursor, name, desc)
	}

	sb.WriteString("\n")
	sb.WriteString(helpStyle.Render("up/down navigate | enter confirm | q quit"))

	return sb.String()
}

func (m Model) viewSummaryInput() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("Enter changelog message"))
	sb.WriteString("\n\n")

	// Show selected packages and versions
	var pkgLines strings.Builder
	pkgLines.WriteString("Selected packages:\n")
	for i, pkg := range m.selectedPackages {
		fmt.Fprintf(&pkgLines, "  - %s: %s\n", pkg.Name, m.versionTypes[i])
	}
	sb.WriteString(dimStyle.Render(pkgLines.String()))
	sb.WriteString("\n")

	sb.WriteString(m.textInput.View())
	sb.WriteString("\n\n")
	sb.WriteString(helpStyle.Render("enter confirm | ctrl+c quit"))

	return sb.String()
}

// Result returns the changeset result after the TUI completes.
func (m Model) Result() *changeset.Changeset {
	return m.result
}

// Run starts the TUI and returns the resulting changeset.
func Run(packages []discovery.Package) (*changeset.Changeset, error) {
	return RunWithChanged(packages, nil)
}

// RunWithChanged starts the TUI with packages grouped by changed status.
// Changed packages are shown first but not pre-selected.
func RunWithChanged(packages []discovery.Package, changedPkgs map[string]bool) (*changeset.Changeset, error) {
	if len(packages) == 0 {
		return nil, errors.New("no packages found")
	}

	var model Model
	if len(changedPkgs) > 0 {
		model = NewModelWithChanged(packages, changedPkgs)
	} else {
		model = NewModel(packages)
		// For non-grouped display, populate displayItems for consistency
		for i, pkg := range packages {
			model.displayItems = append(model.displayItems, packageItem{
				pkg:       pkg,
				origIndex: i,
				changed:   false,
			})
		}
	}

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
	return RunWithChanged(packages, preselected)
}
