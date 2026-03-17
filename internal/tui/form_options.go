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
	km.Select.Next = key.NewBinding(key.WithKeys("enter", "tab"), key.WithHelp("enter", "next"))
	km.Select.Prev = key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "back"))
	return km
}

// Unified options form data for both workspace and project levels.
// Workspace uses "enabled"/"disabled"; project adds "inherit".
type optionsFormData struct {
	Worktree           string // "inherit", "enabled", "disabled"
	DisableAttribution string // "inherit", "enabled", "disabled"
}

// wsDefaults holds the workspace-level defaults shown in "Inherit" labels.
type wsDefaults struct {
	Worktree           bool
	DisableAttribution bool
}

func selectOptions(label string, allowInherit bool, wsDefault *bool) []huh.Option[string] {
	var opts []huh.Option[string]
	if allowInherit {
		inheritLabel := label
		if wsDefault != nil {
			if *wsDefault {
				inheritLabel += " (currently: enabled)"
			} else {
				inheritLabel += " (currently: disabled)"
			}
		}
		opts = append(opts, huh.NewOption(inheritLabel, "inherit"))
	}
	opts = append(opts,
		huh.NewOption("Enabled", "enabled"),
		huh.NewOption("Disabled", "disabled"),
	)
	return opts
}

func newOptionsForm(data *optionsFormData, orig optionsFormData, allowInherit bool, defaults *wsDefaults) *huh.Form {
	var worktreeDefault, attrDefault *bool
	if defaults != nil {
		worktreeDefault = &defaults.Worktree
		attrDefault = &defaults.DisableAttribution
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				TitleFunc(func() string {
					return dirtyTitle("Disable commit/PR attributions?", data.DisableAttribution != orig.DisableAttribution)
				}, &data.DisableAttribution).
				Description("Removes \"Made with Claude Code\" from commits and PRs").
				Options(selectOptions("Inherit from workspace", allowInherit, attrDefault)...).
				Value(&data.DisableAttribution),
			huh.NewSelect[string]().
				TitleFunc(func() string {
					return dirtyTitle("Always use worktree?", data.Worktree != orig.Worktree)
				}, &data.Worktree).
				Description("Runs claude --worktree by default (bypass with --no-worktree)").
				Options(selectOptions("Inherit from workspace", allowInherit, worktreeDefault)...).
				Value(&data.Worktree),
		),
	).WithKeyMap(formKeyMap())
}
