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

	logoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("99")).
			Bold(true)

	versionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	appliedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))
)
