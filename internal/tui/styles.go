package tui

import "github.com/charmbracelet/lipgloss"

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99"))

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	authStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	unauthStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)
