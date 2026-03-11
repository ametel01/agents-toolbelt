package tui

import (
	"fmt"
	"strings"

	"github.com/ametel01/agents-toolbelt/internal/skill"
	tea "github.com/charmbracelet/bubbletea"
)

// TargetModel is the Bubble Tea model for skill target selection.
type TargetModel struct {
	cancelled bool
	cursor    int
	selected  map[string]bool
	targets   []skill.Target
	width     int
	height    int
}

// RunTargetPicker launches the interactive target picker and returns the selected targets.
func RunTargetPicker(targets []skill.Target) ([]skill.Target, error) {
	model := NewTargetModel(targets)
	finalModel, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
	if err != nil {
		return nil, fmt.Errorf("run target picker: %w", err)
	}

	return finalModel.(TargetModel).SelectedTargets(), nil
}

// NewTargetModel constructs a target model with all targets preselected.
func NewTargetModel(targets []skill.Target) TargetModel {
	selected := make(map[string]bool, len(targets))
	for _, target := range targets {
		selected[target.ID] = true
	}

	return TargetModel{
		selected: selected,
		targets:  targets,
	}
}

// Init satisfies the Bubble Tea model interface.
func (m TargetModel) Init() tea.Cmd {
	return nil
}

// Update satisfies the Bubble Tea model interface.
func (m TargetModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

// View renders the target picker state.
func (m TargetModel) View() string {
	var builder strings.Builder

	builder.WriteString(categoryStyle.Render("Select skill targets"))
	builder.WriteString("\n")
	builder.WriteString(subtleStyle.Render("Choose which AI agents receive the cli-tools skill file."))
	builder.WriteString("\n\n")

	for index, target := range m.targets {
		prefix := "  "
		if index == m.cursor {
			prefix = cursorStyle.Render("❯ ")
		}

		checkbox := subtleStyle.Render("◯")
		if m.selected[target.ID] {
			checkbox = selectedStyle.Render("◉")
		}

		line := fmt.Sprintf("%s %s", checkbox, target.Name)
		path := subtleStyle.Render("  ~/" + target.RelPath)
		builder.WriteString(prefix + line + "\n")
		builder.WriteString("      " + path + "\n")
	}

	if m.noneSelected() {
		builder.WriteString("\n")
		builder.WriteString(subtleStyle.Render("  Select at least one target to continue."))
	}

	builder.WriteString("\n\n")
	builder.WriteString(m.helpBar())

	return builder.String()
}

// SelectedTargets returns the current selected target list.
// Returns nil if the user cancelled the picker.
func (m TargetModel) SelectedTargets() []skill.Target {
	if m.cancelled {
		return nil
	}

	selected := make([]skill.Target, 0, len(m.selected))
	for _, target := range m.targets {
		if m.selected[target.ID] {
			selected = append(selected, target)
		}
	}

	return selected
}

func (m TargetModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if handled := m.handleCursorKey(msg.String()); handled {
		return m, nil
	}

	if handled := m.handleSelectionKey(msg.String()); handled {
		return m, nil
	}

	return m.handleConfirmOrQuit(msg.String())
}

func (m *TargetModel) handleCursorKey(key string) bool {
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

		return true
	case "down", "j":
		if m.cursor < len(m.targets)-1 {
			m.cursor++
		}

		return true
	default:
		return false
	}
}

func (m *TargetModel) handleSelectionKey(key string) bool {
	switch key {
	case " ", "space":
		if m.cursor < len(m.targets) {
			id := m.targets[m.cursor].ID
			m.selected[id] = !m.selected[id]
		}

		return true
	case "a":
		for _, target := range m.targets {
			m.selected[target.ID] = true
		}

		return true
	default:
		return false
	}
}

func (m TargetModel) handleConfirmOrQuit(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "enter":
		if m.noneSelected() {
			return m, nil
		}

		return m, tea.Quit
	case "q", "esc":
		m.cancelled = true

		return m, tea.Quit
	default:
		return m, nil
	}
}

func (m TargetModel) noneSelected() bool {
	for _, v := range m.selected {
		if v {
			return false
		}
	}

	return true
}

func (m TargetModel) helpBar() string {
	sep := helpSepStyle.Render(" • ")

	return helpKeyStyle.Render("space") + subtleStyle.Render(" toggle") + sep +
		helpKeyStyle.Render("a") + subtleStyle.Render(" all") + sep +
		helpKeyStyle.Render("↵") + subtleStyle.Render(" confirm") + sep +
		helpKeyStyle.Render("q") + subtleStyle.Render(" quit")
}
