package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
			Foreground(lipgloss.Color("196"))

	appNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("99")).
			Bold(true)

	versionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("248"))

	appliedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	staleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	updateStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	breadcrumbStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))

	breadcrumbCurrentStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("141")).
				Bold(true)

	breadcrumbSepStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("242"))
)

// breadcrumb builds a path-like header showing where you are in the TUI.
// Pass segments from root → current. The last segment is highlighted.
func breadcrumb(segments ...string) string {
	if len(segments) == 0 {
		return ""
	}
	sep := breadcrumbSepStyle.Render("  ›  ")
	out := ""
	for i, s := range segments {
		if i == len(segments)-1 {
			out += breadcrumbCurrentStyle.Render(s)
		} else {
			out += breadcrumbStyle.Render(s)
		}
		if i < len(segments)-1 {
			out += sep
		}
	}
	return out
}

// topBar renders a full-width header with `left` aligned to the left and
// `right` aligned to the right, padded to `width` columns. Both sides are
// pre-styled strings; width is the target column count.
func topBar(width int, left, right string) string {
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	gap := width - leftW - rightW
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}
