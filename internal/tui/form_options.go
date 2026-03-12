package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// confirmLeft creates a left-aligned confirm field.
func confirmLeft() *huh.Confirm {
	return huh.NewConfirm().WithButtonAlignment(lipgloss.Left)
}

var modifiedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

func dirtyTitle(title string, dirty bool) string {
	if dirty {
		return modifiedStyle.Render(title + " (modified)")
	}
	return title
}

// formKeyMap returns a keymap with up/down arrow support for navigating between fields.
func formKeyMap() *huh.KeyMap {
	km := huh.NewDefaultKeyMap()
	km.Confirm.Next = key.NewBinding(key.WithKeys("enter", "tab", "down"), key.WithHelp("↓/enter", "next"))
	km.Confirm.Prev = key.NewBinding(key.WithKeys("shift+tab", "up"), key.WithHelp("↑/shift+tab", "back"))
	km.Input.Next = key.NewBinding(key.WithKeys("enter", "tab", "down"), key.WithHelp("↓/enter", "next"))
	km.Input.Prev = key.NewBinding(key.WithKeys("shift+tab", "up"), key.WithHelp("↑/shift+tab", "back"))
	km.Note.Next = key.NewBinding(key.WithKeys("enter", "tab", "down"), key.WithHelp("↓/enter", "next"))
	km.Note.Prev = key.NewBinding(key.WithKeys("shift+tab", "up"), key.WithHelp("↑/shift+tab", "back"))
	return km
}

type optionsFormData struct {
	DisableAttribution bool
	AlwaysWorktree     bool
}

func newOptionsForm(data *optionsFormData, orig optionsFormData) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			confirmLeft().
				TitleFunc(func() string {
					return dirtyTitle("Disable commit/PR attributions?", data.DisableAttribution != orig.DisableAttribution)
				}, &data.DisableAttribution).
				Description("Removes \"Made with Claude Code\" from commits and PRs").
				Affirmative("Enabled").
				Negative("Disabled").
				Value(&data.DisableAttribution),
			confirmLeft().
				TitleFunc(func() string {
					return dirtyTitle("Always use worktree?", data.AlwaysWorktree != orig.AlwaysWorktree)
				}, &data.AlwaysWorktree).
				Description("Runs claude --worktree by default (bypass with --no-worktree)").
				Affirmative("Enabled").
				Negative("Disabled").
				Value(&data.AlwaysWorktree),
		),
	).WithKeyMap(formKeyMap())
}
