// Package tui provides the interactive picker used during installation.
package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	cursorStyle   = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	subtleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)
