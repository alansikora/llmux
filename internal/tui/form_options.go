package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/huh"
)

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

func newOptionsForm(data *optionsFormData) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Disable commit/PR attributions?").
				Description("Removes \"Made with Claude Code\" from commits and PRs").
				Affirmative("Yes").
				Negative("No").
				Value(&data.DisableAttribution),
			huh.NewConfirm().
				Title("Always use worktree?").
				Description("Runs claude --worktree by default (bypass with --no-worktree)").
				Affirmative("Yes").
				Negative("No").
				Value(&data.AlwaysWorktree),
		),
	).WithKeyMap(formKeyMap())
}
