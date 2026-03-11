// Package tui provides the interactive picker used during installation.
package tui

import "github.com/charmbracelet/lipgloss"

var (
	bannerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	subtitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
	categoryStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("5"))
	cursorStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	subtleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	helpKeyStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	helpSepStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	counterStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	searchStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
)
