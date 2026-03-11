package tui

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/ametel01/agents-toolbelt/internal/catalog"
	"github.com/ametel01/agents-toolbelt/internal/discovery"
	tea "github.com/charmbracelet/bubbletea"
)

type rowKind string

const (
	rowKindMore rowKind = "more"
	rowKindTool rowKind = "tool"
)

type row struct {
	kind rowKind
	tool catalog.Tool
}

// PickerModel is the Bubble Tea model for interactive tool selection.
type PickerModel struct {
	cursor       int
	expandedNice bool
	height       int
	query        string
	rows         []row
	searching    bool
	selected     map[string]bool
	snapshot     discovery.Snapshot
	tools        []catalog.Tool
	width        int
}

// RunPicker launches the interactive picker and returns the selected tools.
func RunPicker(tools []catalog.Tool, snapshot discovery.Snapshot) ([]catalog.Tool, error) {
	model := NewPickerModel(tools, snapshot)
	finalModel, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
	if err != nil {
		return nil, fmt.Errorf("run picker: %w", err)
	}

	return finalModel.(PickerModel).SelectedTools(), nil
}

// NewPickerModel constructs a picker model with no tools preselected.
func NewPickerModel(tools []catalog.Tool, snapshot discovery.Snapshot) PickerModel {
	ordered := slices.Clone(tools)
	sort.SliceStable(ordered, func(left, right int) bool {
		if ordered[left].Category == ordered[right].Category {
			return ordered[left].Name < ordered[right].Name
		}

		return ordered[left].Category < ordered[right].Category
	})

	model := PickerModel{
		selected: make(map[string]bool, len(ordered)),
		snapshot: snapshot,
		tools:    ordered,
	}
	model.rows = model.visibleRows()

	return model
}

// Init satisfies the Bubble Tea model interface.
func (m PickerModel) Init() tea.Cmd {
	return nil
}

// Update satisfies the Bubble Tea model interface.
func (m PickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(typed)
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height

		return m, nil
	default:
		return m, nil
	}
}

// View renders the picker state.
func (m PickerModel) View() string {
	var builder strings.Builder
	builder.WriteString(titleStyle.Render("Select CLI Tools"))
	builder.WriteString("\n\n")

	if m.searching {
		builder.WriteString("Search: ")
		builder.WriteString(m.query)
		builder.WriteString("\n\n")
	}

	if len(m.rows) == 0 {
		builder.WriteString(subtleStyle.Render("No matching tools."))

		return builder.String()
	}

	start, end := m.windowBounds()
	if start > 0 {
		builder.WriteString(subtleStyle.Render("... more tools above ..."))
		builder.WriteString("\n\n")
	}

	lastCategory := ""
	for index := start; index < end; index++ {
		row := m.rows[index]
		category := displayedCategory(row)
		if row.kind == rowKindTool && category != lastCategory {
			if lastCategory != "" {
				builder.WriteString("\n")
			}

			lastCategory = category
			builder.WriteString(titleStyle.Render(category))
			builder.WriteString("\n")
		}

		prefix := "  "
		if index == m.cursor {
			prefix = cursorStyle.Render("> ")
		}

		builder.WriteString(renderPrefixedRow(prefix, m.renderRow(row)))
	}

	if end < len(m.rows) {
		builder.WriteString("\n")
		builder.WriteString(subtleStyle.Render("... more tools below ..."))
	}

	builder.WriteString("\n")
	builder.WriteString(subtleStyle.Render("space toggle • a select all • n deselect all • / search • enter confirm • q quit"))

	return builder.String()
}

// SelectedTools returns the current selected tool list.
func (m PickerModel) SelectedTools() []catalog.Tool {
	selected := make([]catalog.Tool, 0, len(m.selected))
	for _, tool := range m.tools {
		if m.selected[tool.ID] {
			selected = append(selected, tool)
		}
	}

	return selected
}

func (m PickerModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.searching && msg.String() != "enter" && msg.String() != "esc" {
		return m.handleSearchInput(msg)
	}

	if handled, nextModel, cmd := m.handleQuitOrConfirm(msg); handled {
		return nextModel, cmd
	}

	if handled := m.handleCursor(msg.String()); handled {
		m.refreshRows()

		return m, nil
	}

	if handled := m.handleSelection(msg.String()); handled {
		m.refreshRows()

		return m, nil
	}

	if msg.String() == "/" {
		m.searching = true
		m.refreshRows()

		return m, nil
	}

	return m, nil
}

func (m *PickerModel) handleCursor(key string) bool {
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

		return true
	case "down", "j":
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}

		return true
	default:
		return false
	}
}

func (m *PickerModel) handleSelection(key string) bool {
	switch key {
	case " ", "space":
		m.toggleCurrent()

		return true
	case "a":
		for _, tool := range m.tools {
			m.selected[tool.ID] = true
		}

		return true
	case "n":
		m.selected = make(map[string]bool)

		return true
	default:
		return false
	}
}

func (m PickerModel) handleQuitOrConfirm(msg tea.KeyMsg) (bool, tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.searching {
			m.searching = false
			m.refreshRows()

			return true, m, nil
		}

		return true, m, tea.Quit
	case "q", "esc":
		return true, m, tea.Quit
	default:
		return false, m, nil
	}
}

func (m PickerModel) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyBackspace, tea.KeyDelete:
		if len(m.query) > 0 {
			m.query = m.query[:len(m.query)-1]
		}
	case tea.KeyEsc:
		m.searching = false
	case tea.KeyRunes:
		m.query += string(msg.Runes)
	}

	m.refreshRows()

	return m, nil
}

func (m *PickerModel) toggleCurrent() {
	if len(m.rows) == 0 || m.cursor >= len(m.rows) {
		return
	}

	current := m.rows[m.cursor]
	if current.kind == rowKindMore {
		m.expandedNice = !m.expandedNice

		return
	}

	m.selected[current.tool.ID] = !m.selected[current.tool.ID]
}

func (m *PickerModel) refreshRows() {
	m.rows = m.visibleRows()
	if len(m.rows) == 0 {
		m.cursor = 0

		return
	}

	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
}

func (m PickerModel) visibleRows() []row {
	rows := make([]row, 0, len(m.tools)+1)
	query := strings.ToLower(strings.TrimSpace(m.query))
	niceCount := 0

	for _, tool := range m.tools {
		if query != "" && !matchesQuery(tool, query) {
			continue
		}

		if tool.Tier == catalog.TierNice && !m.expandedNice {
			niceCount++

			continue
		}

		rows = append(rows, row{kind: rowKindTool, tool: tool})
	}

	if niceCount > 0 {
		rows = append(rows, row{
			kind: rowKindMore,
			tool: catalog.Tool{Name: fmt.Sprintf("More tools (%d)", niceCount)},
		})
	}

	return rows
}

func (m PickerModel) renderRow(row row) string {
	if row.kind == rowKindMore {
		return subtleStyle.Render("▸ " + row.tool.Name)
	}

	checkbox := "[ ]"
	if m.selected[row.tool.ID] {
		checkbox = selectedStyle.Render("[x]")
	}

	suffix := ""
	if presence, ok := m.snapshot.Tools[row.tool.ID]; ok && presence.Installed {
		suffix = subtleStyle.Render("  ✓ installed")
	}

	description := row.tool.Description
	if description == "" {
		return fmt.Sprintf("%s %s%s", checkbox, row.tool.Name, suffix)
	}

	return fmt.Sprintf(
		"%s %s%s\n%s",
		checkbox,
		row.tool.Name,
		suffix,
		subtleStyle.Render(description),
	)
}

func matchesQuery(tool catalog.Tool, query string) bool {
	values := []string{tool.ID, tool.Name, tool.Category, humanCategory(tool.Category), tool.Description}
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}

	return false
}

func humanCategory(category string) string {
	if label, ok := categoryLabels[category]; ok {
		return label
	}

	return strings.ReplaceAll(category, "_", " ")
}

var categoryLabels = map[string]string{
	"cloud_gcp":       "Cloud",
	"database":        "Databases",
	"env_management":  "Environment",
	"filesystem":      "Filesystem",
	"forge":           "Source Control / Forge",
	"grpc_api":        "HTTP / APIs",
	"http_api":        "HTTP / APIs",
	"iac":             "Infrastructure as Code",
	"json":            "Structured Data",
	"kubernetes":      "Kubernetes",
	"linting":         "Linting",
	"python_runtime":  "Runtime Management",
	"runtime_manager": "Runtime Management",
	"search":          "Search",
	"task_runner":     "Task Running",
	"text_processing": "Text Processing",
	"yaml":            "Structured Data",
}

func (m PickerModel) windowBounds() (int, int) {
	if m.height <= 0 || len(m.rows) == 0 {
		return 0, len(m.rows)
	}

	availableLines := m.height - 4
	if m.searching {
		availableLines -= 2
	}
	if availableLines < 3 {
		availableLines = 3
	}

	start := 0
	for start < m.cursor && m.rangeLineCount(start, m.cursor+1) > availableLines {
		start++
	}

	end := m.cursor + 1
	for end < len(m.rows) && m.rangeLineCount(start, end+1) <= availableLines {
		end++
	}

	for start > 0 && m.rangeLineCount(start-1, end) <= availableLines {
		start--
	}

	return start, end
}

func (m PickerModel) rangeLineCount(start, end int) int {
	lines := 0
	lastCategory := ""

	for index := start; index < end; index++ {
		row := m.rows[index]
		category := displayedCategory(row)
		if row.kind == rowKindTool && category != lastCategory {
			if lastCategory != "" {
				lines++
			}

			lines++
			lastCategory = category
		}

		lines += rowLineCount(row)
	}

	return lines
}

func renderPrefixedRow(prefix, row string) string {
	lines := strings.Split(row, "\n")
	for index, line := range lines {
		if index == 0 {
			lines[index] = prefix + line

			continue
		}

		lines[index] = "  " + line
	}

	return strings.Join(lines, "\n") + "\n"
}

func rowLineCount(row row) int {
	if row.kind == rowKindMore || row.tool.Description == "" {
		return 1
	}

	return 2
}

func displayedCategory(row row) string {
	if row.kind != rowKindTool {
		return ""
	}

	return humanCategory(row.tool.Category)
}
