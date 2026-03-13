package tui

import (
	"fmt"
	"strings"

	"github.com/ametel01/agents-toolbelt/internal/pkgmgr"
	tea "github.com/charmbracelet/bubbletea"
)

// DependencyModel is the Bubble Tea model for dependency bootstrap selection.
type DependencyModel struct {
	cancelled    bool
	cursor       int
	dependencies []pkgmgr.DependencyPlanItem
	selected     map[string]bool
}

// RunDependencyPicker launches the interactive dependency picker and returns the selected dependencies.
func RunDependencyPicker(dependencies []pkgmgr.DependencyPlanItem) ([]pkgmgr.DependencyPlanItem, error) {
	model := NewDependencyModel(dependencies)
	finalModel, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
	if err != nil {
		return nil, fmt.Errorf("run dependency picker: %w", err)
	}

	return finalModel.(DependencyModel).SelectedDependencies(), nil
}

// NewDependencyModel constructs a dependency model with all dependencies preselected.
func NewDependencyModel(dependencies []pkgmgr.DependencyPlanItem) DependencyModel {
	selected := make(map[string]bool, len(dependencies))
	for _, dependency := range dependencies {
		selected[dependency.Name] = true
	}

	return DependencyModel{
		dependencies: dependencies,
		selected:     selected,
	}
}

// Init satisfies the Bubble Tea model interface.
func (m DependencyModel) Init() tea.Cmd {
	return nil
}

// Update satisfies the Bubble Tea model interface.
func (m DependencyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(typed.String())
	default:
		return m, nil
	}
}

// View renders the dependency picker state.
func (m DependencyModel) View() string {
	var builder strings.Builder

	builder.WriteString(categoryStyle.Render("Install missing dependencies"))
	builder.WriteString("\n")
	builder.WriteString(subtleStyle.Render("Choose which install-manager dependencies to bootstrap before installing the selected tools."))
	builder.WriteString("\n\n")

	for index, dependency := range m.dependencies {
		prefix := "  "
		if index == m.cursor {
			prefix = cursorStyle.Render("❯ ")
		}

		checkbox := subtleStyle.Render("◯")
		if m.selected[dependency.Name] {
			checkbox = selectedStyle.Render("◉")
		}

		managerLine := dependency.Name
		if dependency.Method.Package != "" && dependency.Method.Package != dependency.Name {
			managerLine = fmt.Sprintf("%s (%s package)", dependency.Name, dependency.Method.Package)
		}

		requiredBy := make([]string, 0, len(dependency.RequiredBy))
		for _, tool := range dependency.RequiredBy {
			requiredBy = append(requiredBy, tool.Name)
		}

		builder.WriteString(prefix + fmt.Sprintf("%s %s via %s\n", checkbox, managerLine, dependency.Manager.Name()))
		builder.WriteString("      " + subtleStyle.Render("Required by: "+strings.Join(requiredBy, ", ")) + "\n")
	}

	builder.WriteString("\n")
	builder.WriteString(subtleStyle.Render("Leave items unselected to continue without bootstrapping them."))
	builder.WriteString("\n\n")
	builder.WriteString(m.helpBar())

	return builder.String()
}

// SelectedDependencies returns the current selected dependency list.
// Returns nil if the user cancelled the picker.
func (m DependencyModel) SelectedDependencies() []pkgmgr.DependencyPlanItem {
	if m.cancelled {
		return nil
	}

	selected := make([]pkgmgr.DependencyPlanItem, 0, len(m.dependencies))
	for _, dependency := range m.dependencies {
		if m.selected[dependency.Name] {
			selected = append(selected, dependency)
		}
	}

	return selected
}

func (m DependencyModel) handleKey(key string) (tea.Model, tea.Cmd) {
	if handled := m.handleCursorKey(key); handled {
		return m, nil
	}

	if handled := m.handleSelectionKey(key); handled {
		return m, nil
	}

	if handled, nextModel, cmd := m.handleConfirmOrQuit(key); handled {
		return nextModel, cmd
	}

	return m, nil
}

func (m *DependencyModel) handleCursorKey(key string) bool {
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

		return true
	case "down", "j":
		if m.cursor < len(m.dependencies)-1 {
			m.cursor++
		}

		return true
	default:
		return false
	}
}

func (m *DependencyModel) handleSelectionKey(key string) bool {
	switch key {
	case " ", "space":
		if m.cursor < len(m.dependencies) {
			name := m.dependencies[m.cursor].Name
			m.selected[name] = !m.selected[name]
		}

		return true
	case "a":
		for _, dependency := range m.dependencies {
			m.selected[dependency.Name] = true
		}

		return true
	case "n":
		for _, dependency := range m.dependencies {
			m.selected[dependency.Name] = false
		}

		return true
	default:
		return false
	}
}

func (m DependencyModel) handleConfirmOrQuit(key string) (bool, tea.Model, tea.Cmd) {
	switch key {
	case "enter":
		return true, m, tea.Quit
	case "q", "esc":
		m.cancelled = true

		return true, m, tea.Quit
	default:
		return false, m, nil
	}
}

func (m DependencyModel) helpBar() string {
	sep := helpSepStyle.Render(" • ")

	return helpKeyStyle.Render("space") + subtleStyle.Render(" toggle") + sep +
		helpKeyStyle.Render("a") + subtleStyle.Render(" all") + sep +
		helpKeyStyle.Render("n") + subtleStyle.Render(" none") + sep +
		helpKeyStyle.Render("↵") + subtleStyle.Render(" confirm") + sep +
		helpKeyStyle.Render("q") + subtleStyle.Render(" skip")
}
